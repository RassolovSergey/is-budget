package common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestParseUUIDParam_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/projects/not-a-uuid", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("projectId", "not-a-uuid")
	req = contextWithRoute(req, routeCtx)
	rec := httptest.NewRecorder()

	if _, ok := ParseUUIDParam(rec, req, "projectId", "project"); ok {
		t.Fatal("ParseUUIDParam() ok = true, want false")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestParseDateValue_Invalid(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, ok := ParseDateValue(rec, "16-03-2026", "from"); ok {
		t.Fatal("ParseDateValue() ok = true, want false")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func contextWithRoute(req *http.Request, routeCtx *chi.Context) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}
