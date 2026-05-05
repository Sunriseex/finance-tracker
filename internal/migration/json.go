package migration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/repository"
)

type JSONMigrator struct {
	accounts     repository.AccountRepository
	transactions repository.TransactionRepository
	rules        repository.InterestRuleRepository
}

func NewJSONMigrator(accounts repository.AccountRepository, transactions repository.TransactionRepository, rules repository.InterestRuleRepository) *JSONMigrator {
	return &JSONMigrator{
		accounts:     accounts,
		transactions: transactions,
		rules:        rules,
	}
}

type JSONMigrationReport struct {
	TotalDeposits        int
	CreatedAccounts      int
	CreatedInterestRules int
	CreatedTransactions  int
	SkippedExisting      int
	SourceBalanceMinor   int64
	MigratedBalanceMinor int64
	BalanceMatchesSource bool
	Errors               []string
}

func (m *JSONMigrator) MigrateDeposits(ctx context.Context, deposits []models.Deposit) (*JSONMigrationReport, error) {
	if m.accounts == nil || m.transactions == nil || m.rules == nil {
		return nil, fmt.Errorf("migration repositories are required")
	}

	report := &JSONMigrationReport{TotalDeposits: len(deposits)}
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
		report.SkippedExisting++
		return m.accountBalance(ctx, existing.ID)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return 0, fmt.Errorf("lookup legacy account: %w", err)
	}

	now := time.Now().UTC()
	openedAt := firstNonZeroDate(parseDate(deposit.StartDate), deposit.CreatedAt, now)
	legacyIDPtr := legacyID
	account := &models.Account{
		ID:        uuid.NewString(),
		LegacyID:  &legacyIDPtr,
		Name:      strings.TrimSpace(deposit.Name),
		Bank:      strings.TrimSpace(deposit.Bank),
		Type:      accountTypeForDeposit(deposit.Type),
		Currency:  "RUB",
		IsActive:  true,
		OpenedAt:  openedAt,
		CreatedAt: firstNonZeroTime(deposit.CreatedAt, now),
		UpdatedAt: firstNonZeroTime(deposit.UpdatedAt, now),
	}
	if account.Name == "" {
		return 0, fmt.Errorf("deposit name is required")
	}
	if err := m.accounts.Create(ctx, account); err != nil {
		return 0, err
	}
	report.CreatedAccounts++

	rule, err := interestRuleForDeposit(deposit, account.ID, openedAt)
	if err != nil {
		return 0, err
	}
	if err := m.rules.Create(ctx, rule); err != nil {
		return 0, err
	}
	report.CreatedInterestRules++

	transaction := &models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeInitialBalance,
		AmountMinor: deposit.Amount,
		Description: fmt.Sprintf("legacy initial balance deposit=%s", legacyID),
		OccurredAt:  openedAt,
		CreatedAt:   now,
	}
	if err := m.transactions.Create(ctx, transaction); err != nil {
		return 0, err
	}
	report.CreatedTransactions++

	return deposit.Amount, nil
}

func (m *JSONMigrator) accountBalance(ctx context.Context, accountID string) (int64, error) {
	transactions, err := m.transactions.ListByAccount(ctx, accountID)
	if err != nil {
		return 0, err
	}
	var balance int64
	for i := range transactions {
		switch transactions[i].Type {
		case models.TransactionTypeInitialBalance,
			models.TransactionTypeIncome,
			models.TransactionTypeTransferIn,
			models.TransactionTypeInterestIncome,
			models.TransactionTypeAdjustment:
			balance += transactions[i].AmountMinor
		case models.TransactionTypeExpense,
			models.TransactionTypeTransferOut:
			balance -= transactions[i].AmountMinor
		default:
			return 0, fmt.Errorf("unknown transaction type: %s", transactions[i].Type)
		}
	}
	return balance, nil
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

	return &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               accountID,
		AnnualRateBps:           rateToBps(deposit.InterestRate),
		PromoRateBps:            promoRateBps,
		PromoEndDate:            promoEndDate,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: capitalizationForDeposit(deposit.Capitalization),
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               dateOnly(openedAt),
		EndDate:                 endDate,
	}, nil
}

func accountTypeForDeposit(depositType string) models.AccountType {
	if depositType == models.DepositTypeTerm {
		return models.AccountTypeTermDeposit
	}
	return models.AccountTypeSavings
}

func capitalizationForDeposit(capitalization string) models.CapitalizationFrequency {
	switch capitalization {
	case models.CapitalizationDaily:
		return models.CapitalizationFrequencyDaily
	case models.CapitalizationMonthly:
		return models.CapitalizationFrequencyMonthly
	case models.CapitalizationEnd:
		return models.CapitalizationFrequencyEndOfTerm
	default:
		return models.CapitalizationFrequencyNone
	}
}

func rateToBps(rate float64) int64 {
	return decimal.NewFromFloat(rate).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
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

func firstNonZeroDate(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return dateOnly(value)
		}
	}
	return time.Time{}
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
