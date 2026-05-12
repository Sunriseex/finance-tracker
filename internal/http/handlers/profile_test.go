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

func TestProfileRoutesRequireJWT(t *testing.T) {
	tokens, err := auth.NewTokenService(testJWTSecret, "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	router := NewRouter(newTestProfileStore(), &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/settings/profile", http.NoBody)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestProfileRoutesRejectRevokedSessionJWT(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.refresh.byID[pair.RefreshTokenID] = &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    "user-1",
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: new(time.Time),
		CreatedAt: time.Now(),
	}
	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/settings/profile", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestProfileRoutesGetAndPatchPrimaryCurrency(t *testing.T) {
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

	patchReq := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/settings/profile", strings.NewReader(`{"primary_currency":"usd"}`))
	patchReq.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)

	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d", patchRec.Code, http.StatusOK)
	}
	if store.users.byID["user-1"].PrimaryCurrency != "USD" {
		t.Fatalf("primary currency = %q, want USD", store.users.byID["user-1"].PrimaryCurrency)
	}

	getReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/settings/profile", http.NoBody)
	getReq.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if !strings.Contains(getRec.Body.String(), `"primary_currency":"USD"`) {
		t.Fatalf("response body = %s", getRec.Body.String())
	}
}

func TestProfileRoutesRejectInvalidPrimaryCurrency(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com", PrimaryCurrency: "RUB"}
	store.refresh.byID[pair.RefreshTokenID] = &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    "user-1",
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/settings/profile", strings.NewReader(`{"primary_currency":"RU"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

const testJWTSecret = "01234567890123456789012345678901"

func testProfileTokenPair(t *testing.T) (*auth.TokenService, *auth.TokenPair) {
	t.Helper()

	tokens, err := auth.NewTokenService(testJWTSecret, "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	pair, err := tokens.IssuePair("user-1", "user@example.com", time.Now())
	if err != nil {
		t.Fatalf("issue pair: %v", err)
	}
	return tokens, pair
}

type testProfileStore struct {
	users   *testProfileUserRepo
	refresh *testProfileRefreshRepo
	idem    *testIdempotencyRepo
}

func newTestProfileStore() *testProfileStore {
	return &testProfileStore{
		users:   &testProfileUserRepo{byID: map[string]*models.User{}},
		refresh: &testProfileRefreshRepo{byID: map[string]*models.RefreshToken{}},
		idem:    newTestIdempotencyRepo(),
	}
}

func (s *testProfileStore) Accounts() repository.AccountRepository {
	return nil
}

func (s *testProfileStore) Transactions() repository.TransactionRepository {
	return nil
}

func (s *testProfileStore) Categories() repository.CategoryRepository {
	return nil
}

func (s *testProfileStore) InterestRules() repository.InterestRuleRepository {
	return nil
}

func (s *testProfileStore) InterestAccruals() repository.InterestAccrualRepository {
	return nil
}

func (s *testProfileStore) Users() repository.UserRepository {
	return s.users
}

func (s *testProfileStore) RefreshTokens() repository.RefreshTokenRepository {
	return s.refresh
}

func (s *testProfileStore) AuthAuditEvents() repository.AuthAuditRepository {
	return nil
}

func (s *testProfileStore) Idempotency() repository.IdempotencyRepository {
	return s.idem
}

func (s *testProfileStore) Ping(context.Context) error {
	return nil
}

type testProfileUserRepo struct {
	byID map[string]*models.User
}

func (r *testProfileUserRepo) Create(_ context.Context, user *models.User) error {
	r.byID[user.ID] = user
	return nil
}

func (r *testProfileUserRepo) Count(context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

func (r *testProfileUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, user := range r.byID {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *testProfileUserRepo) GetByID(_ context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (r *testProfileUserRepo) RecordLoginFailure(_ context.Context, id string, attempts int, lockedUntil *time.Time, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.FailedLoginAttempts = attempts
	user.LockedUntil = lockedUntil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *testProfileUserRepo) ClearLoginFailures(_ context.Context, id string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *testProfileUserRepo) UpdatePassword(_ context.Context, id, passwordHash string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PasswordHash = passwordHash
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *testProfileUserRepo) UpdatePrimaryCurrency(_ context.Context, id, primaryCurrency string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PrimaryCurrency = primaryCurrency
	user.UpdatedAt = updatedAt
	return nil
}

type testProfileRefreshRepo struct {
	byID map[string]*models.RefreshToken
}

func (r *testProfileRefreshRepo) Create(_ context.Context, token *models.RefreshToken) error {
	r.byID[token.ID] = token
	return nil
}

func (r *testProfileRefreshRepo) GetByID(_ context.Context, id string) (*models.RefreshToken, error) {
	token, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return token, nil
}

func (r *testProfileRefreshRepo) GetByHash(_ context.Context, tokenHash string) (*models.RefreshToken, error) {
	for _, token := range r.byID {
		if token.TokenHash == tokenHash {
			return token, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *testProfileRefreshRepo) ListByUser(_ context.Context, userID string) ([]models.RefreshToken, error) {
	tokens := []models.RefreshToken{}
	for _, token := range r.byID {
		if token.UserID == userID {
			tokens = append(tokens, *token)
		}
	}
	return tokens, nil
}

func (r *testProfileRefreshRepo) Revoke(_ context.Context, id string, revokedAt time.Time) error {
	token, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	token.RevokedAt = &revokedAt
	return nil
}

func (r *testProfileRefreshRepo) RevokeByUserSession(_ context.Context, userID, id string, revokedAt time.Time) error {
	token, ok := r.byID[id]
	if !ok || token.UserID != userID || token.RevokedAt != nil {
		return repository.ErrNotFound
	}
	token.RevokedAt = &revokedAt
	return nil
}

func (r *testProfileRefreshRepo) RevokeByUser(_ context.Context, userID string, revokedAt time.Time) error {
	for _, token := range r.byID {
		if token.UserID == userID {
			token.RevokedAt = &revokedAt
		}
	}
	return nil
}
