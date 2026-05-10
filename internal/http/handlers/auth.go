package handlers

import (
	"fmt"
	"net/http"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) authStatus(w http.ResponseWriter, r *http.Request) {
	count, err := h.store.Users().Count(r.Context())
	if err != nil {
		writeServiceError(w, fmt.Errorf("count users: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, dto.AuthStatusResponse{SetupRequired: count == 0})
}

func (h *Handler) authSetup(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	session, err := h.authService().Setup(r.Context(), services.AuthRequest{
		Email:           req.Email,
		Password:        req.Password,
		PrimaryCurrency: req.PrimaryCurrency,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse(session))
}

func (h *Handler) authLogin(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	session, err := h.authService().Login(r.Context(), services.AuthRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) authRefresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	session, err := h.authService().Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	if err := h.authService().Logout(r.Context(), req.RefreshToken); err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authService() *services.AuthService {
	return services.NewAuthService(
		h.store.Users(),
		h.store.RefreshTokens(),
		h.tokens,
		h.store.AuthAuditEvents(),
	).WithAccountRepository(h.store.Accounts())
}

func authResponse(session *services.AuthSession) dto.AuthResponse {
	return dto.AuthResponse{
		User:             authUser(session.User),
		AccessToken:      session.AccessToken,
		AccessExpiresAt:  session.AccessExpiresAt,
		RefreshToken:     session.RefreshToken,
		RefreshExpiresAt: session.RefreshExpiresAt,
	}
}

func authUser(user *models.User) dto.AuthUser {
	return dto.AuthUser{
		ID:              user.ID,
		Email:           user.Email,
		PrimaryCurrency: user.PrimaryCurrency,
	}
}
