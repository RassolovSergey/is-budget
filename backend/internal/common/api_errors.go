package common

import (
	"errors"
	"net/http"
)

type ErrorMapping struct {
	Err     error
	Status  int
	Code    string
	Message string
}

func WriteMappedError(w http.ResponseWriter, err error, fallbackStatus int, fallbackCode, fallbackMessage string, mappings ...ErrorMapping) {
	for _, mapping := range mappings {
		if mapping.Err != nil && errors.Is(err, mapping.Err) {
			msg := mapping.Message
			if msg == "" {
				msg = err.Error()
			}
			WriteErrorWithCode(w, mapping.Status, mapping.Code, msg)
			return
		}
	}

	msg := fallbackMessage
	if msg == "" && err != nil {
		msg = err.Error()
	}
	WriteErrorWithCode(w, fallbackStatus, fallbackCode, msg)
}
