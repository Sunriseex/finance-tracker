package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimitByIPRejectsAfterLimit(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(2, time.Minute, func() time.Time { return now }, &mu, buckets)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i := range 3 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if i < 2 && rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i+1, rec.Code, http.StatusOK)
		}
		if i == 2 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d status = %d, want %d", i+1, rec.Code, http.StatusTooManyRequests)
		}
	}
}

func TestRateLimitByIPResetsAfterWindow(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(1, time.Minute, func() time.Time { return now }, &mu, buckets)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)
	req.RemoteAddr = "192.0.2.1:1234"
	handler.ServeHTTP(httptest.NewRecorder(), req)

	now = now.Add(time.Minute)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
