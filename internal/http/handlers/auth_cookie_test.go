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
	if response.RefreshToken == "" {
		t.Fatal("refresh response must keep JSON refresh token compatibility")
	}
	cookie := requireRefreshCookie(t, refreshRec.Result().Cookies())
	if cookie.Value == "" || cookie.Value != response.RefreshToken {
		t.Fatal("refresh cookie must contain the rotated refresh token")
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
	setupSession := decodeAuthResponse(t, setupRec)

	refreshRec := httptest.NewRecorder()
	refreshReq := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/auth/refresh",
		strings.NewReader(`{"refresh_token":"`+setupSession.RefreshToken+`"}`),
	)
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
		strings.NewReader(`{"refresh_token":"`+setupSession.RefreshToken+`"}`),
	)
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
