package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
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
		TRUNCATE idempotency_keys, interest_accruals, interest_rules, transfers, transactions, categories, accounts, refresh_tokens, auth_audit_events, users RESTART IDENTITY CASCADE
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

func seedUser(ctx context.Context, t *testing.T, store *Store, email string) string {
	t.Helper()

	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           email,
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return userID
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
		TRUNCATE idempotency_keys, interest_accruals, interest_rules, transfers, transactions, categories, accounts, refresh_tokens, auth_audit_events, users RESTART IDENTITY CASCADE
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

func TestUserRepositorySetupRollsBackOnRefreshTokenFailure(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()

	account := seedAccount(ctx, t, store)
	user := &models.User{
		ID:              uuid.NewString(),
		Email:           "setup@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	refreshToken := &models.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		TokenHash: "setup-refresh-hash",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}

	setupRepo := store.Users().(repository.AuthSetupRepository)
	err := setupRepo.Setup(ctx, user, refreshToken, nil)
	if err == nil {
		t.Fatal("expected setup refresh token failure")
	}

	if _, err := store.Users().GetByID(ctx, user.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("setup user err = %v, want not found", err)
	}
	gotAccount, err := store.Accounts().GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if gotAccount.OwnerUserID != nil {
		t.Fatalf("account owner = %v, want nil after rollback", *gotAccount.OwnerUserID)
	}
	if _, err := store.RefreshTokens().GetByID(ctx, refreshToken.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("setup refresh token err = %v, want not found", err)
	}
}

func TestUserRepositoryChangePasswordAndRevokeSessions(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()

	userID := uuid.NewString()
	lockedUntil := now.Add(time.Hour)
	if err := store.Users().Create(ctx, &models.User{
		ID:                  userID,
		Email:               "change-password@example.com",
		PasswordHash:        "old-hash",
		PrimaryCurrency:     "RUB",
		FailedLoginAttempts: 3,
		LockedUntil:         &lockedUntil,
		CreatedAt:           now,
		UpdatedAt:           now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	otherUserID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              otherUserID,
		Email:           "other-change-password@example.com",
		PasswordHash:    "other-hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	activeToken := &models.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: "active-change-password-token",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	otherToken := &models.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    otherUserID,
		TokenHash: "other-change-password-token",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := store.RefreshTokens().Create(ctx, activeToken); err != nil {
		t.Fatalf("create active token: %v", err)
	}
	if err := store.RefreshTokens().Create(ctx, otherToken); err != nil {
		t.Fatalf("create other token: %v", err)
	}

	changedAt := now.Add(time.Minute)
	if err := store.Users().ChangePasswordAndRevokeSessions(ctx, userID, "new-hash", changedAt, "password_change"); err != nil {
		t.Fatalf("change password and revoke sessions: %v", err)
	}

	user, err := store.Users().GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("get changed user: %v", err)
	}
	if user.PasswordHash != "new-hash" || user.FailedLoginAttempts != 0 || user.LockedUntil != nil {
		t.Fatalf("changed user = %+v, want new hash and cleared lockout", user)
	}
	gotActiveToken, err := store.RefreshTokens().GetByID(ctx, activeToken.ID)
	if err != nil {
		t.Fatalf("get active token: %v", err)
	}
	if gotActiveToken.RevokedAt == nil || gotActiveToken.RevokedReason == nil || *gotActiveToken.RevokedReason != "password_change" {
		t.Fatalf("active token revoke state = %+v", gotActiveToken)
	}
	gotOtherToken, err := store.RefreshTokens().GetByID(ctx, otherToken.ID)
	if err != nil {
		t.Fatalf("get other token: %v", err)
	}
	if gotOtherToken.RevokedAt != nil {
		t.Fatal("other user token was revoked")
	}
}

func TestUserRepositoryRecordLoginFailureConcurrentLocksOnce(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	userID := seedUser(ctx, t, store, "login-failure-concurrent@example.com")

	const workers = 8
	const threshold = 3
	delays := []time.Duration{time.Minute}
	start := make(chan struct{})
	results := make(chan int, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	now := time.Now().UTC()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			attempts, _, err := store.Users().RecordLoginFailure(
				ctx,
				userID,
				threshold,
				delays,
				now.Add(time.Duration(i)*time.Millisecond),
			)
			if err != nil {
				errs <- err
				return
			}
			results <- attempts
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Errorf("record login failure: %v", err)
	}

	seenAttempts := make(map[int]bool, workers)
	for attempts := range results {
		seenAttempts[attempts] = true
	}
	if len(seenAttempts) != workers {
		t.Fatalf("unique attempt results = %d, want %d: %v", len(seenAttempts), workers, seenAttempts)
	}
	for attempt := 1; attempt <= workers; attempt++ {
		if !seenAttempts[attempt] {
			t.Fatalf("missing attempt count %d in %v", attempt, seenAttempts)
		}
	}

	user, err := store.Users().GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.FailedLoginAttempts != workers {
		t.Fatalf("failed_login_attempts = %d, want %d", user.FailedLoginAttempts, workers)
	}
	if user.LockedUntil == nil {
		t.Fatal("locked_until is nil")
	}
}

func TestRefreshTokenRepositoryConcurrentRevokeOnlyOneSucceeds(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID := seedUser(ctx, t, store, "refresh-revoke-concurrent@example.com")
	token := &models.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: "concurrent-revoke-token",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := store.RefreshTokens().Create(ctx, token); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	const workers = 8
	start := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results <- store.RefreshTokens().Revoke(ctx, token.ID, now.Add(time.Duration(i)*time.Millisecond), "logout")
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0
	notFound := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, repository.ErrNotFound):
			notFound++
		default:
			t.Fatalf("unexpected revoke err: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful revokes = %d, want 1", successes)
	}
	if notFound != workers-1 {
		t.Fatalf("not found revokes = %d, want %d", notFound, workers-1)
	}

	got, err := store.RefreshTokens().GetByID(ctx, token.ID)
	if err != nil {
		t.Fatalf("get refresh token: %v", err)
	}
	if got.RevokedAt == nil || got.RevokedReason == nil || *got.RevokedReason != "logout" {
		t.Fatalf("revoke state = %+v, want revoked logout", got)
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
		TRUNCATE idempotency_keys, interest_accruals, interest_rules, transfers, transactions, categories, accounts, refresh_tokens, auth_audit_events, users RESTART IDENTITY CASCADE
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

func TestAccountRepositoryUpdateForUserEnforcesCurrencyInvariant(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()

	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "account-currency@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	account := &models.Account{
		ID:          uuid.NewString(),
		OwnerUserID: &userID,
		Name:        "Currency invariant",
		Type:        models.AccountTypeSavings,
		Currency:    "RUB",
		IsActive:    true,
		OpenedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := store.Transactions().Create(ctx, &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeInitialBalance,
		AmountMinor: 100,
		OccurredAt:  now,
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	account.Currency = "USD"
	account.UpdatedAt = now.Add(time.Minute)
	err := store.Accounts().UpdateForUserEnforcingCurrencyInvariant(ctx, account, userID)
	if !errors.Is(err, repository.ErrAccountCurrencyInvariant) {
		t.Fatalf("currency change err = %v, want ErrAccountCurrencyInvariant", err)
	}
	got, err := store.Accounts().GetByIDForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if got.Currency != "RUB" {
		t.Fatalf("currency = %s, want RUB", got.Currency)
	}

	got.Name = "Renamed with same currency"
	got.UpdatedAt = now.Add(2 * time.Minute)
	if err := store.Accounts().UpdateForUserEnforcingCurrencyInvariant(ctx, got, userID); err != nil {
		t.Fatalf("update same currency: %v", err)
	}
}

func TestTransactionRepositoryCreateForUserScopesByOwner(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()

	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "transaction-owner@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	otherUserID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              otherUserID,
		Email:           "transaction-other@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	account := transferTestAccount(t, store, userID, "owned")
	relatedAccount := transferTestAccount(t, store, userID, "owned-related")
	tx := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 100,
		OccurredAt:  now,
		CreatedAt:   now,
	}
	if err := store.Transactions().CreateForUser(ctx, userID, tx); err != nil {
		t.Fatalf("create transaction for user: %v", err)
	}
	if _, err := store.Transactions().GetByIDForUser(ctx, tx.ID, userID); err != nil {
		t.Fatalf("get transaction for owner: %v", err)
	}

	relatedTx := &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        account.ID,
		RelatedAccountID: &relatedAccount.ID,
		Type:             models.TransactionTypeIncome,
		AmountMinor:      100,
		OccurredAt:       now,
		CreatedAt:        now,
	}
	if err := store.Transactions().CreateForUser(ctx, userID, relatedTx); err != nil {
		t.Fatalf("create transaction with owned related account: %v", err)
	}

	otherTx := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 100,
		OccurredAt:  now,
		CreatedAt:   now,
	}
	if err := store.Transactions().CreateForUser(ctx, otherUserID, otherTx); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("wrong owner err = %v, want ErrNotFound", err)
	}

	otherAccount := transferTestAccount(t, store, otherUserID, "foreign-related")
	foreignRelatedTx := &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        account.ID,
		RelatedAccountID: &otherAccount.ID,
		Type:             models.TransactionTypeIncome,
		AmountMinor:      100,
		OccurredAt:       now,
		CreatedAt:        now,
	}
	if err := store.Transactions().CreateForUser(ctx, userID, foreignRelatedTx); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("foreign related account err = %v, want ErrNotFound", err)
	}
}

