package handlers

import (
	"net/http"

	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
)

func currentUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := appmiddleware.ClaimsFromContext(r.Context())
	if !ok || claims.UserID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required", nil)
		return "", false
	}
	return claims.UserID, true
}
