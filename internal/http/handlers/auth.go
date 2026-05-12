package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
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
	setRefreshCookie(w, session)
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
	setRefreshCookie(w, session)
	writeJSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) authRefresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	req.RefreshToken = refreshTokenFromRequest(r, req.RefreshToken)

	session, err := h.authService().Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	setRefreshCookie(w, session)
	writeJSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	req.RefreshToken = refreshTokenFromRequest(r, req.RefreshToken)

	if err := h.authService().Logout(r.Context(), req.RefreshToken); err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	var req dto.ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	if err := h.authService().ChangePassword(r.Context(), services.ChangePasswordRequest{
		UserID:          claims,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	sessions, err := h.authService().ListSessions(r.Context(), claims.UserID, claims.SessionID)
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authSessionsResponse(sessions))
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	sessionID := chi.URLParam(r, "id")
	if err := h.authService().RevokeSession(r.Context(), userID, sessionID); err != nil {
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

func authSessionsResponse(sessions []services.SessionInfo) dto.AuthSessionsResponse {
	response := dto.AuthSessionsResponse{
		Sessions: make([]dto.AuthSessionInfo, 0, len(sessions)),
	}
	for _, session := range sessions {
		response.Sessions = append(response.Sessions, dto.AuthSessionInfo{
			ID:        session.ID,
			ExpiresAt: session.ExpiresAt,
			RevokedAt: session.RevokedAt,
			CreatedAt: session.CreatedAt,
			Active:    session.Active,
			Current:   session.Current,
		})
	}
	return response
}

const refreshCookieName = "__Secure-capitalflow_refresh"

func setRefreshCookie(w http.ResponseWriter, session *services.AuthSession) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    session.RefreshToken,
		Path:     "/auth",
		Expires:  session.RefreshExpiresAt,
		MaxAge:   max(1, int(time.Until(session.RefreshExpiresAt).Seconds())),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func refreshTokenFromRequest(r *http.Request, bodyToken string) string {
	if bodyToken != "" {
		return bodyToken
	}
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
