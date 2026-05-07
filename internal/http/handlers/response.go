package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/sunriseex/finance-manager/internal/http/dto"
	"github.com/sunriseex/finance-manager/internal/repository"
	"github.com/sunriseex/finance-manager/internal/services"
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

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("json body must contain a single object")
	}

	return nil
}

func decodeOptionalJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}

	err := decodeJSON(r, dst)
	if err == nil {
		return nil
	}

	if errors.Is(err, io.EOF) {
		return nil
	}

	return fmt.Errorf("decode optional json body: %w", err)
}

func writeValidationOrServiceError(w http.ResponseWriter, err error) {
	if services.IsValidationError(err) {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	writeServiceError(w, err)
}
