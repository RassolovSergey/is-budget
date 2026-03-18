package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type decodeValidateRequest struct {
	Name string `json:"name" validate:"required,min=2"`
}

func TestDecodeJSONRejectsTrailingObjects(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}{"name":"extra"}`))
	var dst decodeValidateRequest

	err := DecodeJSON(req, &dst)
	if err == nil {
		t.Fatal("DecodeJSON() error = nil, want trailing json error")
	}
}

func TestDecodeAndValidateWritesStructuredValidationError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a"}`))
	rec := httptest.NewRecorder()
	var dst decodeValidateRequest

	ok := DecodeAndValidate(rec, req, NewValidator(), &dst)
	if ok {
		t.Fatal("DecodeAndValidate() = true, want false")
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("response body missing error field: %s", body)
	}
	if !strings.Contains(body, `"code":"validation_error"`) {
		t.Fatalf("response body missing code field: %s", body)
	}
	if !strings.Contains(body, `"details"`) {
		t.Fatalf("response body missing details field: %s", body)
	}
}