func TestTransactionRepositoryCreateForUserLocksRelatedAccountsWithoutDeadlock(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()

	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "transaction-related-locks@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	accountA := transferTestAccount(t, store, userID, "related-lock-a")
	accountB := transferTestAccount(t, store, userID, "related-lock-b")
	txAB := &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        accountA.ID,
		RelatedAccountID: &accountB.ID,
		Type:             models.TransactionTypeAdjustment,
		AmountMinor:      100,
		OccurredAt:       now,
		CreatedAt:        now,
	}
	txBA := &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        accountB.ID,
		RelatedAccountID: &accountA.ID,
		Type:             models.TransactionTypeAdjustment,
		AmountMinor:      -100,
		OccurredAt:       now,
		CreatedAt:        now,
	}

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	errs := make(chan error, 2)
	for _, transaction := range []*models.Transaction{txAB, txBA} {
		go func(transaction *models.Transaction) {
			<-start
			errs <- store.Transactions().CreateForUser(runCtx, userID, transaction)
		}(transaction)
	}

	close(start)
	for range 2 {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatalf("create mirrored related transaction: %v", err)
			}
		case <-runCtx.Done():
			t.Fatalf("mirrored related transactions did not finish: %v", runCtx.Err())
		}
	}

	gotAB, err := store.Transactions().GetByIDForUser(ctx, txAB.ID, userID)
	if err != nil {
		t.Fatalf("get A to B transaction: %v", err)
	}
	if gotAB.RelatedAccountID == nil || *gotAB.RelatedAccountID != accountB.ID {
		t.Fatalf("A to B related account = %v, want %s", gotAB.RelatedAccountID, accountB.ID)
	}
	gotBA, err := store.Transactions().GetByIDForUser(ctx, txBA.ID, userID)
	if err != nil {
		t.Fatalf("get B to A transaction: %v", err)
	}
	if gotBA.RelatedAccountID == nil || *gotBA.RelatedAccountID != accountA.ID {
		t.Fatalf("B to A related account = %v, want %s", gotBA.RelatedAccountID, accountA.ID)
	}
}

