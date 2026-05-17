package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestAccrueInterestForAccountFallbackPersistsAccrual(t *testing.T) {
	accrualDate := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	store := newTestInterestAccrualStore(accrualDate)
	accruals := &testInterestAccrualRepo{}
	store.accruals = accruals
	h := &Handler{store: store}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/accounts/account-1/accrue-interest", http.NoBody)

	result, err := h.accrueInterestForAccount(req, testInterestAccountID, "user-1", testInterestRuleID, accrualDate)
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if result == nil || result.Skipped {
		t.Fatalf("result = %#v, want persisted accrual", result)
	}
	if accruals.createdTransaction == nil {
		t.Fatal("expected fallback to create transaction with accrual")
	}
	if accruals.createdAccrual == nil {
		t.Fatal("expected fallback to create accrual")
	}
	if accruals.createdTransaction.ID != result.Transaction.ID {
		t.Fatalf("created transaction id = %s, want %s", accruals.createdTransaction.ID, result.Transaction.ID)
	}
	if accruals.createdAccrual.ID != result.Accrual.ID {
		t.Fatalf("created accrual id = %s, want %s", accruals.createdAccrual.ID, result.Accrual.ID)
	}
}

func TestAccrueInterestForAccountConflictRollsBackLockedAccrual(t *testing.T) {
	accrualDate := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	store := newTestInterestAccrualStore(accrualDate)
	txAccruals := &testTransactionalInterestAccrualRepo{
		snapshot: testInterestSnapshot{
			rule:         store.rule,
			transactions: store.transactions.transactions,
			createErr:    repository.ErrConflict,
		},
	}
	store.accruals = txAccruals
	h := &Handler{store: store}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/accounts/account-1/accrue-interest", http.NoBody)

	result, err := h.accrueInterestForAccount(req, testInterestAccountID, "user-1", testInterestRuleID, accrualDate)
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if result == nil || !result.Skipped {
		t.Fatalf("result = %#v, want skipped conflict response", result)
	}
	if txAccruals.committed {
		t.Fatal("expected locked transaction not to commit after conflict")
	}
	if !txAccruals.rolledBack {
		t.Fatal("expected locked transaction to roll back after conflict")
	}
}

