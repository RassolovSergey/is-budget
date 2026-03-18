package common

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func ParseUUIDParam(w http.ResponseWriter, r *http.Request, paramName, entityName string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, paramName))
	if err != nil {
		label := strings.TrimSpace(entityName)
		if label == "" {
			label = paramName
		}
		WriteErrorWithCode(w, http.StatusBadRequest, "invalid_param", fmt.Sprintf("invalid %s id", label))
		return uuid.Nil, false
	}
	return id, true
}

func ParseUUIDValue(w http.ResponseWriter, rawValue, fieldName string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(rawValue))
	if err != nil {
		WriteErrorWithCode(w, http.StatusBadRequest, "invalid_param", fmt.Sprintf("%s must be UUID", fieldName))
		return uuid.Nil, false
	}
	return id, true
}

func ParseDateValue(w http.ResponseWriter, rawValue, fieldName string) (time.Time, bool) {
	date, err := ParseDate(rawValue)
	if err != nil {
		WriteErrorWithCode(w, http.StatusBadRequest, "invalid_param", fmt.Sprintf("%s must be YYYY-MM-DD", fieldName))
		return time.Time{}, false
	}
	return date, true
}
