package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrencyRatesRouteRequiresAuth(t *testing.T) {
	router := NewRouter(nil, RouterConfig{APIAuthToken: "test-token"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/currency-rates?base=RUB", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCurrencyRatesRejectsInvalidBase(t *testing.T) {
	router := NewRouter(nil, RouterConfig{APIAuthToken: "test-token"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/currency-rates?base=RU", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
