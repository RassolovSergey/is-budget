package common

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteMappedError_UsesMatchedMapping(t *testing.T) {
	rootErr := errors.New("root")
	rec := httptest.NewRecorder()

	WriteMappedError(rec, rootErr, http.StatusInternalServerError, "fallback", "fallback message",
		ErrorMapping{Err: rootErr, Status: http.StatusBadRequest, Code: "invalid_request"},
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != "invalid_request" {
		t.Fatalf("code = %q, want %q", payload.Code, "invalid_request")
	}
	if payload.Error != "root" {
		t.Fatalf("error = %q, want %q", payload.Error, "root")
	}
}

func TestWriteMappedError_UsesFallback(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteMappedError(rec, errors.New("boom"), http.StatusInternalServerError, "internal_error", "failed")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Code != "internal_error" {
		t.Fatalf("code = %q, want %q", payload.Code, "internal_error")
	}
	if payload.Error != "failed" {
		t.Fatalf("error = %q, want %q", payload.Error, "failed")
	}
}