func TestAccountCurrencyUpdateWaitsForTransactionCreationAndRejects(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "lock-transaction-first@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	account := transferTestAccount(t, store, userID, "transaction-first")

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin lock transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var lockedID string
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM accounts
		WHERE id = $1 AND owner_user_id = $2
		FOR UPDATE
	`, account.ID, userID).Scan(&lockedID); err != nil {
		t.Fatalf("lock account: %v", err)
	}
	created := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 100,
		OccurredAt:  now,
		CreatedAt:   now,
	}
	if err := insertTransaction(ctx, tx, created); err != nil {
		t.Fatalf("insert locked transaction: %v", err)
	}

	updateAccount := *account
	updateAccount.Currency = "USD"
	updateAccount.UpdatedAt = now.Add(time.Minute)
	errs := make(chan error, 1)
	go func() {
		errs <- store.Accounts().UpdateForUserEnforcingCurrencyInvariant(ctx, &updateAccount, userID)
	}()

	assertStillWaiting(t, errs, "currency update")
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit locked transaction: %v", err)
	}
	err = <-errs
	if !errors.Is(err, repository.ErrAccountCurrencyInvariant) {
		t.Fatalf("currency update err = %v, want ErrAccountCurrencyInvariant", err)
	}

	gotAccount, err := store.Accounts().GetByIDForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if gotAccount.Currency != "RUB" {
		t.Fatalf("currency = %s, want RUB", gotAccount.Currency)
	}
	gotTransactions, err := store.Transactions().ListByAccountForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(gotTransactions) != 1 || gotTransactions[0].ID != created.ID {
		t.Fatalf("transactions = %+v, want committed transaction %s", gotTransactions, created.ID)
	}
}

func TestTransactionCreationWaitsForCurrencyUpdateAndInserts(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "lock-currency-first@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	account := transferTestAccount(t, store, userID, "currency-first")

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin lock update: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var lockedCurrency string
	if err := tx.QueryRow(ctx, `
		SELECT currency
		FROM accounts
		WHERE id = $1 AND owner_user_id = $2
		FOR UPDATE
	`, account.ID, userID).Scan(&lockedCurrency); err != nil {
		t.Fatalf("lock account: %v", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE accounts
		SET currency = $3, updated_at = $4
		WHERE id = $1 AND owner_user_id = $2
	`, account.ID, userID, "USD", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("update locked account currency: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("updated rows = %d, want 1", tag.RowsAffected())
	}

	created := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 100,
		OccurredAt:  now,
		CreatedAt:   now,
	}
	errs := make(chan error, 1)
	go func() {
		errs <- store.Transactions().CreateForUser(ctx, userID, created)
	}()

	assertStillWaiting(t, errs, "transaction creation")
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit locked currency update: %v", err)
	}
	if err := <-errs; err != nil {
		t.Fatalf("create transaction after currency update: %v", err)
	}

	gotAccount, err := store.Accounts().GetByIDForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if gotAccount.Currency != "USD" {
		t.Fatalf("currency = %s, want USD", gotAccount.Currency)
	}
	gotTransaction, err := store.Transactions().GetByIDForUser(ctx, created.ID, userID)
	if err != nil {
		t.Fatalf("get transaction: %v", err)
	}
	if gotTransaction.AccountID != account.ID {
		t.Fatalf("transaction account = %s, want %s", gotTransaction.AccountID, account.ID)
	}
}

func TestInterestAccrualCreateWithTransactionWaitsForCurrencyUpdate(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID, account, rule := seedInterestLockTestData(ctx, t, store, "accrue-lock@example.com", "accrue-lock")

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin currency update: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var lockedID string
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM accounts
		WHERE id = $1 AND owner_user_id = $2
		FOR UPDATE
	`, account.ID, userID).Scan(&lockedID); err != nil {
		t.Fatalf("lock account: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET currency = $2, updated_at = $3 WHERE id = $1`, account.ID, "USD", now.Add(time.Minute)); err != nil {
		t.Fatalf("update locked account: %v", err)
	}

	interestTx, accrual := interestLockTransactionAndAccrual(account.ID, rule.ID, now)
	errs := make(chan error, 1)
	go func() {
		errs <- store.InterestAccruals().CreateWithTransaction(ctx, interestTx, accrual)
	}()

	assertStillWaiting(t, errs, "interest accrual transaction creation")
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit currency update: %v", err)
	}
	if err := <-errs; err != nil {
		t.Fatalf("create interest accrual after currency update: %v", err)
	}

	gotAccount, err := store.Accounts().GetByIDForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if gotAccount.Currency != "USD" {
		t.Fatalf("currency = %s, want USD", gotAccount.Currency)
	}
	if _, err := store.Transactions().GetByIDForUser(ctx, interestTx.ID, userID); err != nil {
		t.Fatalf("get interest transaction: %v", err)
	}
}

