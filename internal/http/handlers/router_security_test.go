package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestRouterUsesAPIV1Only(t *testing.T) {
	router := NewRouter(nil, &RouterConfig{APIAuthToken: "test-token"})

	oldReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/categories", http.NoBody)
	oldReq.Header.Set("Authorization", "Bearer test-token")
	oldRec := httptest.NewRecorder()
	router.ServeHTTP(oldRec, oldReq)
	if oldRec.Code != http.StatusNotFound {
		t.Fatalf("old api status = %d, want %d", oldRec.Code, http.StatusNotFound)
	}

	newReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/categories", http.NoBody)
	newRec := httptest.NewRecorder()
	router.ServeHTTP(newRec, newReq)
	if newRec.Code != http.StatusUnauthorized {
		t.Fatalf("new api status = %d, want %d", newRec.Code, http.StatusUnauthorized)
	}
}

func TestMetricsEndpointExposesAuthCounters(t *testing.T) {
	router := NewRouter(newTestProfileStore(), &RouterConfig{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", http.NoBody)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "capitalflow_auth_events_total") {
		t.Fatalf("response body = %s", rec.Body.String())
	}
}

func TestRouterLimitsAuthEndpoints(t *testing.T) {
	router := NewRouter(newTestProfileStore(), &RouterConfig{
		APIAuthToken:          "test-token",
		AuthRateLimitRequests: 1,
		AuthRateLimitWindow:   time.Minute,
	})

	for i := range 2 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if i == 0 && rec.Code == http.StatusTooManyRequests {
			t.Fatal("first login request must not be rate limited")
		}
		if i == 1 {
			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("second login status = %d, want %d", rec.Code, http.StatusTooManyRequests)
			}
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("Retry-After header is required")
			}
		}
	}
}

func TestRouterLimitsMutationsButNotReads(t *testing.T) {
	router := NewRouter(nil, &RouterConfig{
		APIAuthToken:              "test-token",
		MutationRateLimitRequests: 1,
		MutationRateLimitWindow:   time.Minute,
	})

	for range 3 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatal("GET request must not use mutation rate limit")
		}
	}

	for i := range 2 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/accounts", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("second mutation status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}
	}
}

func TestIdempotencyReplaysStoredMutationResponse(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.users.byID["user-1"] = &models.User{
		ID:              "user-1",
		Email:           "user@example.com",
		PrimaryCurrency: "RUB",
	}
	store.refresh.byID[pair.RefreshTokenID] = &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    "user-1",
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	router := NewRouter(store, &RouterConfig{TokenService: tokens})

	for i := range 2 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/settings/profile", strings.NewReader(`{"primary_currency":"USD"}`))
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		req.Header.Set("Idempotency-Key", "profile-update")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i+1, rec.Code, http.StatusOK)
		}
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/settings/profile", strings.NewReader(`{"primary_currency":"EUR"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "profile-update")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("mismatched idempotency status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

type testIdempotencyRepo struct {
	records map[string]*models.IdempotencyRecord
}

func newTestIdempotencyRepo() *testIdempotencyRepo {
	return &testIdempotencyRepo{records: map[string]*models.IdempotencyRecord{}}
}

func (r *testIdempotencyRepo) Get(_ context.Context, key, userID, method, path string) (*models.IdempotencyRecord, error) {
	record, ok := r.records[idempotencyTestKey(key, userID, method, path)]
	if !ok {
		return nil, repository.ErrNotFound
	}
	recordCopy := *record
	recordCopy.ResponseBody = append([]byte(nil), record.ResponseBody...)
	return &recordCopy, nil
}

func (r *testIdempotencyRepo) CreatePending(_ context.Context, record *models.IdempotencyRecord) (bool, error) {
	key := idempotencyTestKey(record.Key, record.UserID, record.Method, record.Path)
	if _, ok := r.records[key]; ok {
		return false, nil
	}
	recordCopy := *record
	r.records[key] = &recordCopy
	return true, nil
}

func (r *testIdempotencyRepo) Complete(_ context.Context, key, userID, method, path string, statusCode int, responseBody []byte) error {
	record, ok := r.records[idempotencyTestKey(key, userID, method, path)]
	if !ok {
		return repository.ErrNotFound
	}
	record.StatusCode = &statusCode
	record.ResponseBody = append([]byte(nil), responseBody...)
	return nil
}

func idempotencyTestKey(key, userID, method, path string) string {
	return key + "\x00" + userID + "\x00" + method + "\x00" + path
}
