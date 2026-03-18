package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	devJWTSecret   = "dev-secret-change-me"
)

var ErrInsecureJWTSecret = errors.New("JWT_SECRET must be explicitly set in production")

type Config struct {
	Environment                    string
	Port                           string
	DatabaseURL                    string
	JWTSecret                      string
	JWTTTL                         time.Duration
	AnalyticsURL                   string
	MigrationsPath                 string
	FrontendOrigins                []string
	SnapshotsDir                   string
	AvatarsDir                     string
	FXAPIBaseURL                   string
	FXUpdateEvery                  time.Duration
	AuthRateLimitWindow            time.Duration
	AuthRateLimitMax               int
	AccountMutationRateLimitWindow time.Duration
	AccountMutationRateLimitMax    int
	InviteActionRateLimitWindow    time.Duration
	InviteActionRateLimitMax       int
}

func Load() (Config, error) {
	cfg := Config{
		Environment:                    envOrDefault("APP_ENV", EnvDevelopment),
		Port:                           envOrDefault("PORT", "8080"),
		DatabaseURL:                    envOrDefault("DATABASE_URL", "postgres://app:app@localhost:5432/is_budget?sslmode=disable"),
		JWTSecret:                      envOrDefault("JWT_SECRET", devJWTSecret),
		JWTTTL:                         durationEnvOrDefault("JWT_TTL", 24*time.Hour),
		AnalyticsURL:                   envOrDefault("ANALYTICS_URL", "http://localhost:8000"),
		MigrationsPath:                 envOrDefault("MIGRATIONS_PATH", filepath.Join("internal", "db", "migrations")),
		SnapshotsDir:                   envOrDefault("PAYOUT_SNAPSHOTS_DIR", filepath.Join("storage", "payout_snapshots")),
		AvatarsDir:                     envOrDefault("AVATARS_DIR", filepath.Join("storage", "avatars")),
		FXAPIBaseURL:                   envOrDefault("FX_API_BASE_URL", "https://open.er-api.com/v6"),
		FXUpdateEvery:                  durationEnvOrDefault("FX_UPDATE_INTERVAL", 6*time.Hour),
		AuthRateLimitWindow:            durationEnvOrDefault("AUTH_RATE_LIMIT_WINDOW", time.Minute),
		AuthRateLimitMax:               intEnvOrDefault("AUTH_RATE_LIMIT_MAX", 10),
		AccountMutationRateLimitWindow: durationEnvOrDefault("ACCOUNT_MUTATION_RATE_LIMIT_WINDOW", 5*time.Minute),
		AccountMutationRateLimitMax:    intEnvOrDefault("ACCOUNT_MUTATION_RATE_LIMIT_MAX", 10),
		InviteActionRateLimitWindow:    durationEnvOrDefault("INVITE_ACTION_RATE_LIMIT_WINDOW", time.Minute),
		InviteActionRateLimitMax:       intEnvOrDefault("INVITE_ACTION_RATE_LIMIT_MAX", 20),
	}

	origins := parseOrigins(os.Getenv("FRONTEND_ORIGINS"))
	if len(origins) == 0 {
		origin := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
		if origin != "" {
			origins = []string{origin}
		}
	}
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000"}
	}
	cfg.FrontendOrigins = origins

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Environment == EnvProduction && (strings.TrimSpace(c.JWTSecret) == "" || c.JWTSecret == devJWTSecret) {
		return ErrInsecureJWTSecret
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL must not be empty")
	}
	if strings.TrimSpace(c.Port) == "" {
		return errors.New("PORT must not be empty")
	}
	if err := validateHTTPBaseURL("ANALYTICS_URL", c.AnalyticsURL); err != nil {
		return err
	}
	if err := validateHTTPBaseURL("FX_API_BASE_URL", c.FXAPIBaseURL); err != nil {
		return err
	}
	if c.JWTTTL <= 0 {
		return errors.New("JWT_TTL must be a positive duration")
	}
	if c.AuthRateLimitWindow <= 0 || c.AccountMutationRateLimitWindow <= 0 || c.InviteActionRateLimitWindow <= 0 {
		return errors.New("rate limit windows must be positive durations")
	}
	if c.AuthRateLimitMax <= 0 || c.AccountMutationRateLimitMax <= 0 || c.InviteActionRateLimitMax <= 0 {
		return errors.New("rate limit max values must be positive integers")
	}
	for _, origin := range c.FrontendOrigins {
		if err := validateOrigin(origin); err != nil {
			return err
		}
	}
	return nil
}

func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		out = append(out, origin)
	}
	return out
}

func validateOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid FRONTEND origin %q: %w", origin, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid FRONTEND origin scheme %q", origin)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid FRONTEND origin host %q", origin)
	}
	if u.User != nil {
		return fmt.Errorf("FRONTEND origin must not contain credentials: %q", origin)
	}
	return nil
}

func validateHTTPBaseURL(name, raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s is invalid: %w", name, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", name)
	}
	if u.Host == "" {
		return fmt.Errorf("%s host must not be empty", name)
	}
	if u.User != nil {
		return fmt.Errorf("%s must not contain credentials", name)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func durationEnvOrDefault(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func intEnvOrDefault(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
