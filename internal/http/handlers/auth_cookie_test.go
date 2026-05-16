package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/http/dto"
)

func TestAuthLogoutClearsRefreshCookieOnServiceError(t *testing.T) {
	router := newTestAuthRouter(t)

	setupRec := httptest.NewRecorder()
	setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	router.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
	}

	refreshCookie := requireRefreshCookie(t, setupRec.Result().Cookies())

	logoutRec := httptest.NewRecorder()
	logoutReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", http.NoBody)
	logoutReq.AddCookie(refreshCookie)
	router.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("first logout status = %d, want %d: %s", logoutRec.Code, http.StatusNoContent, logoutRec.Body.String())
	}

	staleLogoutRec := httptest.NewRecorder()
	staleLogoutReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", http.NoBody)
	staleLogoutReq.AddCookie(refreshCookie)
	router.ServeHTTP(staleLogoutRec, staleLogoutReq)

	clearCookie := requireRefreshCookie(t, staleLogoutRec.Result().Cookies())
	if clearCookie.MaxAge != -1 {
		t.Fatalf("clear cookie MaxAge = %d, want -1", clearCookie.MaxAge)
	}
	if !clearCookie.Expires.Before(time.Now()) {
		t.Fatalf("clear cookie Expires = %s, want past", clearCookie.Expires)
	}
}

func TestRevokeCurrentSessionClearsRefreshCookie(t *testing.T) {
	router := newTestAuthRouter(t)

	setupRec := httptest.NewRecorder()
	setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	router.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
	}
	session := decodeAuthResponse(t, setupRec)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/sessions", http.NoBody)
	listReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d, want %d: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var sessions dto.AuthSessionsResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("decode sessions response: %v", err)
	}
	if len(sessions.Sessions) != 1 {
		t.Fatalf("sessions count = %d, want 1", len(sessions.Sessions))
	}
	if !sessions.Sessions[0].Current {
		t.Fatalf("session = %+v, want current", sessions.Sessions[0])
	}

	revokeRec := httptest.NewRecorder()
	revokeReq := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodDelete,
		"/api/v1/auth/sessions/"+sessions.Sessions[0].ID,
		http.NoBody,
	)
	revokeReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	revokeReq.Header.Set("Idempotency-Key", "revoke-current-session")
	router.ServeHTTP(revokeRec, revokeReq)

	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("revoke session status = %d, want %d: %s", revokeRec.Code, http.StatusNoContent, revokeRec.Body.String())
	}

	clearCookie := requireRefreshCookie(t, revokeRec.Result().Cookies())
	if clearCookie.MaxAge != -1 {
		t.Fatalf("clear cookie MaxAge = %d, want -1", clearCookie.MaxAge)
	}
	if !clearCookie.Expires.Before(time.Now()) {
		t.Fatalf("clear cookie Expires = %s, want past", clearCookie.Expires)
	}
}

func TestAuthSetupSetsSecureRefreshCookie(t *testing.T) {
	router := newTestAuthRouter(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie := requireRefreshCookie(t, rec.Result().Cookies())
	if !cookie.Secure {
		t.Fatal("refresh cookie must be Secure")
	}
	if !cookie.HttpOnly {
		t.Fatal("refresh cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("SameSite = %v, want Strict", cookie.SameSite)
	}
	if cookie.Path != "/auth" {
		t.Fatalf("Path = %q, want /auth", cookie.Path)
	}
}

func TestAuthRefreshAcceptsRefreshCookieFallback(t *testing.T) {
	router := newTestAuthRouter(t)
	setupRec := httptest.NewRecorder()
	setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	router.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
	}

	refreshReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/refresh", http.NoBody)
	refreshReq.AddCookie(requireRefreshCookie(t, setupRec.Result().Cookies()))
	refreshRec := httptest.NewRecorder()
	router.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d: %s", refreshRec.Code, http.StatusOK, refreshRec.Body.String())
	}
	var response dto.AuthResponse
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if response.AccessToken == "" {
		t.Fatal("refresh response must include access token")
	}
	cookie := requireRefreshCookie(t, refreshRec.Result().Cookies())
	if cookie.Value == "" {
		t.Fatal("refresh cookie must contain the rotated refresh token")
	}
}

