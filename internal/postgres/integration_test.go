package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := t.Context()
	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}

	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `
		TRUNCATE interest_accruals, interest_rules, transactions, categories, accounts RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate test tables; run migrations first: %v", err)
	}

	return NewStore(pool)
}

func seedAccount(ctx context.Context, t *testing.T, store *Store) *models.Account {
	t.Helper()

	now := time.Now().UTC()
	legacyID := "legacy-" + uuid.NewString()

	account := &models.Account{
		ID:        uuid.NewString(),
		LegacyID:  &legacyID,
		Name:      "Integration Savings",
		Bank:      "Yandex",
		Type:      models.AccountTypeSavings,
		Currency:  "RUB",
		IsActive:  true,
		OpenedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	return account
}

func seedInterestRule(ctx context.Context, t *testing.T, store *Store, accountID string) *models.InterestRule {
	t.Helper()

	now := time.Now().UTC()

	rule := &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               accountID,
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyDaily,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               pgDateOnly(now),
	}

	if err := store.InterestRules().Create(ctx, rule); err != nil {
		t.Fatalf("seed interest rule: %v", err)
	}

	return rule
}
func TestPostgresRepositoriesIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := t.Context()
	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		TRUNCATE interest_accruals, interest_rules, transactions, categories, accounts RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate test tables; run migrations first: %v", err)
	}

	store := NewStore(pool)
	accounts := store.Accounts()
	transactions := store.Transactions()
	categories := store.Categories()
	rules := store.InterestRules()
	accruals := store.InterestAccruals()

	now := time.Now().UTC()
	legacyID := "legacy-" + uuid.NewString()
	account := &models.Account{
		ID:        uuid.NewString(),
		LegacyID:  &legacyID,
		Name:      "Integration Savings",
		Bank:      "Yandex",
		Type:      models.AccountTypeSavings,
		Currency:  "RUB",
		IsActive:  true,
		OpenedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := accounts.Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	gotAccount, err := accounts.GetByLegacyID(ctx, legacyID)
	if err != nil {
		t.Fatalf("get by legacy id: %v", err)
	}
	if gotAccount.ID != account.ID {
		t.Fatalf("account id = %s, want %s", gotAccount.ID, account.ID)
	}

	category := &models.Category{
		ID:        uuid.NewString(),
		Slug:      "integration-category",
		Name:      "Integration Category",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := categories.Create(ctx, category); err != nil {
		t.Fatalf("create category: %v", err)
	}
	if _, err := categories.GetBySlug(ctx, category.Slug); err != nil {
		t.Fatalf("get category by slug: %v", err)
	}

	initialBalance := models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeInitialBalance,
		AmountMinor: 100_000,
		CategoryID:  &category.ID,
		Description: "initial",
		OccurredAt:  now,
		CreatedAt:   now,
	}
	income := models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 10_000,
		CategoryID:  &category.ID,
		Description: "income",
		OccurredAt:  now.Add(time.Minute),
		CreatedAt:   now,
	}
	if err := transactions.CreateMany(ctx, []models.Transaction{initialBalance, income}); err != nil {
		t.Fatalf("create transaction batch: %v", err)
	}
	gotTransactions, err := transactions.ListByAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("list account transactions: %v", err)
	}
	if len(gotTransactions) != 2 {
		t.Fatalf("transactions count = %d, want 2", len(gotTransactions))
	}

	rule := &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               account.ID,
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyDaily,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               pgDateOnly(now),
	}
	if err := rules.Create(ctx, rule); err != nil {
		t.Fatalf("create interest rule: %v", err)
	}

	accrualDate := pgDateOnly(now)
	accrual := &models.InterestAccrual{
		ID:            uuid.NewString(),
		AccountID:     account.ID,
		RuleID:        rule.ID,
		TransactionID: income.ID,
		AccrualDate:   accrualDate,
		AmountMinor:   100,
		BalanceMinor:  100_000,
		AnnualRateBps: 1_200,
		CreatedAt:     now,
	}
	if err := accruals.Create(ctx, accrual); err != nil {
		t.Fatalf("create interest accrual: %v", err)
	}
	if err := accruals.Create(ctx, accrual); err == nil {
		t.Fatal("duplicate interest accrual must fail")
	}
	gotAccrual, err := accruals.GetByAccountDateRule(ctx, account.ID, accrualDate.Format(time.DateOnly), rule.ID)
	if err != nil {
		t.Fatalf("get interest accrual: %v", err)
	}
	if gotAccrual.ID != accrual.ID {
		t.Fatalf("accrual id = %s, want %s", gotAccrual.ID, accrual.ID)
	}

	if err := accounts.Archive(ctx, account.ID); err != nil {
		t.Fatalf("archive account: %v", err)
	}
	if err := transactions.Delete(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("delete missing err = %v, want ErrNotFound", err)
	}
}

func TestAccountCreateClaimsSingleExistingUser(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := t.Context()
	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		TRUNCATE interest_accruals, interest_rules, transactions, categories, accounts, refresh_tokens, auth_audit_events, users RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate test tables; run migrations first: %v", err)
	}

	store := NewStore(pool)
	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "owner@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	account := &models.Account{
		ID:        uuid.NewString(),
		Name:      "CLI account",
		Type:      models.AccountTypeSavings,
		Currency:  "RUB",
		IsActive:  true,
		OpenedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	got, err := store.Accounts().GetByIDForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("get account for user: %v", err)
	}
	if got.OwnerUserID == nil || *got.OwnerUserID != userID {
		t.Fatalf("owner_user_id = %v, want %s", got.OwnerUserID, userID)
	}
}

func pgDateOnly(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func TestStoreCreateMigratedDepositRollsBackOnTransactionFailure(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	legacyID := "legacy-rollback"
	account := &models.Account{
		ID:       uuid.NewString(),
		LegacyID: &legacyID,
		Name:     "Rollback test",
		Type:     models.AccountTypeSavings,
		Currency: "RUB",
		IsActive: true,
		OpenedAt: time.Now().UTC(),
	}

	rule := &models.InterestRule{
		ID:                 uuid.NewString(),
		AccountID:          account.ID,
		AnnualRateBps:      1200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Now().UTC(),
	}

	transaction := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   "wrong-account-id",
		Type:        models.TransactionTypeInitialBalance,
		AmountMinor: 100_000,
		OccurredAt:  time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}

	err := store.CreateMigratedDeposit(ctx, account, rule, transaction)
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = store.Accounts().GetByLegacyID(ctx, legacyID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("account must be rolled back, got err = %v", err)
	}
}

func TestInterestAccrualCreateWithTransactionRollsBackOnAccrualFailure(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	account := seedAccount(ctx, t, store)
	seedInterestRule(ctx, t, store, account.ID)
	tx := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeInterestIncome,
		AmountMinor: 100,
		Description: "test interest",
		OccurredAt:  time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}

	accrual := &models.InterestAccrual{
		ID:            uuid.NewString(),
		AccountID:     account.ID,
		RuleID:        "wrong-rule-id",
		TransactionID: tx.ID,
		AccrualDate:   time.Now().UTC(),
		AmountMinor:   100,
		BalanceMinor:  100_000,
		AnnualRateBps: 1200,
		CreatedAt:     time.Now().UTC(),
	}

	err := store.InterestAccruals().CreateWithTransaction(ctx, tx, accrual)
	if err == nil {
		t.Fatal("expected error")
	}

	_, err = store.Transactions().GetByID(ctx, tx.ID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("transaction must be rolled back, got err = %v", err)
	}
}
