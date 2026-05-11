package migration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type JSONMigrator struct {
	accounts     repository.AccountRepository
	transactions repository.TransactionRepository
	rules        repository.InterestRuleRepository
	migration    repository.DepositMigrationRepository
	ownerUserID  string
}

type Option func(*JSONMigrator)

func WithDepositMigrationRepository(repo repository.DepositMigrationRepository) Option {
	return func(m *JSONMigrator) {
		m.migration = repo
	}
}

func WithOwnerUserID(userID string) Option {
	return func(m *JSONMigrator) {
		m.ownerUserID = strings.TrimSpace(userID)
	}
}

func NewJSONMigrator(accounts repository.AccountRepository, transactions repository.TransactionRepository, rules repository.InterestRuleRepository, options ...Option) *JSONMigrator {
	migrator := &JSONMigrator{
		accounts:     accounts,
		transactions: transactions,
		rules:        rules,
	}
	for _, option := range options {
		option(migrator)
	}
	return migrator
}

type JSONMigrationReport struct {
	TotalDeposits        int
	CreatedAccounts      int
	CreatedInterestRules int
	CreatedTransactions  int
	SkippedExisting      int
	OwnerUserID          string
	SourceBalanceMinor   int64
	MigratedBalanceMinor int64
	BalanceMatchesSource bool
	Errors               []string
}

func (m *JSONMigrator) MigrateDeposits(ctx context.Context, deposits []models.Deposit) (*JSONMigrationReport, error) {
	if m.accounts == nil || m.transactions == nil || m.rules == nil || m.migration == nil {
		return nil, fmt.Errorf("migration repositories are required")
	}

	report := &JSONMigrationReport{TotalDeposits: len(deposits), OwnerUserID: m.ownerUserID}

	for i := range deposits {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("migrate deposits: %w", ctx.Err())
		default:
		}

		deposit := &deposits[i]
		report.SourceBalanceMinor += deposit.Amount
		balance, err := m.migrateDeposit(ctx, deposit, report)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", depositLabel(deposit), err))
			continue
		}
		report.MigratedBalanceMinor += balance
	}
	report.BalanceMatchesSource = report.SourceBalanceMinor == report.MigratedBalanceMinor && len(report.Errors) == 0
	return report, nil
}

func parseRequiredDate(fieldName, input string) (time.Time, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", fieldName)
	}

	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s: %q", fieldName, input)
	}

	return dateOnly(date), nil
}

func (m *JSONMigrator) migrateDeposit(ctx context.Context, deposit *models.Deposit, report *JSONMigrationReport) (int64, error) {
	legacyID := strings.TrimSpace(deposit.ID)
	if legacyID == "" {
		return 0, fmt.Errorf("legacy deposit id is required")
	}
	if deposit.Amount < 0 {
		return 0, fmt.Errorf("deposit amount must not be negative")
	}

	existing, err := m.accounts.GetByLegacyID(ctx, legacyID)
	if err == nil {
		return m.migrateExistingDeposit(ctx, deposit, existing, report)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return 0, fmt.Errorf("lookup legacy account: %w", err)
	}

	now := time.Now().UTC()
	openedAt, err := parseRequiredDate("start date", deposit.StartDate)
	if err != nil {
		return 0, err
	}
	legacyIDPtr := legacyID
	accountType, err := accountTypeForDeposit(deposit.Type)
	if err != nil {
		return 0, err
	}
	account := &models.Account{
		ID:          uuid.NewString(),
		LegacyID:    &legacyIDPtr,
		OwnerUserID: ownerUserIDPtr(m.ownerUserID),
		Name:        strings.TrimSpace(deposit.Name),
		Bank:        strings.TrimSpace(deposit.Bank),
		Type:        accountType,
		Currency:    "RUB",
		IsActive:    true,
		OpenedAt:    openedAt,
		CreatedAt:   firstNonZeroTime(deposit.CreatedAt, now),
		UpdatedAt:   firstNonZeroTime(deposit.UpdatedAt, now),
	}
	if account.Name == "" {
		return 0, fmt.Errorf("deposit name is required")
	}

	rule, err := interestRuleForDeposit(deposit, account.ID, openedAt)
	if err != nil {
		return 0, err
	}
	transaction := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeInitialBalance,
		AmountMinor: deposit.Amount,
		Description: legacyInitialDescription(legacyID),
		OccurredAt:  openedAt,
		CreatedAt:   now,
	}

	if err := m.migration.CreateMigratedDeposit(ctx, account, rule, transaction); err != nil {
		return 0, fmt.Errorf("create migrated deposit: %w", err)
	}

	report.CreatedAccounts++
	report.CreatedInterestRules++
	report.CreatedTransactions++

	return deposit.Amount, nil
}

func ownerUserIDPtr(userID string) *string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	return &userID
}