func TestInterestAccrualReplaceRangeWaitsForCurrencyUpdate(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID, account, rule := seedInterestLockTestData(ctx, t, store, "replace-lock@example.com", "replace-lock")

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin currency update: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	var lockedID string
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM accounts
		WHERE id = $1 AND owner_user_id = $2
		FOR UPDATE
	`, account.ID, userID).Scan(&lockedID); err != nil {
		t.Fatalf("lock account: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET currency = $2, updated_at = $3 WHERE id = $1`, account.ID, "USD", now.Add(time.Minute)); err != nil {
		t.Fatalf("update locked account: %v", err)
	}

	interestTx, accrual := interestLockTransactionAndAccrual(account.ID, rule.ID, now)
	errs := make(chan error, 1)
	go func() {
		_, err := store.InterestAccruals().ReplaceRangeWithTransactions(ctx, account.ID, rule.ID, pgDateOnly(now), pgDateOnly(now), []models.Transaction{*interestTx}, []models.InterestAccrual{*accrual})
		errs <- err
	}()

	assertStillWaiting(t, errs, "replace interest accrual transactions")
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit currency update: %v", err)
	}
	if err := <-errs; err != nil {
		t.Fatalf("replace interest accruals after currency update: %v", err)
	}

	gotAccount, err := store.Accounts().GetByIDForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if gotAccount.Currency != "USD" {
		t.Fatalf("currency = %s, want USD", gotAccount.Currency)
	}
	if _, err := store.Transactions().GetByIDForUser(ctx, interestTx.ID, userID); err != nil {
		t.Fatalf("get replaced interest transaction: %v", err)
	}
}

func TestInterestCalculationSnapshotBlocksConcurrentTransactionInsert(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID, account, _ := seedInterestLockTestData(ctx, t, store, "snapshot-lock@example.com", "snapshot-lock")

	snapshotReady := make(chan struct{})
	releaseSnapshot := make(chan struct{})
	snapshotErrs := make(chan error, 1)
	go func() {
		snapshotErrs <- store.InterestAccruals().(repository.InterestAccrualTransactionalRepository).WithAccountInterestLock(ctx, account.ID, userID, func(ctx context.Context, snapshot repository.InterestCalculationRepository) error {
			if _, err := snapshot.ListTransactionsByAccountForUser(ctx, account.ID, userID); err != nil {
				return fmt.Errorf("list snapshot transactions: %w", err)
			}
			close(snapshotReady)
			<-releaseSnapshot
			return nil
		})
	}()

	<-snapshotReady
	created := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 1_000,
		OccurredAt:  now,
		CreatedAt:   now,
	}
	insertErrs := make(chan error, 1)
	go func() {
		insertErrs <- store.Transactions().CreateForUser(ctx, userID, created)
	}()

	assertStillWaiting(t, insertErrs, "transaction insert during interest calculation snapshot")
	close(releaseSnapshot)
	if err := <-snapshotErrs; err != nil {
		t.Fatalf("interest calculation snapshot: %v", err)
	}
	if err := <-insertErrs; err != nil {
		t.Fatalf("create transaction after interest snapshot: %v", err)
	}
}

func TestInterestCalculationOverlappingRecalculationsCannotInterleave(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID, account, rule := seedInterestLockTestData(ctx, t, store, "overlap-lock@example.com", "overlap-lock")
	txRepo := store.InterestAccruals().(repository.InterestAccrualTransactionalRepository)
	accrualDate := pgDateOnly(now)

	firstReady := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstErrs := make(chan error, 1)
	firstTx, firstAccrual := interestLockTransactionAndAccrual(account.ID, rule.ID, now)
	go func() {
		firstErrs <- txRepo.WithAccountInterestLock(ctx, account.ID, userID, func(ctx context.Context, snapshot repository.InterestCalculationRepository) error {
			if _, err := snapshot.ReplaceInterestAccrualRangeWithTransactions(ctx, account.ID, rule.ID, accrualDate, accrualDate, []models.Transaction{*firstTx}, []models.InterestAccrual{*firstAccrual}); err != nil {
				return fmt.Errorf("first replace interest accrual range: %w", err)
			}
			close(firstReady)
			<-releaseFirst
			return nil
		})
	}()

	<-firstReady
	secondTx, secondAccrual := interestLockTransactionAndAccrual(account.ID, rule.ID, now.Add(time.Minute))
	secondAccrual.AccrualDate = accrualDate
	secondErrs := make(chan error, 1)
	go func() {
		secondErrs <- txRepo.WithAccountInterestLock(ctx, account.ID, userID, func(ctx context.Context, snapshot repository.InterestCalculationRepository) error {
			if _, err := snapshot.ReplaceInterestAccrualRangeWithTransactions(ctx, account.ID, rule.ID, accrualDate, accrualDate, []models.Transaction{*secondTx}, []models.InterestAccrual{*secondAccrual}); err != nil {
				return fmt.Errorf("second replace interest accrual range: %w", err)
			}
			return nil
		})
	}()

	assertStillWaiting(t, secondErrs, "overlapping interest recalculation")
	close(releaseFirst)
	if err := <-firstErrs; err != nil {
		t.Fatalf("first recalculation: %v", err)
	}
	if err := <-secondErrs; err != nil {
		t.Fatalf("second recalculation: %v", err)
	}

	accruals, err := store.InterestAccruals().ListByAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("list final accruals: %v", err)
	}
	if len(accruals) != 1 {
		t.Fatalf("accrual count = %d, want 1", len(accruals))
	}
	if accruals[0].TransactionID != secondTx.ID {
		t.Fatalf("final accrual transaction = %s, want %s", accruals[0].TransactionID, secondTx.ID)
	}
	if _, err := store.Transactions().GetByIDForUser(ctx, firstTx.ID, userID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("first recalculation transaction err = %v, want ErrNotFound", err)
	}
	if _, err := store.Transactions().GetByIDForUser(ctx, secondTx.ID, userID); err != nil {
		t.Fatalf("get second recalculation transaction: %v", err)
	}
}

