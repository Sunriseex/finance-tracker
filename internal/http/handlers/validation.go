package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type validationError string

func (e validationError) Error() string {
	return string(e)
}

func errValidation(message string) error {
	return validationError(message)
}

func routeUUIDParam(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	value := strings.TrimSpace(chi.URLParam(r, name))
	if value == "" {
		writeError(w, http.StatusBadRequest, "validation_error", name+" is required", nil)
		return "", false
	}

	if _, err := uuid.Parse(value); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid "+name, nil)
		return "", false
	}

	return value, true
}
func validateOptionalUUID(w http.ResponseWriter, value, field string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}

	if _, err := uuid.Parse(value); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid "+field, nil)
		return false
	}

	return true
}
