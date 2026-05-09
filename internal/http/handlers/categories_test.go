package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCategoriesRouteRequiresAuth(t *testing.T) {
	router := NewRouter(nil, RouterConfig{APIAuthToken: "test-token"})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/categories", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCategoriesPreflightSkipsAuth(t *testing.T) {
	router := NewRouter(nil, RouterConfig{
		APIAuthToken:       "test-token",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/api/categories", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow origin = %q", got)
	}
}
