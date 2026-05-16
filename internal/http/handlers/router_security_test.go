package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/auth"
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
	if strings.Contains(rec.Body.String(), "cmdline") {
		t.Fatalf("metrics response leaked expvar cmdline: %s", rec.Body.String())
	}
}

func TestRouterCORSAllowsCredentialsForConfiguredOrigin(t *testing.T) {
	tokens, err := auth.NewTokenService(testJWTSecret, "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	router := NewRouter(newTestProfileStore(), &RouterConfig{
		TokenService:       tokens,
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/auth/login", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want http://localhost:5173", got)
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

func TestFinanceMutationsRequireIdempotencyKey(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := NewRouter(store, &RouterConfig{TokenService: tokens})

	tests := []struct {
		name   string
		path   string
		method string
		body   string
	}{
		{name: "transaction", path: "/api/v1/transactions", method: http.MethodPost, body: `{}`},
		{name: "transfer", path: "/api/v1/transfers", method: http.MethodPost, body: `{}`},
		{name: "accrue interest", path: "/api/v1/accounts/11111111-1111-1111-1111-111111111111/accrue-interest", method: http.MethodPost, body: `{}`},
		{name: "recalculate interest", path: "/api/v1/accounts/11111111-1111-1111-1111-111111111111/recalculate-interest", method: http.MethodPost, body: `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusPreconditionRequired {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusPreconditionRequired, rec.Body.String())
			}
		})
	}
}

func TestFinanceMutationIdempotencyReplaysStoredResponse(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		"11111111-1111-1111-1111-111111111111": testAccount("11111111-1111-1111-1111-111111111111", "user-1", "RUB"),
	}}
	transactions := &testTransactionRepo{transactionCountByAccount: map[string]int64{}}
	store.transactions = transactions
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := NewRouter(store, &RouterConfig{TokenService: tokens})

	body := `{
		"account_id":"11111111-1111-1111-1111-111111111111",
		"type":"income",
		"amount_minor":1000
	}`
	var firstBody string
	for i := range 2 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transactions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		req.Header.Set("Idempotency-Key", "create-income")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, want %d: %s", i+1, rec.Code, http.StatusCreated, rec.Body.String())
		}
		if i == 0 {
			firstBody = rec.Body.String()
		} else if rec.Body.String() != firstBody {
			t.Fatalf("replayed response body changed: got %s want %s", rec.Body.String(), firstBody)
		}
	}
	if transactions.createCalls != 1 {
		t.Fatalf("transaction create calls = %d, want 1", transactions.createCalls)
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
