package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"time"

	"isbudget/backend/internal/auth"
	"isbudget/backend/internal/common"
	appconfig "isbudget/backend/internal/config"
	"isbudget/backend/internal/dashboard"
	"isbudget/backend/internal/db/sqlc"
	"isbudget/backend/internal/finance/actual_expenses"
	"isbudget/backend/internal/finance/income"
	"isbudget/backend/internal/finance/payout"
	"isbudget/backend/internal/finance/planned_expenses"
	"isbudget/backend/internal/finance/taxes"
	"isbudget/backend/internal/forecast"
	"isbudget/backend/internal/fx"
	"isbudget/backend/internal/projects"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if err := runMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create pgx pool")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}

	q := sqlc.New(pool)
	fxSvc := fx.NewService(q, cfg.FXAPIBaseURL)
	startCurrencyRatesWorker(fxSvc, cfg.FXUpdateEvery)
	startProjectStatusWorker(q)

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := auth.NewService(q, jwtManager)
	authHandler := auth.NewHandler(authSvc, cfg.AvatarsDir)
	authLimiter := common.NewIPRateLimiter(cfg.AuthRateLimitMax, cfg.AuthRateLimitWindow)
	accountMutationLimiter := common.NewIPRateLimiter(cfg.AccountMutationRateLimitMax, cfg.AccountMutationRateLimitWindow)
	inviteActionLimiter := common.NewIPRateLimiter(cfg.InviteActionRateLimitMax, cfg.InviteActionRateLimitWindow)

	projectsSvc := projects.NewService(q, pool)
	projectsHandler := projects.NewHandler(projectsSvc)

	plannedSvc := planned_expenses.NewService(q)
	plannedHandler := planned_expenses.NewHandler(plannedSvc)

	incomeSvc := income.NewService(q, pool)
	incomeHandler := income.NewHandler(incomeSvc)

	actualSvc := actual_expenses.NewService(q)
	actualHandler := actual_expenses.NewHandler(actualSvc)

	taxSvc := taxes.NewService(q, pool)
	taxHandler := taxes.NewHandler(taxSvc)

	payoutRepo := payout.NewPGRepository(q, pool)
	payoutSvc := payout.NewService(payoutRepo, cfg.SnapshotsDir)
	payoutHandler := payout.NewHandler(payoutSvc)

	forecastClient := forecast.NewHTTPAnalyticsClient(cfg.AnalyticsURL)
	forecastRepo := forecast.NewPGRepository(q, pool)
	forecastSvc := forecast.NewService(forecastRepo, forecastClient)
	forecastHandler := forecast.NewHandler(forecastSvc)

	dashboardSvc := dashboard.NewService(q, payoutSvc)
	dashboardHandler := dashboard.NewHandler(dashboardSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.FrontendOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/swagger", serveSwaggerUI)
	r.Get("/swagger/", serveSwaggerUI)
	r.Get("/Swagger", serveSwaggerUI)
	r.Get("/Swagger/", serveSwaggerUI)
	r.Get("/openapi.yaml", serveOpenAPI)

	r.Route("/api", func(api chi.Router) {
		api.With(authLimiter.Middleware("auth_register")).Post("/auth/register", authHandler.Register)
		api.With(authLimiter.Middleware("auth_login")).Post("/auth/login", authHandler.Login)
		api.Get("/auth/avatar/{fileName}", authHandler.GetAvatar)
		api.Get("/openapi.yaml", serveOpenAPI)
		api.Get("/swagger", serveSwaggerUI)

		api.Group(func(private chi.Router) {
			private.Use(auth.Middleware(jwtManager))

			private.Get("/auth/me", authHandler.Me)
			private.Put("/auth/preferences", authHandler.UpdatePreferences)
			private.Put("/auth/profile", authHandler.UpdateProfile)
			private.With(accountMutationLimiter.Middleware("auth_email")).Put("/auth/email", authHandler.UpdateEmail)
			private.With(accountMutationLimiter.Middleware("auth_password")).Put("/auth/password", authHandler.UpdatePassword)
			private.Post("/auth/avatar", authHandler.UploadAvatar)
			private.Delete("/auth/avatar", authHandler.DeleteAvatar)

			private.Get("/projects", projectsHandler.ListProjects)
			private.Get("/projects/category-folders", projectsHandler.ListCategoryFolders)
			private.Post("/projects", projectsHandler.CreateProject)

			private.With(auth.RequireProjectPermission(q, auth.PermProjectRead)).Get("/projects/{projectId}", projectsHandler.GetProject)
			private.With(auth.RequireProjectPermission(q, auth.PermProjectWrite)).Put("/projects/{projectId}", projectsHandler.UpdateProject)
			private.With(auth.RequireProjectPermission(q, auth.PermProjectWrite)).Post("/projects/{projectId}/activate", projectsHandler.ActivateProject)
			private.With(auth.RequireProjectPermission(q, auth.PermProjectWrite)).Delete("/projects/{projectId}", projectsHandler.DeleteProject)
			private.With(auth.RequireProjectPermission(q, auth.PermProjectRead)).Post("/projects/{projectId}/leave", projectsHandler.LeaveProject)

			private.With(auth.RequireProjectPermission(q, auth.PermProjectRead)).Get("/projects/{projectId}/members", projectsHandler.ListMembers)
			private.With(auth.RequireProjectPermission(q, auth.PermMembersManage)).Post("/projects/{projectId}/members", projectsHandler.AddMember)
			private.With(auth.RequireProjectPermission(q, auth.PermMembersManage)).Delete("/projects/{projectId}/members/{memberId}", projectsHandler.DeleteMember)
			private.With(auth.RequireProjectPermission(q, auth.PermMembersManage)).Get("/projects/{projectId}/invites", projectsHandler.ListInvites)
			private.With(auth.RequireProjectPermission(q, auth.PermMembersManage)).Delete("/projects/{projectId}/invites/{inviteId}", projectsHandler.RevokeInvite)
			private.Get("/project-invites/incoming", projectsHandler.ListIncomingInvites)
			private.With(inviteActionLimiter.Middleware("invite_accept")).Post("/project-invites/{token}/accept", projectsHandler.AcceptInvite)
			private.With(inviteActionLimiter.Middleware("invite_decline")).Post("/project-invites/{token}/decline", projectsHandler.DeclineInvite)

			private.With(auth.RequireProjectPermission(q, auth.PermFinanceRead)).Get("/projects/{projectId}/planned-expenses", plannedHandler.List)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Post("/projects/{projectId}/planned-expenses", plannedHandler.Create)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Put("/projects/{projectId}/planned-expenses/{id}", plannedHandler.Update)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Delete("/projects/{projectId}/planned-expenses/{id}", plannedHandler.Delete)

			private.With(auth.RequireProjectPermission(q, auth.PermFinanceRead)).Get("/projects/{projectId}/income-daily", incomeHandler.List)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Post("/projects/{projectId}/income-daily", incomeHandler.Upsert)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Put("/projects/{projectId}/income-daily/cumulative", incomeHandler.UpsertCumulative)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Delete("/projects/{projectId}/income-daily/{date}", incomeHandler.Delete)

			private.With(auth.RequireProjectPermission(q, auth.PermFinanceRead)).Get("/projects/{projectId}/actual-expenses", actualHandler.List)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Post("/projects/{projectId}/actual-expenses", actualHandler.Create)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Put("/projects/{projectId}/actual-expenses/{id}", actualHandler.Update)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Delete("/projects/{projectId}/actual-expenses/{id}", actualHandler.Delete)

			private.With(auth.RequireProjectPermission(q, auth.PermFinanceRead)).Get("/projects/{projectId}/tax-rules", taxHandler.List)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Post("/projects/{projectId}/tax-rules", taxHandler.Create)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Put("/projects/{projectId}/tax-rules/reorder", taxHandler.Reorder)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Put("/projects/{projectId}/tax-rules/{id}", taxHandler.Update)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Delete("/projects/{projectId}/tax-rules/{id}", taxHandler.Delete)

			private.With(auth.RequireProjectPermission(q, auth.PermFinanceRead)).Get("/projects/{projectId}/profit-share-rules", payoutHandler.ListRules)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Post("/projects/{projectId}/profit-share-rules", payoutHandler.CreateRule)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Put("/projects/{projectId}/profit-share-rules/{id}", payoutHandler.UpdateRule)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Delete("/projects/{projectId}/profit-share-rules/{id}", payoutHandler.DeleteRule)

			private.With(auth.RequireProjectPermission(q, auth.PermFinanceRead)).Get("/projects/{projectId}/payout-preview", payoutHandler.PayoutPreview)
			private.With(auth.RequireProjectPermission(q, auth.PermProjectRead)).Get("/projects/{projectId}/my-income", payoutHandler.MyIncome)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceWrite)).Post("/projects/{projectId}/payouts/calculate", payoutHandler.CalculatePayouts)
			private.With(auth.RequireProjectPermission(q, auth.PermFinanceRead)).Get("/projects/{projectId}/payout-snapshots/{fileName}", payoutHandler.DownloadSnapshot)

			private.With(auth.RequireProjectPermission(q, auth.PermDashboardRead)).Get("/projects/{projectId}/dashboard", dashboardHandler.Get)

			private.With(auth.RequireProjectPermission(q, auth.PermForecastWrite)).Post("/projects/{projectId}/forecast", forecastHandler.Run)
			private.With(auth.RequireProjectPermission(q, auth.PermForecastRead)).Get("/projects/{projectId}/forecast/runs", forecastHandler.ListRuns)
			private.With(auth.RequireProjectPermission(q, auth.PermForecastRead)).Get("/projects/{projectId}/forecast/latest", forecastHandler.GetLatest)
			private.With(auth.RequireProjectPermission(q, auth.PermForecastRead)).Get("/projects/{projectId}/forecast/{runId}", forecastHandler.GetByID)
			private.With(auth.RequireProjectPermission(q, auth.PermForecastRead)).Get("/projects/{projectId}/forecast/{runId}/report.pdf", forecastHandler.DownloadRunReport)
			private.With(auth.RequireProjectPermission(q, auth.PermForecastWrite)).Delete("/projects/{projectId}/forecast/{runId}", forecastHandler.DeleteRun)

			private.Get("/historical-campaigns", forecastHandler.ListHistoricalCampaignsForUser)
			private.Post("/historical-campaigns", forecastHandler.CreateHistoricalCampaignForUser)
			private.Put("/historical-campaigns/{campaignId}", forecastHandler.UpdateHistoricalCampaignForUser)
			private.Delete("/historical-campaigns/{campaignId}", forecastHandler.DeleteHistoricalCampaignForUser)
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Info().Str("port", cfg.Port).Msg("backend server started")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("server failed")
	}
}

func runMigrations(databaseURL, migrationsPath string) error {
	uri := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(uri, databaseURL)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "openapi.yaml")
}

func serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	specBytes, err := os.ReadFile("openapi.yaml")
	if err != nil {
		http.Error(w, "failed to load local OpenAPI schema", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(buildOfflineAPIDocsHTML(string(specBytes))))
}

func buildOfflineAPIDocsHTML(openAPISpec string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>IS Budget API Docs</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      :root {
        color-scheme: light;
        font-family: "Segoe UI", system-ui, sans-serif;
        background: #f6f7fb;
        color: #152033;
      }
      body {
        margin: 0;
        padding: 32px 20px 48px;
        background:
          radial-gradient(circle at top left, rgba(35, 90, 180, 0.10), transparent 34%%),
          linear-gradient(180deg, #f8faff 0%%, #eef2f8 100%%);
      }
      .page {
        max-width: 1180px;
        margin: 0 auto;
      }
      .hero, .card {
        background: rgba(255, 255, 255, 0.92);
        border: 1px solid rgba(21, 32, 51, 0.10);
        border-radius: 18px;
        box-shadow: 0 14px 32px rgba(21, 32, 51, 0.08);
      }
      .hero {
        padding: 28px;
        margin-bottom: 18px;
      }
      .eyebrow {
        display: inline-block;
        margin-bottom: 8px;
        font-size: 12px;
        font-weight: 700;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: #2f5fb6;
      }
      h1 {
        margin: 0 0 10px;
        font-size: 32px;
        line-height: 1.15;
      }
      p {
        margin: 0;
        line-height: 1.55;
      }
      .muted {
        color: #4f5c73;
      }
      .actions {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        margin-top: 18px;
      }
      .button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        padding: 10px 14px;
        border-radius: 12px;
        border: 1px solid rgba(47, 95, 182, 0.18);
        background: #f2f6ff;
        color: #1d3f7a;
        font-weight: 600;
        text-decoration: none;
      }
      .button:hover {
        background: #e7eefc;
      }
      .grid {
        display: grid;
        gap: 18px;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      }
      .card {
        padding: 20px;
      }
      h2 {
        margin: 0 0 12px;
        font-size: 20px;
      }
      ul {
        margin: 0;
        padding-left: 18px;
        color: #344158;
      }
      li + li {
        margin-top: 8px;
      }
      code, pre {
        font-family: "Cascadia Code", "Fira Code", monospace;
      }
      pre {
        margin: 0;
        padding: 18px;
        overflow: auto;
        border-radius: 14px;
        background: #111827;
        color: #e5eefc;
        line-height: 1.5;
        font-size: 13px;
      }
    </style>
  </head>
  <body>
    <main class="page">
      <section class="hero">
        <span class="eyebrow">Offline API Docs</span>
        <h1>IS Budget Backend API</h1>
        <p class="muted">
          Эта страница обслуживается полностью локально из backend и не зависит от внешнего CDN.
          Источником истины остаётся актуальная схема OpenAPI.
        </p>
        <div class="actions">
          <a class="button" href="/api/openapi.yaml">Открыть /api/openapi.yaml</a>
          <a class="button" href="/openapi.yaml">Открыть /openapi.yaml</a>
        </div>
      </section>
      <section class="grid">
        <article class="card">
          <h2>Что здесь проверять в демо</h2>
          <ul>
            <li>доступность локальной документации без интернета;</li>
            <li>корректную отдачу raw schema по <code>/api/openapi.yaml</code>;</li>
            <li>наличие ключевых auth, finance, forecast и historical campaign endpoints;</li>
            <li>согласованность схемы с текущей реализацией backend.</li>
          </ul>
        </article>
        <article class="card">
          <h2>Подсказка</h2>
          <p class="muted">
            Для richer-visualization можно открыть тот же <code>openapi.yaml</code> во внешнем viewer,
            но для локальной демонстрации и проверки контракта этого экрана достаточно.
          </p>
        </article>
      </section>
      <section class="card" style="margin-top: 18px;">
        <h2>Локальная схема OpenAPI</h2>
        <pre>%s</pre>
      </section>
    </main>
  </body>
</html>`, html.EscapeString(openAPISpec))
}

func startProjectStatusWorker(q *sqlc.Queries) {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			if err := q.CompleteAllProjectsIfDue(context.Background()); err != nil {
				log.Error().Err(err).Msg("project status worker failed")
			}
		}
	}()
}

func startCurrencyRatesWorker(svc *fx.Service, interval time.Duration) {
	go func() {
		refresh := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if err := svc.RefreshRates(ctx); err != nil {
				log.Error().Err(err).Msg("currency rates refresh failed")
			}
		}

		refresh()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			refresh()
		}
	}()
}