func TestAuthResponsesDoNotExposeRefreshTokenJSON(t *testing.T) {
	for _, tc := range []struct {
		name       string
		statusCode int
		request    func(cookie *http.Cookie) *http.Request
	}{
		{
			name:       "setup",
			statusCode: http.StatusCreated,
			request: func(_ *http.Cookie) *http.Request {
				return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
					"email":"user@example.com",
					"password":"correct horse battery staple",
					"primary_currency":"RUB"
				}`))
			},
		},
		{
			name:       "refresh",
			statusCode: http.StatusOK,
			request: func(cookie *http.Cookie) *http.Request {
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/refresh", http.NoBody)
				req.AddCookie(cookie)
				return req
			},
		},
	} {
		router := newTestAuthRouter(t)
		rec := httptest.NewRecorder()
		var req *http.Request
		if tc.name == "refresh" {
			setupRec := httptest.NewRecorder()
			setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
				"email":"refresh@example.com",
				"password":"correct horse battery staple",
				"primary_currency":"RUB"
			}`))
			router.ServeHTTP(setupRec, setupReq)
			if setupRec.Code != http.StatusCreated {
				t.Fatalf("setup for refresh status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
			}
			req = tc.request(requireRefreshCookie(t, setupRec.Result().Cookies()))
		} else {
			req = tc.request(nil)
		}

		router.ServeHTTP(rec, req)

		if rec.Code != tc.statusCode {
			t.Fatalf("%s status = %d, want %d: %s", tc.name, rec.Code, tc.statusCode, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode %s response: %v", tc.name, err)
		}
		if _, ok := payload["refresh_token"]; ok {
			t.Fatalf("%s response exposes refresh_token", tc.name)
		}
		if _, ok := payload["refresh_expires_at"]; ok {
			t.Fatalf("%s response exposes refresh_expires_at", tc.name)
		}
	}
}

func TestAuthLogoutClearsRefreshCookie(t *testing.T) {
	router := newTestAuthRouter(t)
	setupRec := httptest.NewRecorder()
	setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	router.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
	}

	logoutReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/logout", http.NoBody)
	logoutReq.AddCookie(requireRefreshCookie(t, setupRec.Result().Cookies()))
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d: %s", logoutRec.Code, http.StatusNoContent, logoutRec.Body.String())
	}
	cookie := requireRefreshCookie(t, logoutRec.Result().Cookies())
	if cookie.MaxAge != -1 {
		t.Fatalf("clear cookie MaxAge = %d, want -1", cookie.MaxAge)
	}
	if !cookie.Expires.Before(time.Now()) {
		t.Fatalf("clear cookie Expires = %s, want past", cookie.Expires)
	}
}

func TestAuthRefreshReuseRevokesSessionFamily(t *testing.T) {
	router := newTestAuthRouter(t)
	setupRec := httptest.NewRecorder()
	setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	router.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
	}
	setupCookie := requireRefreshCookie(t, setupRec.Result().Cookies())

	refreshRec := httptest.NewRecorder()
	refreshReq := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/auth/refresh",
		http.NoBody,
	)
	refreshReq.AddCookie(setupCookie)
	router.ServeHTTP(refreshRec, refreshReq)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want %d: %s", refreshRec.Code, http.StatusOK, refreshRec.Body.String())
	}
	rotatedSession := decodeAuthResponse(t, refreshRec)

	reuseRec := httptest.NewRecorder()
	reuseReq := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/auth/refresh",
		http.NoBody,
	)
	reuseReq.AddCookie(setupCookie)
	router.ServeHTTP(reuseRec, reuseReq)
	if reuseRec.Code != http.StatusBadRequest {
		t.Fatalf("reuse status = %d, want %d: %s", reuseRec.Code, http.StatusBadRequest, reuseRec.Body.String())
	}

	profileRec := httptest.NewRecorder()
	profileReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/settings/profile", http.NoBody)
	profileReq.Header.Set("Authorization", "Bearer "+rotatedSession.AccessToken)
	router.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusUnauthorized {
		t.Fatalf("profile status = %d, want %d after refresh reuse", profileRec.Code, http.StatusUnauthorized)
	}
}

