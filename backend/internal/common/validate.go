package common

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	ErrInvalidJSONBody      = errors.New("invalid json body")
	ErrUnexpectedJSONValues = errors.New("unexpected extra json values")
)

type ValidationError struct {
	Message string
	Details map[string]string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			return field.Name
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			return field.Name
		}
		return name
	})
	return v
}

func ValidateStruct(v *validator.Validate, req any) error {
	if err := v.Struct(req); err != nil {
		if verrs, ok := err.(validator.ValidationErrors); ok {
			parts := make([]string, 0, len(verrs))
			details := make(map[string]string, len(verrs))
			for _, verr := range verrs {
				field := verr.Field()
				if verr.StructField() != "" && field == verr.StructField() {
					field = strings.ToLower(verr.Field())
				}
				msg := fmt.Sprintf("%s failed on %s", field, verr.Tag())
				parts = append(parts, msg)
				details[field] = verr.Tag()
			}
			return &ValidationError{
				Message: strings.Join(parts, ", "),
				Details: details,
			}
		}
		return err
	}
	return nil
}
