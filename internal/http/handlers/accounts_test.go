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

func TestUpdateAccountRejectsCurrencyChangeWithTransactions(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		"11111111-1111-1111-1111-111111111111": testAccount("11111111-1111-1111-1111-111111111111", "user-1", "RUB"),
	}}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/accounts/11111111-1111-1111-1111-111111111111", strings.NewReader(`{
		"currency":"USD"
	}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "change-account-currency")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if store.accounts.(*testAccountRepo).byID["11111111-1111-1111-1111-111111111111"].Currency != "RUB" {
		t.Fatalf("currency changed to %s, want RUB", store.accounts.(*testAccountRepo).byID["11111111-1111-1111-1111-111111111111"].Currency)
	}
	if store.accounts.(*testAccountRepo).updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", store.accounts.(*testAccountRepo).updateCalls)
	}
	if store.accounts.(*testAccountRepo).updateEnforcingInvariantCalls != 1 {
		t.Fatalf("update enforcing invariant calls = %d, want 1", store.accounts.(*testAccountRepo).updateEnforcingInvariantCalls)
	}
}

func TestUpdateAccountAllowsCurrencyChangeWithoutTransactions(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		"11111111-1111-1111-1111-111111111111": testAccount("11111111-1111-1111-1111-111111111111", "user-1", "RUB"),
	}, allowCurrencyChange: true}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/accounts/11111111-1111-1111-1111-111111111111", strings.NewReader(`{
		"currency":"USD"
	}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "change-empty-account-currency")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if store.accounts.(*testAccountRepo).byID["11111111-1111-1111-1111-111111111111"].Currency != "USD" {
		t.Fatalf("currency = %s, want USD", store.accounts.(*testAccountRepo).byID["11111111-1111-1111-1111-111111111111"].Currency)
	}
}

func testAccount(id, ownerUserID, currency string) *models.Account {
	now := time.Now().UTC()
	return &models.Account{
		ID:          id,
		OwnerUserID: &ownerUserID,
		Name:        "Main",
		Type:        models.AccountTypeSavings,
		Currency:    currency,
		IsActive:    true,
		OpenedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func activeTestRefreshToken(pair *auth.TokenPair, userID string) *models.RefreshToken {
	return &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    userID,
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
}

type testAccountRepo struct {
	byID                          map[string]*models.Account
	allowCurrencyChange           bool
	updateCalls                   int
	updateEnforcingInvariantCalls int
}

func (r *testAccountRepo) Create(_ context.Context, account *models.Account) error {
	r.byID[account.ID] = account
	return nil
}

func (r *testAccountRepo) GetByID(_ context.Context, id string) (*models.Account, error) {
	account, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	accountCopy := *account
	return &accountCopy, nil
}

func (r *testAccountRepo) GetByIDForUser(_ context.Context, id, userID string) (*models.Account, error) {
	account, ok := r.byID[id]
	if !ok || account.OwnerUserID == nil || *account.OwnerUserID != userID {
		return nil, repository.ErrNotFound
	}
	accountCopy := *account
	return &accountCopy, nil
}

func (r *testAccountRepo) GetByLegacyID(context.Context, string) (*models.Account, error) {
	return nil, repository.ErrNotFound
}

func (r *testAccountRepo) List(context.Context) ([]models.Account, error) {
	return nil, nil
}

func (r *testAccountRepo) ListByUser(context.Context, string) ([]models.Account, error) {
	return nil, nil
}

func (r *testAccountRepo) Update(_ context.Context, account *models.Account) error {
	r.byID[account.ID] = account
	r.updateCalls++
	return nil
}

func (r *testAccountRepo) UpdateForUser(_ context.Context, account *models.Account, userID string) error {
	stored, ok := r.byID[account.ID]
	if !ok || stored.OwnerUserID == nil || *stored.OwnerUserID != userID {
		return repository.ErrNotFound
	}
	accountCopy := *account
	r.byID[account.ID] = &accountCopy
	r.updateCalls++
	return nil
}

func (r *testAccountRepo) UpdateForUserEnforcingCurrencyInvariant(_ context.Context, account *models.Account, userID string) error {
	r.updateEnforcingInvariantCalls++
	stored, ok := r.byID[account.ID]
	if !ok || stored.OwnerUserID == nil || *stored.OwnerUserID != userID {
		return repository.ErrNotFound
	}
	if stored.Currency != account.Currency && !r.allowCurrencyChange {
		return repository.ErrAccountCurrencyInvariant
	}
	accountCopy := *account
	r.byID[account.ID] = &accountCopy
	r.updateCalls++
	return nil
}

func (r *testAccountRepo) Archive(context.Context, string) error {
	return nil
}

func (r *testAccountRepo) ArchiveForUser(context.Context, string, string) error {
	return nil
}

func (r *testAccountRepo) ClaimUnowned(context.Context, string) error {
	return nil
}

type testTransactionRepo struct {
	transactionCountByAccount map[string]int64
	createCalls               int
	oldCreateCalls            int
	createForUserCalls        int
	createForUserUserID       string
	createForUserErr          error
}

func (r *testTransactionRepo) Create(context.Context, *models.Transaction) error {
	r.createCalls++
	r.oldCreateCalls++
	return nil
}

func (r *testTransactionRepo) CreateForUser(_ context.Context, userID string, _ *models.Transaction) error {
	r.createForUserCalls++
	r.createCalls++
	r.createForUserUserID = userID
	if r.createForUserErr != nil {
		return r.createForUserErr
	}
	return nil
}

func (r *testTransactionRepo) CreateMany(context.Context, []models.Transaction) error {
	return nil
}

func (r *testTransactionRepo) CreateTransfer(context.Context, string, string, string, string, string, []models.Transaction) error {
	return nil
}

func (r *testTransactionRepo) GetByID(context.Context, string) (*models.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (r *testTransactionRepo) GetByIDForUser(context.Context, string, string) (*models.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (r *testTransactionRepo) List(context.Context) ([]models.Transaction, error) {
	return nil, nil
}

func (r *testTransactionRepo) ListByUser(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *testTransactionRepo) ListByAccount(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *testTransactionRepo) ListByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *testTransactionRepo) GetBalanceByAccountForUser(_ context.Context, accountID, _ string) (balanceMinor, transactionCount int64, err error) {
	return 0, r.transactionCountByAccount[accountID], nil
}

func (r *testTransactionRepo) Delete(context.Context, string) error {
	return nil
}

func (r *testTransactionRepo) DeleteForUser(context.Context, string, string) error {
	return nil
}
