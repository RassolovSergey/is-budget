package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("FRONTEND_ORIGINS", "")
	t.Setenv("FRONTEND_ORIGIN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Environment != EnvDevelopment {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, EnvDevelopment)
	}
	if cfg.JWTSecret != devJWTSecret {
		t.Fatalf("JWTSecret = %q, want dev default", cfg.JWTSecret)
	}
	if len(cfg.FrontendOrigins) != 1 || cfg.FrontendOrigins[0] != "http://localhost:3000" {
		t.Fatalf("FrontendOrigins = %#v", cfg.FrontendOrigins)
	}
	if cfg.FXUpdateEvery != 6*time.Hour {
		t.Fatalf("FXUpdateEvery = %v", cfg.FXUpdateEvery)
	}
	if cfg.JWTTTL != 24*time.Hour {
		t.Fatalf("JWTTTL = %v, want %v", cfg.JWTTTL, 24*time.Hour)
	}
	if cfg.AuthRateLimitMax != 10 || cfg.AccountMutationRateLimitMax != 10 || cfg.InviteActionRateLimitMax != 20 {
		t.Fatalf("unexpected rate limit defaults: %+v", cfg)
	}
}

func TestLoadProductionRequiresJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", EnvProduction)
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if !errors.Is(err, ErrInsecureJWTSecret) {
		t.Fatalf("Load() error = %v, want %v", err, ErrInsecureJWTSecret)
	}
}

func TestLoadRejectsInvalidAnalyticsURL(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("ANALYTICS_URL", "ftp://analytics.local")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid ANALYTICS_URL")
	}
}

func TestLoadParsesFrontendOrigins(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("FRONTEND_ORIGINS", "http://localhost:3000, https://example.com, http://localhost:3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.FrontendOrigins) != 2 {
		t.Fatalf("FrontendOrigins len = %d, want 2", len(cfg.FrontendOrigins))
	}
}

func TestLoadParsesJWTTTL(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("JWT_TTL", "12h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.JWTTTL != 12*time.Hour {
		t.Fatalf("JWTTTL = %v, want %v", cfg.JWTTTL, 12*time.Hour)
	}
}
