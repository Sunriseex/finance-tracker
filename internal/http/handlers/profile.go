package handlers

import (
	"net/http"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	user, err := services.NewProfileService(h.store.Users()).Get(r.Context(), claims.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.ProfileResponse{User: authUser(user)})
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	var req dto.UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	user, err := services.NewProfileService(h.store.Users()).Update(r.Context(), services.UpdateProfileRequest{
		UserID:          claims.UserID,
		PrimaryCurrency: req.PrimaryCurrency,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.ProfileResponse{User: authUser(user)})
}
