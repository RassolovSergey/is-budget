package forecast

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPAnalyticsClientMapsUnavailableStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()

	client := NewHTTPAnalyticsClient(ts.URL)
	_, err := client.Forecast(context.Background(), AnalyticsRequest{})
	if !errors.Is(err, ErrAnalyticsUnavailable) {
		t.Fatalf("expected ErrAnalyticsUnavailable, got %v", err)
	}
}

func TestHTTPAnalyticsClientMapsInvalidRequestStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer ts.Close()

	client := NewHTTPAnalyticsClient(ts.URL)
	_, err := client.Forecast(context.Background(), AnalyticsRequest{})
	if !errors.Is(err, ErrAnalyticsInvalidRequest) {
		t.Fatalf("expected ErrAnalyticsInvalidRequest, got %v", err)
	}
	if errors.Is(err, ErrAnalyticsUnavailable) {
		t.Fatalf("did not expect ErrAnalyticsUnavailable for 422, got %v", err)
	}
}

func TestHTTPAnalyticsClientMapsInvalidResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"forecast":`))
	}))
	defer ts.Close()

	client := NewHTTPAnalyticsClient(ts.URL)
	_, err := client.Forecast(context.Background(), AnalyticsRequest{})
	if !errors.Is(err, ErrAnalyticsInvalidResponse) {
		t.Fatalf("expected ErrAnalyticsInvalidResponse, got %v", err)
	}
}

func TestHTTPAnalyticsClientMapsTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model_version":"v1","horizon_days":1,"forecast":[]}`))
	}))
	defer ts.Close()

	client := NewHTTPAnalyticsClient(ts.URL)
	client.http.Timeout = 20 * time.Millisecond
	_, err := client.Forecast(context.Background(), AnalyticsRequest{})
	if !errors.Is(err, ErrAnalyticsTimeout) {
		t.Fatalf("expected ErrAnalyticsTimeout, got %v", err)
	}
}