func TestLatestApplicableInterestRuleSelectsRuleForResolvedEndDate(t *testing.T) {
	oldRule := models.InterestRule{
		ID:        "old-rule",
		AccountID: "account-1",
		IsActive:  true,
		StartDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   ptrTime(time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)),
	}

	currentRule := models.InterestRule{
		ID:        "current-rule",
		AccountID: "account-1",
		IsActive:  true,
		StartDate: time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC),
	}

	got := latestApplicableInterestRule(
		[]models.InterestRule{oldRule, currentRule},
		time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC),
	)

	if got == nil {
		t.Fatal("expected rule")
	}
	if got.ID != "current-rule" {
		t.Fatalf("rule id = %s, want current-rule", got.ID)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

const (
	testInterestAccountID = "11111111-1111-1111-1111-111111111111"
	testInterestRuleID    = "22222222-2222-2222-2222-222222222222"
)

type testInterestAccrualStore struct {
	rule         *models.InterestRule
	transactions *testInterestTransactionRepo
	rules        *testInterestRuleRepo
	accruals     repository.InterestAccrualRepository
}

func newTestInterestAccrualStore(accrualDate time.Time) *testInterestAccrualStore {
	startDate := accrualDate.AddDate(0, 0, -1)
	rule := &models.InterestRule{
		ID:                      testInterestRuleID,
		AccountID:               testInterestAccountID,
		AnnualRateBps:           36500,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               startDate,
	}
	return &testInterestAccrualStore{
		rule: rule,
		transactions: &testInterestTransactionRepo{transactions: []models.Transaction{
			{
				ID:          "principal-1",
				AccountID:   testInterestAccountID,
				Type:        models.TransactionTypeIncome,
				AmountMinor: 10000,
				OccurredAt:  startDate,
				CreatedAt:   startDate,
			},
		}},
		rules: &testInterestRuleRepo{rule: rule},
	}
}

func (s *testInterestAccrualStore) Accounts() repository.AccountRepository {
	return nil
}

func (s *testInterestAccrualStore) Transactions() repository.TransactionRepository {
	return s.transactions
}

func (s *testInterestAccrualStore) Categories() repository.CategoryRepository {
	return nil
}

func (s *testInterestAccrualStore) InterestRules() repository.InterestRuleRepository {
	return s.rules
}

func (s *testInterestAccrualStore) InterestAccruals() repository.InterestAccrualRepository {
	return s.accruals
}

func (s *testInterestAccrualStore) Users() repository.UserRepository {
	return nil
}

func (s *testInterestAccrualStore) RefreshTokens() repository.RefreshTokenRepository {
	return nil
}

func (s *testInterestAccrualStore) AuthAuditEvents() repository.AuthAuditRepository {
	return nil
}

func (s *testInterestAccrualStore) Idempotency() repository.IdempotencyRepository {
	return nil
}

func (s *testInterestAccrualStore) Ping(context.Context) error {
	return nil
}

type testInterestRuleRepo struct {
	rule *models.InterestRule
}

func (r *testInterestRuleRepo) Create(context.Context, *models.InterestRule) error {
	return nil
}

func (r *testInterestRuleRepo) GetByID(_ context.Context, id string) (*models.InterestRule, error) {
	if r.rule != nil && r.rule.ID == id {
		return r.rule, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testInterestRuleRepo) ListByAccount(_ context.Context, accountID string) ([]models.InterestRule, error) {
	if r.rule != nil && r.rule.AccountID == accountID {
		return []models.InterestRule{*r.rule}, nil
	}
	return nil, nil
}

func (r *testInterestRuleRepo) Update(context.Context, *models.InterestRule) error {
	return nil
}

type testInterestTransactionRepo struct {
	transactions []models.Transaction
}

func (r *testInterestTransactionRepo) Create(context.Context, *models.Transaction) error {
	return nil
}

func (r *testInterestTransactionRepo) CreateForUser(context.Context, string, *models.Transaction) error {
	return nil
}

func (r *testInterestTransactionRepo) CreateMany(context.Context, []models.Transaction) error {
	return nil
}

func (r *testInterestTransactionRepo) CreateTransfer(context.Context, string, string, string, string, string, []models.Transaction) error {
	return nil
}

func (r *testInterestTransactionRepo) GetByID(context.Context, string) (*models.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (r *testInterestTransactionRepo) GetByIDForUser(context.Context, string, string) (*models.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (r *testInterestTransactionRepo) List(context.Context) ([]models.Transaction, error) {
	return r.transactions, nil
}

func (r *testInterestTransactionRepo) ListByUser(context.Context, string) ([]models.Transaction, error) {
	return r.transactions, nil
}

func (r *testInterestTransactionRepo) ListByAccount(context.Context, string) ([]models.Transaction, error) {
	return r.transactions, nil
}

func (r *testInterestTransactionRepo) ListByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return r.transactions, nil
}

func (r *testInterestTransactionRepo) GetBalanceByAccountForUser(context.Context, string, string) (balanceMinor, transactionCount int64, err error) {
	return 0, 0, nil
}

func (r *testInterestTransactionRepo) Delete(context.Context, string) error {
	return nil
}

func (r *testInterestTransactionRepo) DeleteForUser(context.Context, string, string) error {
	return nil
}

type testInterestAccrualRepo struct {
	accruals           []models.InterestAccrual
	createdTransaction *models.Transaction
	createdAccrual     *models.InterestAccrual
	createErr          error
}

func (r *testInterestAccrualRepo) Create(_ context.Context, accrual *models.InterestAccrual) error {
	r.accruals = append(r.accruals, *accrual)
	return nil
}

func (r *testInterestAccrualRepo) CreateWithTransaction(_ context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.createdTransaction = transaction
	r.createdAccrual = accrual
	r.accruals = append(r.accruals, *accrual)
	return nil
}

func (r *testInterestAccrualRepo) ReplaceRangeWithTransactions(context.Context, string, string, time.Time, time.Time, []models.Transaction, []models.InterestAccrual) (int64, error) {
	return 0, nil
}

func (r *testInterestAccrualRepo) GetByAccountDateRule(_ context.Context, accountID, accrualDate, ruleID string) (*models.InterestAccrual, error) {
	for i := range r.accruals {
		accrual := &r.accruals[i]
		if accrual.AccountID == accountID && accrual.RuleID == ruleID && accrual.AccrualDate.Format(time.DateOnly) == accrualDate {
			return accrual, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *testInterestAccrualRepo) ListByAccount(_ context.Context, accountID string) ([]models.InterestAccrual, error) {
	var accruals []models.InterestAccrual
	for i := range r.accruals {
		accrual := r.accruals[i]
		if accrual.AccountID == accountID {
			accruals = append(accruals, accrual)
		}
	}
	return accruals, nil
}

type testTransactionalInterestAccrualRepo struct {
	testInterestAccrualRepo
	snapshot   testInterestSnapshot
	committed  bool
	rolledBack bool
}

func (r *testTransactionalInterestAccrualRepo) WithAccountInterestLock(ctx context.Context, _, _ string, fn func(context.Context, repository.InterestCalculationRepository) error) error {
	if err := fn(ctx, &r.snapshot); err != nil {
		r.rolledBack = true
		return err
	}
	r.committed = true
	return nil
}

type testInterestSnapshot struct {
	rule         *models.InterestRule
	transactions []models.Transaction
	accruals     []models.InterestAccrual
	createErr    error
}

func (s *testInterestSnapshot) GetInterestRuleByID(_ context.Context, id string) (*models.InterestRule, error) {
	if s.rule != nil && s.rule.ID == id {
		return s.rule, nil
	}
	return nil, repository.ErrNotFound
}

func (s *testInterestSnapshot) ListInterestRulesByAccount(_ context.Context, accountID string) ([]models.InterestRule, error) {
	if s.rule != nil && s.rule.AccountID == accountID {
		return []models.InterestRule{*s.rule}, nil
	}
	return nil, nil
}

func (s *testInterestSnapshot) ListTransactionsByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return s.transactions, nil
}

func (s *testInterestSnapshot) ListInterestAccrualsByAccount(context.Context, string) ([]models.InterestAccrual, error) {
	return s.accruals, nil
}

func (s *testInterestSnapshot) CreateInterestAccrualWithTransaction(context.Context, *models.Transaction, *models.InterestAccrual) error {
	return s.createErr
}

func (s *testInterestSnapshot) ReplaceInterestAccrualRangeWithTransactions(context.Context, string, string, time.Time, time.Time, []models.Transaction, []models.InterestAccrual) (int64, error) {
	return 0, nil
}
