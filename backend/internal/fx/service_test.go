package fx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"isbudget/backend/internal/db/sqlc"
)

func TestFetchRatesMapsUnavailableStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()

	svc := NewService(sqlc.New(nil), srv.URL)
	_, err := svc.fetchRates(context.Background(), "EUR")
	if !errors.Is(err, ErrFXUnavailable) {
		t.Fatalf("fetchRates() error = %v, want ErrFXUnavailable", err)
	}
}

func TestFetchRatesMapsInvalidResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"success","rates":{}}`))
	}))
	defer srv.Close()

	svc := NewService(sqlc.New(nil), srv.URL)
	_, err := svc.fetchRates(context.Background(), "EUR")
	if !errors.Is(err, ErrFXInvalidResponse) {
		t.Fatalf("fetchRates() error = %v, want ErrFXInvalidResponse", err)
	}
}