func TestChangePasswordRevokesAllSessions(t *testing.T) {
	router := newTestAuthRouter(t)
	setupRec := httptest.NewRecorder()
	setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	router.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
	}
	session := decodeAuthResponse(t, setupRec)

	changeRec := httptest.NewRecorder()
	changeReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/password", strings.NewReader(`{
		"current_password":"correct horse battery staple",
		"new_password":"fresh correct horse battery staple 2026!"
	}`))
	changeReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	changeReq.Header.Set("Idempotency-Key", "change-password")
	router.ServeHTTP(changeRec, changeReq)
	if changeRec.Code != http.StatusNoContent {
		t.Fatalf("change password status = %d, want %d: %s", changeRec.Code, http.StatusNoContent, changeRec.Body.String())
	}

	profileRec := httptest.NewRecorder()
	profileReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/settings/profile", http.NoBody)
	profileReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	router.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusUnauthorized {
		t.Fatalf("profile status = %d, want %d after password change", profileRec.Code, http.StatusUnauthorized)
	}

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", strings.NewReader(`{
		"email":"user@example.com",
		"password":"fresh correct horse battery staple 2026!"
	}`))
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
}

func TestAuthSessionManagementListsAndRevokesSession(t *testing.T) {
	router := newTestAuthRouter(t)
	setupRec := httptest.NewRecorder()
	setupReq := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/setup", strings.NewReader(`{
		"email":"user@example.com",
		"password":"correct horse battery staple",
		"primary_currency":"RUB"
	}`))
	router.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d: %s", setupRec.Code, http.StatusCreated, setupRec.Body.String())
	}
	session := decodeAuthResponse(t, setupRec)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/sessions", http.NoBody)
	listReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d, want %d: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var sessions dto.AuthSessionsResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("decode sessions response: %v", err)
	}
	if len(sessions.Sessions) != 1 {
		t.Fatalf("sessions count = %d, want 1", len(sessions.Sessions))
	}
	if !sessions.Sessions[0].Active || !sessions.Sessions[0].Current {
		t.Fatalf("session = %+v, want active current", sessions.Sessions[0])
	}

	revokeRec := httptest.NewRecorder()
	revokeReq := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/auth/sessions/"+sessions.Sessions[0].ID, http.NoBody)
	revokeReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	revokeReq.Header.Set("Idempotency-Key", "revoke-session")
	router.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("revoke session status = %d, want %d: %s", revokeRec.Code, http.StatusNoContent, revokeRec.Body.String())
	}

	profileRec := httptest.NewRecorder()
	profileReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/settings/profile", http.NoBody)
	profileReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	router.ServeHTTP(profileRec, profileReq)
	if profileRec.Code != http.StatusUnauthorized {
		t.Fatalf("profile status = %d, want %d after session revoke", profileRec.Code, http.StatusUnauthorized)
	}
}

func newTestAuthRouter(t *testing.T) http.Handler {
	t.Helper()

	tokens, err := auth.NewTokenService(testJWTSecret, "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	return NewRouter(newTestProfileStore(), &RouterConfig{TokenService: tokens})
}

func decodeAuthResponse(t *testing.T, rec *httptest.ResponseRecorder) dto.AuthResponse {
	t.Helper()

	var response dto.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	return response
}

func requireRefreshCookie(t *testing.T, cookies []*http.Cookie) *http.Cookie {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name == refreshCookieName {
			return cookie
		}
	}
	t.Fatalf("missing %s cookie in %v", refreshCookieName, cookies)
	return nil
}