func TestInterestAccrualReplaceRangeWithTransactionsRollsBackOnInsertFailure(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	userID, account, rule := seedInterestLockTestData(ctx, t, store, "replace-rollback@example.com", "replace-rollback")
	originalTx, originalAccrual := interestLockTransactionAndAccrual(account.ID, rule.ID, now)
	if err := store.InterestAccruals().CreateWithTransaction(ctx, originalTx, originalAccrual); err != nil {
		t.Fatalf("seed original interest accrual: %v", err)
	}

	replacementTx, replacementAccrual := interestLockTransactionAndAccrual(uuid.NewString(), rule.ID, now.Add(time.Minute))
	replacementAccrual.AccountID = account.ID
	replacementAccrual.AccrualDate = originalAccrual.AccrualDate
	_, err := store.InterestAccruals().ReplaceRangeWithTransactions(
		ctx,
		account.ID,
		rule.ID,
		originalAccrual.AccrualDate,
		originalAccrual.AccrualDate,
		[]models.Transaction{*replacementTx},
		[]models.InterestAccrual{*replacementAccrual},
	)
	if err == nil {
		t.Fatal("expected replacement insert failure")
	}

	gotAccrual, err := store.InterestAccruals().GetByAccountDateRule(ctx, account.ID, originalAccrual.AccrualDate.Format(time.DateOnly), rule.ID)
	if err != nil {
		t.Fatalf("get original accrual after rollback: %v", err)
	}
	if gotAccrual.ID != originalAccrual.ID {
		t.Fatalf("accrual id after rollback = %s, want %s", gotAccrual.ID, originalAccrual.ID)
	}
	if _, err := store.Transactions().GetByIDForUser(ctx, originalTx.ID, userID); err != nil {
		t.Fatalf("get original transaction after rollback: %v", err)
	}
	if _, err := store.Transactions().GetByIDForUser(ctx, replacementTx.ID, userID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("replacement transaction err = %v, want ErrNotFound", err)
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

func TestTransactionCreateTransferLocksAccountsAndRollsBack(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "transfer-owner@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	from := transferTestAccount(t, store, userID, "from")
	to := transferTestAccount(t, store, userID, "to")
	transactions := store.Transactions()

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := range 2 {
		fromID, toID := from.ID, to.ID
		if i == 1 {
			fromID, toID = to.ID, from.ID
		}
		wg.Go(func() {
			transfer, transferTransactions := transferTestRows(userID, fromID, toID, "RUB", "RUB", 100, 100, "1", now)
			errs <- transactions.CreateTransfer(ctx, transfer, transferTransactions)
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("create concurrent transfer: %v", err)
		}
	}

	got, err := transactions.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list transfer transactions: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("transaction count = %d, want 4", len(got))
	}
	fromBalance, fromCount, err := transactions.GetBalanceByAccountForUser(ctx, from.ID, userID)
	if err != nil {
		t.Fatalf("get from balance: %v", err)
	}
	if fromBalance != 0 || fromCount != 2 {
		t.Fatalf("from balance/count = %d/%d, want 0/2", fromBalance, fromCount)
	}

	otherUserID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              otherUserID,
		Email:           "other@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create other user: %v", err)
	}
	otherAccount := transferTestAccount(t, store, otherUserID, "other")
	transfer, transferTransactions := transferTestRows(userID, from.ID, otherAccount.ID, "RUB", "RUB", 100, 100, "1", now)
	err = transactions.CreateTransfer(ctx, transfer, transferTransactions)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("wrong owner err = %v, want ErrNotFound", err)
	}
	got, err = transactions.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list after failed transfer: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("failed transfer inserted rows, count = %d, want 4", len(got))
	}
}

func TestTransactionCreateTransferRejectsStaleLockedCurrencies(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "stale-transfer-currency@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	from := transferTestAccount(t, store, userID, "from-stale-currency")
	to := transferTestAccount(t, store, userID, "to-stale-currency")
	to.Currency = "USD"
	to.UpdatedAt = now.Add(time.Minute)
	if err := store.Accounts().UpdateForUserEnforcingCurrencyInvariant(ctx, to, userID); err != nil {
		t.Fatalf("update to account currency: %v", err)
	}

	transfer, transferTransactions := transferTestRows(userID, from.ID, to.ID, "RUB", "KRW", 100, 1_625, "16.25", now)
	err := store.Transactions().CreateTransfer(ctx, transfer, transferTransactions)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("stale currency err = %v, want ErrConflict", err)
	}
	got, err := store.Transactions().ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("stale currency transfer inserted transactions: %+v", got)
	}
}

func TestTransactionCreateTransferPersistsAuditRecord(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           "audit-transfer@example.com",
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	from := transferTestAccount(t, store, userID, "from-audit")
	to := transferTestAccount(t, store, userID, "to-audit")
	to.Currency = "KRW"
	to.UpdatedAt = now.Add(time.Minute)
	if err := store.Accounts().UpdateForUserEnforcingCurrencyInvariant(ctx, to, userID); err != nil {
		t.Fatalf("update to account currency: %v", err)
	}

	transfer, transferTransactions := transferTestRows(userID, from.ID, to.ID, "RUB", "KRW", 1_000_000, 16_250_000, "16.25", now)
	if err := store.Transactions().CreateTransfer(ctx, transfer, transferTransactions); err != nil {
		t.Fatalf("create transfer: %v", err)
	}

	var gotRate string
	var gotFromTx string
	var gotToTx string
	if err := store.pool.QueryRow(ctx, `
		SELECT exchange_rate::text, from_transaction_id::text, to_transaction_id::text
		FROM transfers
		WHERE id = $1
	`, transfer.ID).Scan(&gotRate, &gotFromTx, &gotToTx); err != nil {
		t.Fatalf("get transfer audit: %v", err)
	}
	if gotRate != "16.250000000000000000" {
		t.Fatalf("exchange rate = %s, want 16.250000000000000000", gotRate)
	}
	if gotFromTx != transferTransactions[0].ID || gotToTx != transferTransactions[1].ID {
		t.Fatalf("linked tx ids = %s/%s, want %s/%s", gotFromTx, gotToTx, transferTransactions[0].ID, transferTransactions[1].ID)
	}

	gotTransactions, err := store.Transactions().ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list transfer transactions: %v", err)
	}
	for i := range gotTransactions {
		if gotTransactions[i].TransferID == nil || *gotTransactions[i].TransferID != transfer.ID {
			t.Fatalf("transaction %s transfer_id = %v, want %s", gotTransactions[i].ID, gotTransactions[i].TransferID, transfer.ID)
		}
	}
}