func (m *JSONMigrator) migrateExistingDeposit(ctx context.Context, deposit *models.Deposit, account *models.Account, report *JSONMigrationReport) (int64, error) {
	report.SkippedExisting++
	legacyID := strings.TrimSpace(deposit.ID)
	openedAt, err := parseRequiredDate("start date", deposit.StartDate)
	if err != nil {
		return 0, err
	}

	rules, err := m.rules.ListByAccount(ctx, account.ID)
	if err != nil {
		return 0, fmt.Errorf("list existing rules: %w", err)
	}
	if len(rules) == 0 {
		rule, err := interestRuleForDeposit(deposit, account.ID, openedAt)
		if err != nil {
			return 0, err
		}
		if err := m.rules.Create(ctx, rule); err != nil {
			return 0, fmt.Errorf("create missing interest rule: %w", err)
		}
		report.CreatedInterestRules++
	}

	transactions, err := m.transactions.ListByAccount(ctx, account.ID)
	if err != nil {
		return 0, fmt.Errorf("list existing transactions: %w", err)
	}
	legacyInitialBalance, ok := legacyInitialBalanceFromTransactions(transactions, legacyID)
	if !ok {
		now := time.Now().UTC()
		transaction := &models.Transaction{
			ID:          uuid.NewString(),
			AccountID:   account.ID,
			Type:        models.TransactionTypeInitialBalance,
			AmountMinor: deposit.Amount,
			Description: legacyInitialDescription(legacyID),
			OccurredAt:  openedAt,
			CreatedAt:   now,
		}
		if err := m.transactions.Create(ctx, transaction); err != nil {
			return 0, fmt.Errorf("create missing initial balance transaction: %w", err)
		}
		report.CreatedTransactions++
		legacyInitialBalance = deposit.Amount
	}

	return legacyInitialBalance, nil
}

func legacyInitialBalanceFromTransactions(transactions []models.Transaction, legacyID string) (int64, bool) {
	description := legacyInitialDescription(legacyID)
	var balance int64
	var found bool
	for i := range transactions {
		if transactions[i].Type == models.TransactionTypeInitialBalance && transactions[i].Description == description {
			balance += transactions[i].AmountMinor
			found = true
		}
	}
	return balance, found
}

func interestRuleForDeposit(deposit *models.Deposit, accountID string, openedAt time.Time) (*models.InterestRule, error) {
	if deposit.InterestRate <= 0 {
		return nil, fmt.Errorf("interest rate must be positive")
	}

	var promoRateBps *int64
	if deposit.PromoRate != nil {
		value := rateToBps(*deposit.PromoRate)
		promoRateBps = &value
	}

	var promoEndDate *time.Time
	if strings.TrimSpace(deposit.PromoEndDate) != "" {
		date := parseDate(deposit.PromoEndDate)
		if date.IsZero() {
			return nil, fmt.Errorf("invalid promo end date: %s", deposit.PromoEndDate)
		}
		promoEndDate = &date
	}
	if (promoRateBps == nil) != (promoEndDate == nil) {
		return nil, fmt.Errorf("promo rate and promo end date must be set together")
	}

	var endDate *time.Time
	if strings.TrimSpace(deposit.EndDate) != "" {
		date := parseDate(deposit.EndDate)
		if date.IsZero() {
			return nil, fmt.Errorf("invalid end date: %s", deposit.EndDate)
		}
		endDate = &date
	}

	capitalizationFrequency, err := capitalizationForDeposit(deposit.Capitalization)
	if err != nil {
		return nil, err
	}

	return &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               accountID,
		AnnualRateBps:           rateToBps(deposit.InterestRate),
		PromoRateBps:            promoRateBps,
		PromoEndDate:            promoEndDate,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: capitalizationFrequency,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               dateOnly(openedAt),
		EndDate:                 endDate,
	}, nil
}

func accountTypeForDeposit(depositType string) (models.AccountType, error) {
	switch strings.TrimSpace(depositType) {
	case models.DepositTypeSavings:
		return models.AccountTypeSavings, nil
	case models.DepositTypeTerm:
		return models.AccountTypeTermDeposit, nil
	default:
		return "", fmt.Errorf("unsupported legacy deposit type: %q", depositType)
	}
}

func capitalizationForDeposit(capitalization string) (models.CapitalizationFrequency, error) {
	switch strings.TrimSpace(capitalization) {
	case "":
		return models.CapitalizationFrequencyNone, nil
	case models.CapitalizationDaily:
		return models.CapitalizationFrequencyDaily, nil
	case models.CapitalizationMonthly:
		return models.CapitalizationFrequencyMonthly, nil
	case models.CapitalizationEnd:
		return models.CapitalizationFrequencyEndOfTerm, nil
	case "quarterly":
		return "", fmt.Errorf("unsupported legacy capitalization: quarterly")
	default:
		return "", fmt.Errorf("unsupported legacy capitalization: %q", capitalization)
	}
}

func rateToBps(rate float64) int64 {
	return decimal.NewFromFloat(rate).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

func legacyInitialDescription(legacyID string) string {
	return fmt.Sprintf("legacy initial balance deposit=%s", legacyID)
}

func parseDate(input string) time.Time {
	if strings.TrimSpace(input) == "" {
		return time.Time{}
	}
	date, err := time.Parse(time.DateOnly, input)
	if err != nil {
		return time.Time{}
	}
	return dateOnly(date)
}

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func depositLabel(deposit *models.Deposit) string {
	return fmt.Sprintf("deposit id=%q name=%q bank=%q", deposit.ID, deposit.Name, deposit.Bank)
}
