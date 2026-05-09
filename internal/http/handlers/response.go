package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	response := errorResponseFromError(err)
	writeError(w, response.status, response.code, response.message, response.details)
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
	writeServiceError(w, err)
}

type errorResponse struct {
	status  int
	code    string
	message string
	details map[string]any
}

func errorResponseFromError(err error) errorResponse {
	if err == nil {
		return errorResponse{
			status:  http.StatusInternalServerError,
			code:    "internal_error",
			message: "Internal server error",
		}
	}

	if errors.Is(err, repository.ErrNotFound) {
		return errorResponse{
			status:  http.StatusNotFound,
			code:    "not_found",
			message: "Resource not found",
		}
	}

	if errors.Is(err, context.Canceled) {
		return errorResponse{
			status:  499,
			code:    "request_canceled",
			message: "Request canceled",
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return errorResponse{
			status:  http.StatusGatewayTimeout,
			code:    "request_timeout",
			message: "Request timed out",
		}
	}

	if services.IsValidationError(err) || isHandlerValidationError(err) {
		return errorResponse{
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: strings.TrimSpace(err.Error()),
		}
	}

	return errorResponse{
		status:  http.StatusInternalServerError,
		code:    "internal_error",
		message: "Internal server error",
	}
}

func isHandlerValidationError(err error) bool {
	var validationErr validationError
	return errors.As(err, &validationErr)
}