func TestTransactionCreateTransferRejectsBrokenTransferInvariants(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	now := time.Now().UTC()
	userID := seedUser(ctx, t, store, "broken-transfer-invariant@example.com")
	from := transferTestAccount(t, store, userID, "from-broken-transfer")
	to := transferTestAccount(t, store, userID, "to-broken-transfer")

	standalone := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   from.ID,
		Type:        models.TransactionTypeTransferOut,
		AmountMinor: 100,
		OccurredAt:  now,
		CreatedAt:   now,
	}
	if err := store.Transactions().CreateForUser(ctx, userID, standalone); err == nil {
		t.Fatal("standalone transfer transaction must fail")
	}

	transfer, transferTransactions := transferTestRows(userID, from.ID, to.ID, "RUB", "RUB", 100, 100, "1", now)
	transferTransactions[1].AmountMinor = 101
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin broken transfer: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := insertTransfer(ctx, tx, transfer); err != nil {
		t.Fatalf("insert transfer audit: %v", err)
	}
	for i := range transferTransactions {
		if err := insertTransaction(ctx, tx, &transferTransactions[i]); err != nil {
			t.Fatalf("insert transfer transaction: %v", err)
		}
	}
	if err := tx.Commit(ctx); err == nil {
		t.Fatal("mismatched transfer legs must fail at commit")
	}
}

func TestInterestAccrualTransactionIDIsUnique(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	now := time.Now().UTC()
	userID := seedUser(ctx, t, store, "unique-accrual-transaction@example.com")
	account := transferTestAccount(t, store, userID, "unique-accrual")
	rule := seedInterestRule(ctx, t, store, account.ID)

	tx, accrual := interestLockTransactionAndAccrual(account.ID, rule.ID, now)
	if err := store.InterestAccruals().CreateWithTransaction(ctx, tx, accrual); err != nil {
		t.Fatalf("create first accrual: %v", err)
	}

	duplicate := *accrual
	duplicate.ID = uuid.NewString()
	duplicate.AccrualDate = pgDateOnly(now.AddDate(0, 0, 1))
	if err := store.InterestAccruals().Create(ctx, &duplicate); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("duplicate transaction accrual err = %v, want ErrConflict", err)
	}
}

func TestFinancialSchemaHasExpectedIndexesAndConstraints(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	expectedIndexes := []string{
		"transactions_related_account_id_idx",
		"transactions_category_id_idx",
		"interest_accruals_rule_id_idx",
		"categories_parent_id_idx",
	}
	for _, indexName := range expectedIndexes {
		var exists bool
		if err := store.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND indexname = $1
			)
		`, indexName).Scan(&exists); err != nil {
			t.Fatalf("check index %s: %v", indexName, err)
		}
		if !exists {
			t.Fatalf("index %s does not exist", indexName)
		}
	}

	expectedConstraints := []string{
		"transactions_transfer_type_id_check",
		"interest_accruals_transaction_id_unique",
	}
	for _, constraintName := range expectedConstraints {
		var exists bool
		if err := store.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = $1
			)
		`, constraintName).Scan(&exists); err != nil {
			t.Fatalf("check constraint %s: %v", constraintName, err)
		}
		if !exists {
			t.Fatalf("constraint %s does not exist", constraintName)
		}
	}
}

func TestTransactionRepositoryGetBalanceByAccountForUserMatchesBalanceService(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	now := time.Now().UTC()
	userID := seedUser(ctx, t, store, "balance-owner@example.com")
	otherUserID := seedUser(ctx, t, store, "balance-other@example.com")
	account := transferTestAccount(t, store, userID, "balance-primary")
	transferPeer := transferTestAccount(t, store, userID, "balance-peer")
	otherAccount := transferTestAccount(t, store, otherUserID, "balance-other")

	transactions := []models.Transaction{
		{ID: uuid.NewString(), AccountID: account.ID, Type: models.TransactionTypeInitialBalance, AmountMinor: 100_000, OccurredAt: now, CreatedAt: now},
		{ID: uuid.NewString(), AccountID: account.ID, Type: models.TransactionTypeIncome, AmountMinor: 50_000, OccurredAt: now, CreatedAt: now},
		{ID: uuid.NewString(), AccountID: account.ID, Type: models.TransactionTypeExpense, AmountMinor: 20_000, OccurredAt: now, CreatedAt: now},
		{ID: uuid.NewString(), AccountID: account.ID, Type: models.TransactionTypeAdjustment, AmountMinor: -5_000, OccurredAt: now, CreatedAt: now},
		{ID: uuid.NewString(), AccountID: account.ID, Type: models.TransactionTypeInterestIncome, AmountMinor: 300, OccurredAt: now, CreatedAt: now},
	}
	for i := range transactions {
		if err := store.Transactions().CreateForUser(ctx, userID, &transactions[i]); err != nil {
			t.Fatalf("create balance transaction %s: %v", transactions[i].Type, err)
		}
	}

	transferOut, transferOutRows := transferTestRows(userID, account.ID, transferPeer.ID, "RUB", "RUB", 25_000, 25_000, "1", now)
	if err := store.Transactions().CreateTransfer(ctx, transferOut, transferOutRows); err != nil {
		t.Fatalf("create transfer out: %v", err)
	}
	transferIn, transferInRows := transferTestRows(userID, transferPeer.ID, account.ID, "RUB", "RUB", 10_000, 10_000, "1", now)
	if err := store.Transactions().CreateTransfer(ctx, transferIn, transferInRows); err != nil {
		t.Fatalf("create transfer in: %v", err)
	}

	otherTransaction := &models.Transaction{ID: uuid.NewString(), AccountID: otherAccount.ID, Type: models.TransactionTypeIncome, AmountMinor: 999_999, OccurredAt: now, CreatedAt: now}
	if err := store.Transactions().CreateForUser(ctx, otherUserID, otherTransaction); err != nil {
		t.Fatalf("create other user transaction: %v", err)
	}

	listed, err := store.Transactions().ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list user transactions: %v", err)
	}
	want, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
		AccountID:    account.ID,
		Transactions: listed,
	})
	if err != nil {
		t.Fatalf("calculate expected balance: %v", err)
	}
	gotBalance, gotCount, err := store.Transactions().GetBalanceByAccountForUser(ctx, account.ID, userID)
	if err != nil {
		t.Fatalf("get SQL balance: %v", err)
	}
	if gotBalance != want.BalanceMinor || gotCount != int64(want.Count) {
		t.Fatalf("SQL balance/count = %d/%d, want %d/%d", gotBalance, gotCount, want.BalanceMinor, want.Count)
	}

	otherBalance, otherCount, err := store.Transactions().GetBalanceByAccountForUser(ctx, account.ID, otherUserID)
	if err != nil {
		t.Fatalf("get other user balance: %v", err)
	}
	if otherBalance != 0 || otherCount != 0 {
		t.Fatalf("other user balance/count = %d/%d, want 0/0", otherBalance, otherCount)
	}
}

