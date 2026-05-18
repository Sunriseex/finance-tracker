package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRateLimitByIPRejectsAfterLimit(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(2, time.Minute, func() time.Time { return now }, &mu, buckets, trustedProxySet{})(
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

func TestRateLimitByIPIgnoresSpoofedForwardedHeaders(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(2, time.Minute, func() time.Time { return now }, &mu, buckets, trustedProxySet{})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i, forwardedFor := range []string{"198.51.100.1", "198.51.100.2", "198.51.100.3"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
		req.RemoteAddr = "192.0.2.1:1234"
		req.Header.Set("X-Forwarded-For", forwardedFor)
		req.Header.Set("X-Real-IP", forwardedFor)
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

func TestRateLimitByIPUsesForwardedForFromTrustedProxy(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(1, time.Minute, func() time.Time { return now }, &mu, buckets, parseTrustedProxies([]string{"192.0.2.10"}))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for _, forwardedFor := range []string{"198.51.100.1", "198.51.100.2"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
		req.RemoteAddr = "192.0.2.10:1234"
		req.Header.Set("X-Forwarded-For", forwardedFor)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d for forwarded ip %s", rec.Code, http.StatusOK, forwardedFor)
		}
	}
}

func TestRateLimitByIPTrustedProxyFallsBackToRemoteAddr(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(1, time.Minute, func() time.Time { return now }, &mu, buckets, parseTrustedProxies([]string{"192.0.2.0/24"}))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for i := range 2 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
		req.RemoteAddr = "192.0.2.10:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", rec.Code, http.StatusOK)
		}
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("second status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}
	}
}

func TestRateLimitByIPUsesLeftMostValidForwardedFor(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(1, time.Minute, func() time.Time { return now }, &mu, buckets, parseTrustedProxies([]string{"192.0.2.10"}))(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	for _, forwardedFor := range []string{"bad, 198.51.100.1", "198.51.100.1, 198.51.100.2"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
		req.RemoteAddr = "192.0.2.10:1234"
		req.Header.Set("X-Forwarded-For", forwardedFor)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if forwardedFor == "bad, 198.51.100.1" && rec.Code != http.StatusOK {
			t.Fatalf("first status = %d, want %d", rec.Code, http.StatusOK)
		}
		if forwardedFor != "bad, 198.51.100.1" && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("second status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}
	}
}

func TestRateLimitByIPInvalidTrustedProxyEntriesAreIgnoredAndLogged(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{})))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	trusted := parseTrustedProxies([]string{"bad-proxy", "192.0.2.10"})

	if len(trusted.prefixes) != 1 {
		t.Fatalf("trusted prefixes = %d, want 1", len(trusted.prefixes))
	}
	if !strings.Contains(logs.String(), "bad-proxy") {
		t.Fatalf("logs = %q, want invalid proxy entry", logs.String())
	}
}

func TestRateLimitByIPRejectsWithJSONEnvelope(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(1, time.Minute, func() time.Time { return now }, &mu, buckets, trustedProxySet{})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)
	req.RemoteAddr = "192.0.2.1:1234"
	handler.ServeHTTP(httptest.NewRecorder(), req)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != "rate_limited" || body.Error.Message != "Rate limit exceeded" || len(body.Error.Details) != 0 {
		t.Fatalf("error = %+v", body.Error)
	}
}

func TestRateLimitByIPResetsAfterWindow(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	handler := rateLimitByIP(1, time.Minute, func() time.Time { return now }, &mu, buckets, trustedProxySet{})(
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
