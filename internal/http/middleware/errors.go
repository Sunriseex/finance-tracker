package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/sunriseex/capitalflow/internal/http/dto"
)

func writeJSONError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dto.ErrorEnvelope{
		Error: dto.APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