func TestTransactionRepositoryListByUserFilteredAppliesSQLFiltersAndPagination(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	userID := seedUser(ctx, t, store, "filtered-transactions@example.com")
	otherUserID := seedUser(ctx, t, store, "filtered-transactions-other@example.com")
	account := transferTestAccount(t, store, userID, "filtered-primary")
	otherAccount := transferTestAccount(t, store, userID, "filtered-secondary")
	foreignAccount := transferTestAccount(t, store, otherUserID, "filtered-foreign")
	categoryID := uuid.NewString()
	if err := store.Categories().Create(ctx, &models.Category{
		ID:        categoryID,
		Slug:      "filtered-salary",
		Name:      "Filtered Salary",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create category: %v", err)
	}

	transactions := []models.Transaction{
		{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			Type:        models.TransactionTypeIncome,
			AmountMinor: 100,
			CategoryID:  &categoryID,
			Description: "Salary May",
			OccurredAt:  time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			Type:        models.TransactionTypeExpense,
			AmountMinor: 50,
			Description: "Salary tagged but expense",
			OccurredAt:  time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			Type:        models.TransactionTypeIncome,
			AmountMinor: 200,
			CategoryID:  &categoryID,
			Description: "Salary June",
			OccurredAt:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          uuid.NewString(),
			AccountID:   otherAccount.ID,
			Type:        models.TransactionTypeIncome,
			AmountMinor: 300,
			CategoryID:  &categoryID,
			Description: "Salary Other Account",
			OccurredAt:  time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC),
		},
	}
	for i := range transactions {
		if err := store.Transactions().CreateForUser(ctx, userID, &transactions[i]); err != nil {
			t.Fatalf("create filtered transaction: %v", err)
		}
	}
	foreign := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   foreignAccount.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 999,
		CategoryID:  &categoryID,
		Description: "Salary Foreign",
		OccurredAt:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		CreatedAt:   time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	}
	if err := store.Transactions().CreateForUser(ctx, otherUserID, foreign); err != nil {
		t.Fatalf("create foreign transaction: %v", err)
	}

	filter := &repository.TransactionListFilter{
		AccountID:  account.ID,
		CategoryID: categoryID,
		Type:       models.TransactionTypeIncome,
		FromDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:     time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		Search:     "salary",
		Limit:      1,
		Page:       2,
	}

	filtered, err := store.Transactions().(*TransactionRepository).ListByUserFiltered(ctx, userID, filter)
	if err != nil {
		t.Fatalf("list filtered transactions: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1: %+v", len(filtered), filtered)
	}
	if filtered[0].ID != transactions[2].ID {
		t.Fatalf("filtered transaction = %s, want %s", filtered[0].ID, transactions[2].ID)
	}

}

func TestInterestRuleRepositoryListByUserScopesRulesToOwnedAccounts(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)
	userID := seedUser(ctx, t, store, "interest-rules-user@example.com")
	otherUserID := seedUser(ctx, t, store, "interest-rules-other@example.com")
	account := transferTestAccount(t, store, userID, "rules-primary")
	otherAccount := transferTestAccount(t, store, otherUserID, "rules-foreign")
	rule := seedInterestRule(ctx, t, store, account.ID)
	seedInterestRule(ctx, t, store, otherAccount.ID)

	got, err := store.InterestRules().(*InterestRuleRepository).ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list user interest rules: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("rules count = %d, want 1: %+v", len(got), got)
	}
	if got[0].ID != rule.ID {
		t.Fatalf("rule id = %s, want %s", got[0].ID, rule.ID)
	}
}

