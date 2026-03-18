package common

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, ErrorResponse{Error: msg})
}

func WriteErrorWithCode(w http.ResponseWriter, status int, code, msg string) {
	WriteJSON(w, status, ErrorResponse{
		Error: msg,
		Code:  code,
	})
}

func WriteValidationError(w http.ResponseWriter, err error) {
	var verr *ValidationError
	if errors.As(err, &verr) {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   verr.Error(),
			Code:    "validation_error",
			Details: verr.Details,
		})
		return
	}
	WriteErrorWithCode(w, http.StatusBadRequest, "validation_error", err.Error())
}

func WriteDecodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidJSONBody):
		WriteErrorWithCode(w, http.StatusBadRequest, "invalid_json", "invalid json body")
	case errors.Is(err, ErrUnexpectedJSONValues):
		WriteErrorWithCode(w, http.StatusBadRequest, "invalid_json", "json body must contain a single object")
	default:
		WriteErrorWithCode(w, http.StatusBadRequest, "invalid_json", "invalid json body")
	}
}

func DecodeAndValidate(w http.ResponseWriter, r *http.Request, v *validator.Validate, dst any) bool {
	if err := DecodeJSON(r, dst); err != nil {
		WriteDecodeError(w, err)
		return false
	}
	if err := ValidateStruct(v, dst); err != nil {
		WriteValidationError(w, err)
		return false
	}
	return true
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.Join(ErrInvalidJSONBody, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return ErrUnexpectedJSONValues
	}
	return nil
}
