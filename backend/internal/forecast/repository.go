package forecast

import (
	"context"
	"sort"
	"strings"
	"time"

	"isbudget/backend/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListIncomeDailyByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualIncomeDaily, error)
	ListActualExpensesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualExpense, error)
	GetProjectByID(ctx context.Context, projectID uuid.UUID) (sqlc.GetProjectByIDRow, error)
	GetProjectOwnerAndCategoryAndDuration(ctx context.Context, projectID uuid.UUID) (uuid.UUID, string, int32, time.Time, error)
	StoreRunWithPoints(ctx context.Context, run sqlc.CreateForecastRunParams, points []sqlc.CreateForecastPointParams) (sqlc.ForecastRun, []sqlc.ForecastPoint, error)
	ListForecastRunsByProject(ctx context.Context, projectID uuid.UUID) ([]RunSummary, error)
	GetForecastRunByID(ctx context.Context, runID uuid.UUID) (sqlc.ForecastRun, error)
	GetLatestForecastRunByProject(ctx context.Context, projectID uuid.UUID) (sqlc.ForecastRun, error)
	ListForecastPointsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ForecastPoint, error)
	DeleteForecastRun(ctx context.Context, projectID, runID uuid.UUID) error
	ListHistoricalCampaignsByOwner(ctx context.Context, ownerUserID uuid.UUID, category string) ([]HistoricalCampaign, error)
	ListCompletedProjectHistoricalCampaignsByOwnerAndCategory(ctx context.Context, ownerUserID uuid.UUID, category string, excludeProjectID uuid.UUID) ([]HistoricalCampaign, error)
	CreateHistoricalCampaign(ctx context.Context, ownerUserID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error)
	UpdateHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error)
	DeleteHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID) error
}

type PGRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewPGRepository(q *sqlc.Queries, pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{q: q, pool: pool}
}

func (r *PGRepository) ListIncomeDailyByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualIncomeDaily, error) {
	rows, err := r.q.ListIncomeDailyByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ActualIncomeDaily, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ActualIncomeDaily(row))
	}
	return items, nil
}

func (r *PGRepository) ListActualExpensesByProject(ctx context.Context, projectID uuid.UUID) ([]sqlc.ActualExpense, error) {
	rows, err := r.q.ListActualExpensesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ActualExpense, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ActualExpense(row))
	}
	return items, nil
}

func (r *PGRepository) GetProjectByID(ctx context.Context, projectID uuid.UUID) (sqlc.GetProjectByIDRow, error) {
	return r.q.GetProjectByID(ctx, projectID)
}

func (r *PGRepository) GetProjectOwnerAndCategoryAndDuration(ctx context.Context, projectID uuid.UUID) (uuid.UUID, string, int32, time.Time, error) {
	var ownerUserID uuid.UUID
	var category string
	var durationDays int32
	var campaignStartDate time.Time
	err := r.pool.QueryRow(ctx, `
SELECT created_by,
       category,
       duration_days,
       COALESCE((activated_at AT TIME ZONE 'UTC')::date, (created_at AT TIME ZONE 'UTC')::date)
FROM projects
WHERE id = $1
`, projectID).Scan(&ownerUserID, &category, &durationDays, &campaignStartDate)
	if err != nil {
		return uuid.Nil, "", 0, time.Time{}, err
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = "Общее"
	}
	return ownerUserID, category, durationDays, campaignStartDate.UTC(), nil
}

func (r *PGRepository) StoreRunWithPoints(ctx context.Context, run sqlc.CreateForecastRunParams, points []sqlc.CreateForecastPointParams) (sqlc.ForecastRun, []sqlc.ForecastPoint, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlc.ForecastRun{}, nil, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)
	createdRun, err := qtx.CreateForecastRun(ctx, run)
	if err != nil {
		return sqlc.ForecastRun{}, nil, err
	}

	createdPoints := make([]sqlc.ForecastPoint, 0, len(points))
	for _, point := range points {
		point.RunID = createdRun.ID
		created, err := qtx.CreateForecastPoint(ctx, point)
		if err != nil {
			return sqlc.ForecastRun{}, nil, err
		}
		createdPoints = append(createdPoints, sqlc.ForecastPoint(created))
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.ForecastRun{}, nil, err
	}
	return createdRun, createdPoints, nil
}

