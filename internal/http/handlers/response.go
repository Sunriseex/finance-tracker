package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sunriseex/finance-manager/internal/http/dto"
	"github.com/sunriseex/finance-manager/internal/repository"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	writeJSON(w, status, dto.ErrorEnvelope{
		Error: dto.APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found", nil)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode json body: %w", err)
	}
	return nil
}