func TestTransactionRepositoryListByUserFilteredTreatsSearchAsLiteralSubstring(t *testing.T) {
	ctx := t.Context()
	store := newTestStore(t)

	userID := seedUser(ctx, t, store, "filtered-literal-search@example.com")
	account := transferTestAccount(t, store, userID, "filtered-literal-search")

	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	transactions := []models.Transaction{
		{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			Type:        models.TransactionTypeIncome,
			AmountMinor: 100,
			Description: "Cashback 5%",
			OccurredAt:  now,
			CreatedAt:   now,
		},
		{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			Type:        models.TransactionTypeIncome,
			AmountMinor: 200,
			Description: "Cashback normal",
			OccurredAt:  now.Add(time.Hour),
			CreatedAt:   now.Add(time.Hour),
		},
		{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			Type:        models.TransactionTypeIncome,
			AmountMinor: 300,
			Description: "fee_code",
			OccurredAt:  now.Add(2 * time.Hour),
			CreatedAt:   now.Add(2 * time.Hour),
		},
	}
	for i := range transactions {
		if err := store.Transactions().CreateForUser(ctx, userID, &transactions[i]); err != nil {
			t.Fatalf("create transaction: %v", err)
		}
	}

	filteredPercent, err := store.Transactions().(*TransactionRepository).ListByUserFiltered(ctx, userID, &repository.TransactionListFilter{
		Search: "%",
	})
	if err != nil {
		t.Fatalf("list percent search: %v", err)
	}
	if len(filteredPercent) != 1 || filteredPercent[0].ID != transactions[0].ID {
		t.Fatalf("percent search = %+v, want only %s", filteredPercent, transactions[0].ID)
	}

	filteredUnderscore, err := store.Transactions().(*TransactionRepository).ListByUserFiltered(ctx, userID, &repository.TransactionListFilter{
		Search: "_",
	})
	if err != nil {
		t.Fatalf("list underscore search: %v", err)
	}
	if len(filteredUnderscore) != 1 || filteredUnderscore[0].ID != transactions[2].ID {
		t.Fatalf("underscore search = %+v, want only %s", filteredUnderscore, transactions[2].ID)
	}
}

func transferTestAccount(t *testing.T, store *Store, userID, name string) *models.Account {
	t.Helper()
	now := time.Now().UTC()
	account := &models.Account{
		ID:          uuid.NewString(),
		OwnerUserID: &userID,
		Name:        name,
		Type:        models.AccountTypeSavings,
		Currency:    "RUB",
		IsActive:    true,
		OpenedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.Accounts().Create(t.Context(), account); err != nil {
		t.Fatalf("create account %s: %v", name, err)
	}
	return account
}

func transferTestRows(userID, fromID, toID, fromCurrency, toCurrency string, fromAmount, toAmount int64, exchangeRate string, now time.Time) (*models.Transfer, []models.Transaction) {
	transferID := uuid.NewString()
	fromTransactionID := uuid.NewString()
	toTransactionID := uuid.NewString()
	relatedTo := toID
	relatedFrom := fromID
	return &models.Transfer{
			ID:                   transferID,
			UserID:               userID,
			FromAccountID:        fromID,
			ToAccountID:          toID,
			FromTransactionID:    fromTransactionID,
			ToTransactionID:      toTransactionID,
			FromAmountMinor:      fromAmount,
			ToAmountMinor:        toAmount,
			FromCurrency:         fromCurrency,
			ToCurrency:           toCurrency,
			ExchangeRate:         exchangeRate,
			ExchangeRateProvider: "test",
			ExchangeRateDate:     now,
			CreatedAt:            now,
		}, []models.Transaction{
			{
				ID:               fromTransactionID,
				AccountID:        fromID,
				RelatedAccountID: &relatedTo,
				TransferID:       &transferID,
				Type:             models.TransactionTypeTransferOut,
				AmountMinor:      fromAmount,
				OccurredAt:       now,
				CreatedAt:        now,
			},
			{
				ID:               toTransactionID,
				AccountID:        toID,
				RelatedAccountID: &relatedFrom,
				TransferID:       &transferID,
				Type:             models.TransactionTypeTransferIn,
				AmountMinor:      toAmount,
				OccurredAt:       now,
				CreatedAt:        now,
			},
		}
}

func seedInterestLockTestData(ctx context.Context, t *testing.T, store *Store, email, accountName string) (string, *models.Account, *models.InterestRule) {
	t.Helper()
	now := time.Now().UTC()
	userID := uuid.NewString()
	if err := store.Users().Create(ctx, &models.User{
		ID:              userID,
		Email:           email,
		PasswordHash:    "hash",
		PrimaryCurrency: "RUB",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	account := &models.Account{
		ID:          uuid.NewString(),
		OwnerUserID: &userID,
		Name:        accountName,
		Type:        models.AccountTypeSavings,
		Currency:    "RUB",
		IsActive:    true,
		OpenedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("create account %s: %v", accountName, err)
	}
	rule := seedInterestRule(ctx, t, store, account.ID)
	return userID, account, rule
}

func interestLockTransactionAndAccrual(accountID, ruleID string, now time.Time) (*models.Transaction, *models.InterestAccrual) {
	txID := uuid.NewString()
	return &models.Transaction{
			ID:          txID,
			AccountID:   accountID,
			Type:        models.TransactionTypeInterestIncome,
			AmountMinor: 100,
			OccurredAt:  now,
			CreatedAt:   now,
		}, &models.InterestAccrual{
			ID:            uuid.NewString(),
			AccountID:     accountID,
			RuleID:        ruleID,
			TransactionID: txID,
			AccrualDate:   pgDateOnly(now),
			AmountMinor:   100,
			BalanceMinor:  10_000,
			AnnualRateBps: 1_200,
			CreatedAt:     now,
		}
}

func assertStillWaiting(t *testing.T, errs <-chan error, operation string) {
	t.Helper()

	select {
	case err := <-errs:
		t.Fatalf("%s finished before account lock was released: %v", operation, err)
	case <-time.After(100 * time.Millisecond):
	}
}