func (r *PGRepository) GetForecastRunByID(ctx context.Context, runID uuid.UUID) (sqlc.ForecastRun, error) {
	return r.q.GetForecastRunByID(ctx, runID)
}

func (r *PGRepository) ListForecastRunsByProject(ctx context.Context, projectID uuid.UUID) ([]RunSummary, error) {
	rows, err := r.pool.Query(ctx, `
SELECT r.id,
       r.project_id,
       r.created_at,
       r.horizon_days,
       r.model_version,
       r.input_hash,
       r.seed,
       COALESCE(SUM(p.yhat), 0)::double precision AS forecast_total_yhat,
       COALESCE(SUM(p.p10), 0)::double precision AS forecast_total_p10,
       COALESCE(SUM(p.p90), 0)::double precision AS forecast_total_p90
FROM forecast_runs r
LEFT JOIN forecast_points p
  ON p.run_id = r.id
WHERE r.project_id = $1
GROUP BY r.id, r.project_id, r.created_at, r.horizon_days, r.model_version, r.input_hash, r.seed
ORDER BY r.created_at DESC
`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RunSummary, 0)
	for rows.Next() {
		var item RunSummary
		if err := rows.Scan(
			&item.Run.ID,
			&item.Run.ProjectID,
			&item.Run.CreatedAt,
			&item.Run.HorizonDays,
			&item.Run.ModelVersion,
			&item.Run.InputHash,
			&item.Run.Seed,
			&item.ForecastTotalYhat,
			&item.ForecastTotalP10,
			&item.ForecastTotalP90,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PGRepository) GetLatestForecastRunByProject(ctx context.Context, projectID uuid.UUID) (sqlc.ForecastRun, error) {
	return r.q.GetLatestForecastRunByProject(ctx, projectID)
}

func (r *PGRepository) ListForecastPointsByRun(ctx context.Context, runID uuid.UUID) ([]sqlc.ForecastPoint, error) {
	rows, err := r.q.ListForecastPointsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	items := make([]sqlc.ForecastPoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, sqlc.ForecastPoint(row))
	}
	return items, nil
}

func (r *PGRepository) DeleteForecastRun(ctx context.Context, projectID, runID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
DELETE FROM forecast_runs
WHERE id = $1
  AND project_id = $2
`, runID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PGRepository) ListHistoricalCampaignsByOwner(ctx context.Context, ownerUserID uuid.UUID, category string) ([]HistoricalCampaign, error) {
	category = strings.TrimSpace(category)
	if category == "" {
		return r.listManualHistoricalCampaigns(ctx, ownerUserID, nil)
	}
	return r.listManualHistoricalCampaigns(ctx, ownerUserID, &category)
}

func (r *PGRepository) listManualHistoricalCampaigns(ctx context.Context, ownerUserID uuid.UUID, category *string) ([]HistoricalCampaign, error) {
	base := `
SELECT id,
       project_id,
       owner_user_id,
       category,
       name,
       duration_days,
       currency_code,
       TO_CHAR(started_at, 'YYYY-MM-DD') AS started_at,
       TO_CHAR(ended_at, 'YYYY-MM-DD') AS ended_at,
       created_at,
       updated_at
FROM historical_campaigns
WHERE owner_user_id = $1
`
	args := []any{ownerUserID}
	if category != nil {
		base += "  AND lower(category) = lower($2)\n"
		args = append(args, *category)
	}
	base += "ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := make([]HistoricalCampaign, 0)
	for rows.Next() {
		var item HistoricalCampaign
		if err := rows.Scan(
			&item.ID,
			&item.ProjectID,
			&item.OwnerUserID,
			&item.Category,
			&item.Name,
			&item.DurationDays,
			&item.CurrencyCode,
			&item.StartedAt,
			&item.EndedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Source = "manual"
		campaigns = append(campaigns, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range campaigns {
		days, err := r.listHistoricalCampaignDays(ctx, campaigns[i].ID)
		if err != nil {
			return nil, err
		}
		campaigns[i].Days = days
	}
	return campaigns, nil
}

func (r *PGRepository) ListCompletedProjectHistoricalCampaignsByOwnerAndCategory(ctx context.Context, ownerUserID uuid.UUID, category string, excludeProjectID uuid.UUID) ([]HistoricalCampaign, error) {
	category = strings.TrimSpace(category)
	rows, err := r.pool.Query(ctx, `
SELECT id, name, duration_days, currency_code, activated_at, created_at, updated_at
FROM projects
WHERE created_by = $1
  AND ($2 = '' OR lower(category) = lower($2))
  AND status = 'completed'
  AND ($3 = '00000000-0000-0000-0000-000000000000'::uuid OR id <> $3)
ORDER BY COALESCE(activated_at, created_at) DESC
`, ownerUserID, category, excludeProjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type completedProject struct {
		ID           uuid.UUID
		Name         string
		DurationDays int32
		CurrencyCode string
		ActivatedAt  *time.Time
		CreatedAt    time.Time
		UpdatedAt    time.Time
	}
	projects := make([]completedProject, 0)
	for rows.Next() {
		var item completedProject
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.DurationDays,
			&item.CurrencyCode,
			&item.ActivatedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		projects = append(projects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]HistoricalCampaign, 0, len(projects))
	for _, project := range projects {
		start := project.CreatedAt.UTC()
		if project.ActivatedAt != nil {
			start = project.ActivatedAt.UTC()
		}
		startDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
		duration := int(project.DurationDays)
		if duration < 1 {
			continue
		}

		days := make([]HistoricalCampaignDay, duration)
		for i := 0; i < duration; i++ {
			days[i] = HistoricalCampaignDay{
				DayNumber:    int32(i + 1),
				IncomeAmount: 0,
				Expenses:     []HistoricalCampaignExpense{},
			}
		}

		incomeRows, err := r.pool.Query(ctx, `
SELECT income_date, amount::double precision AS amount
FROM actual_income_daily
WHERE project_id = $1
`, project.ID)
		if err != nil {
			return nil, err
		}
		for incomeRows.Next() {
			var incomeDate time.Time
			var amount float64
			if err := incomeRows.Scan(&incomeDate, &amount); err != nil {
				incomeRows.Close()
				return nil, err
			}
			dayNumber := int(incomeDate.UTC().Sub(startDate).Hours()/24) + 1
			if dayNumber < 1 || dayNumber > duration {
				continue
			}
			days[dayNumber-1].IncomeAmount += amount
		}
		if err := incomeRows.Err(); err != nil {
			incomeRows.Close()
			return nil, err
		}
		incomeRows.Close()

		expenseRows, err := r.pool.Query(ctx, `
SELECT paid_at, category, amount::double precision AS amount
FROM actual_expenses
WHERE project_id = $1
ORDER BY paid_at ASC, category ASC
`, project.ID)
		if err != nil {
			return nil, err
		}

		dayExpenseMap := make(map[int]map[string]float64)
		for expenseRows.Next() {
			var paidAt time.Time
			var expenseCategory string
			var amount float64
			if err := expenseRows.Scan(&paidAt, &expenseCategory, &amount); err != nil {
				expenseRows.Close()
				return nil, err
			}
			dayNumber := int(paidAt.UTC().Sub(startDate).Hours()/24) + 1
			if dayNumber < 1 || dayNumber > duration {
				continue
			}
			if dayExpenseMap[dayNumber] == nil {
				dayExpenseMap[dayNumber] = make(map[string]float64)
			}
			key := strings.ToLower(strings.TrimSpace(expenseCategory))
			dayExpenseMap[dayNumber][key] += amount
		}
		if err := expenseRows.Err(); err != nil {
			expenseRows.Close()
			return nil, err
		}
		expenseRows.Close()

		for dayNumber, catMap := range dayExpenseMap {
			keys := make([]string, 0, len(catMap))
			for key := range catMap {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			expenses := make([]HistoricalCampaignExpense, 0, len(keys))
			for _, key := range keys {
				expenses = append(expenses, HistoricalCampaignExpense{
					Category: key,
					Amount:   catMap[key],
				})
			}
			days[dayNumber-1].Expenses = expenses
		}

		projectID := project.ID
		campaign := HistoricalCampaign{
			ID:           project.ID,
			ProjectID:    &projectID,
			OwnerUserID:  ownerUserID,
			Category:     category,
			Source:       "completed_project",
			Name:         project.Name,
			DurationDays: project.DurationDays,
			CurrencyCode: project.CurrencyCode,
			StartedAt:    startDate.Format("2006-01-02"),
			EndedAt:      startDate.AddDate(0, 0, duration-1).Format("2006-01-02"),
			CreatedAt:    project.CreatedAt,
			UpdatedAt:    project.UpdatedAt,
			Days:         days,
		}
		out = append(out, campaign)
	}
	return out, nil
}

func (r *PGRepository) CreateHistoricalCampaign(ctx context.Context, ownerUserID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	defer tx.Rollback(ctx)

	var campaign HistoricalCampaign
	err = tx.QueryRow(ctx, `
INSERT INTO historical_campaigns (id, project_id, owner_user_id, category, name, duration_days, currency_code, started_at, ended_at)
VALUES (gen_random_uuid(), NULL, $1, $2, $3, $4, $5, $6::date, $7::date)
RETURNING id,
          project_id,
          owner_user_id,
          category,
          name,
          duration_days,
          currency_code,
          TO_CHAR(started_at, 'YYYY-MM-DD') AS started_at,
          TO_CHAR(ended_at, 'YYYY-MM-DD') AS ended_at,
          created_at,
          updated_at
`, ownerUserID, req.Category, req.Name, req.DurationDays, req.CurrencyCode, req.StartedAt, req.EndedAt).Scan(
		&campaign.ID,
		&campaign.ProjectID,
		&campaign.OwnerUserID,
		&campaign.Category,
		&campaign.Name,
		&campaign.DurationDays,
		&campaign.CurrencyCode,
		&campaign.StartedAt,
		&campaign.EndedAt,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	campaign.Source = "manual"

	if err := upsertHistoricalCampaignDays(ctx, tx, campaign.ID, req.Days); err != nil {
		return HistoricalCampaign{}, err
	}

	days, err := listHistoricalCampaignDaysTx(ctx, tx, campaign.ID)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	campaign.Days = days

	if err := tx.Commit(ctx); err != nil {
		return HistoricalCampaign{}, err
	}
	return campaign, nil
}

func (r *PGRepository) UpdateHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID, req UpsertHistoricalCampaignRequest) (HistoricalCampaign, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	defer tx.Rollback(ctx)

	var campaign HistoricalCampaign
	err = tx.QueryRow(ctx, `
UPDATE historical_campaigns
SET category = $3,
    name = $4,
    duration_days = $5,
    currency_code = $6,
    started_at = $7::date,
    ended_at = $8::date,
    updated_at = NOW()
WHERE id = $1
  AND owner_user_id = $2
  AND project_id IS NULL
RETURNING id,
          project_id,
          owner_user_id,
          category,
          name,
          duration_days,
          currency_code,
          TO_CHAR(started_at, 'YYYY-MM-DD') AS started_at,
          TO_CHAR(ended_at, 'YYYY-MM-DD') AS ended_at,
          created_at,
          updated_at
`, campaignID, ownerUserID, req.Category, req.Name, req.DurationDays, req.CurrencyCode, req.StartedAt, req.EndedAt).Scan(
		&campaign.ID,
		&campaign.ProjectID,
		&campaign.OwnerUserID,
		&campaign.Category,
		&campaign.Name,
		&campaign.DurationDays,
		&campaign.CurrencyCode,
		&campaign.StartedAt,
		&campaign.EndedAt,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	campaign.Source = "manual"

	if _, err := tx.Exec(ctx, `DELETE FROM historical_campaign_daily_metrics WHERE campaign_id = $1`, campaignID); err != nil {
		return HistoricalCampaign{}, err
	}
	if err := upsertHistoricalCampaignDays(ctx, tx, campaign.ID, req.Days); err != nil {
		return HistoricalCampaign{}, err
	}

	days, err := listHistoricalCampaignDaysTx(ctx, tx, campaign.ID)
	if err != nil {
		return HistoricalCampaign{}, err
	}
	campaign.Days = days

	if err := tx.Commit(ctx); err != nil {
		return HistoricalCampaign{}, err
	}
	return campaign, nil
}

func (r *PGRepository) DeleteHistoricalCampaign(ctx context.Context, ownerUserID, campaignID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
DELETE FROM historical_campaigns
WHERE id = $1
  AND owner_user_id = $2
  AND project_id IS NULL
`, campaignID, ownerUserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PGRepository) listHistoricalCampaignDays(ctx context.Context, campaignID uuid.UUID) ([]HistoricalCampaignDay, error) {
	return listHistoricalCampaignDaysTx(ctx, r.pool, campaignID)
}

type queryable interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func listHistoricalCampaignDaysTx(ctx context.Context, q queryable, campaignID uuid.UUID) ([]HistoricalCampaignDay, error) {
	rows, err := q.Query(ctx, `
SELECT m.id,
       m.day_number,
       m.income_amount::double precision AS income_amount,
       e.category,
       e.amount::double precision AS amount
FROM historical_campaign_daily_metrics m
LEFT JOIN historical_campaign_daily_expenses e
  ON e.metric_id = m.id
WHERE m.campaign_id = $1
ORDER BY m.day_number ASC, e.category ASC
`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	days := make([]HistoricalCampaignDay, 0)
	metricIndex := make(map[uuid.UUID]int)
	for rows.Next() {
		var metricID uuid.UUID
		var dayNumber int32
		var incomeAmount float64
		var category *string
		var amount *float64
		if err := rows.Scan(&metricID, &dayNumber, &incomeAmount, &category, &amount); err != nil {
			return nil, err
		}

		idx, exists := metricIndex[metricID]
		if !exists {
			days = append(days, HistoricalCampaignDay{
				DayNumber:    dayNumber,
				IncomeAmount: incomeAmount,
				Expenses:     make([]HistoricalCampaignExpense, 0),
			})
			idx = len(days) - 1
			metricIndex[metricID] = idx
		}

		if category != nil && amount != nil {
			days[idx].Expenses = append(days[idx].Expenses, HistoricalCampaignExpense{
				Category: *category,
				Amount:   *amount,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return days, nil
}

func upsertHistoricalCampaignDays(ctx context.Context, tx pgx.Tx, campaignID uuid.UUID, days []HistoricalCampaignDay) error {
	for _, day := range days {
		var metricID uuid.UUID
		err := tx.QueryRow(ctx, `
INSERT INTO historical_campaign_daily_metrics (id, campaign_id, day_number, income_amount)
VALUES (gen_random_uuid(), $1, $2, $3)
RETURNING id
`, campaignID, day.DayNumber, day.IncomeAmount).Scan(&metricID)
		if err != nil {
			return err
		}

		seenCategories := make(map[string]struct{}, len(day.Expenses))
		for _, expense := range day.Expenses {
			categoryKey := strings.ToLower(strings.TrimSpace(expense.Category))
			if categoryKey == "" {
				continue
			}
			if _, exists := seenCategories[categoryKey]; exists {
				continue
			}
			seenCategories[categoryKey] = struct{}{}
			if _, err := tx.Exec(ctx, `
INSERT INTO historical_campaign_daily_expenses (id, metric_id, category, amount)
VALUES (gen_random_uuid(), $1, $2, $3)
`, metricID, strings.TrimSpace(expense.Category), expense.Amount); err != nil {
				return err
			}
		}
	}
	return nil
}
