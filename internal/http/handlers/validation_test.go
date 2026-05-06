package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestValidateAccountRejectsInvalidCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		{
			name:     "valid RUB",
			currency: "RUB",
		},
		{
			name:     "valid USD",
			currency: "USD",
		},
		{
			name:     "too short",
			currency: "RU",
			wantErr:  true,
		},
		{
			name:     "too long",
			currency: "RUBL",
			wantErr:  true,
		},
		{
			name:     "contains digits",
			currency: "R1B",
			wantErr:  true,
		},
		{
			name:     "contains symbol",
			currency: "12$",
			wantErr:  true,
		},
		{
			name:     "lowercase is rejected at validation level",
			currency: "rub",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := validTestAccount()
			account.Currency = tt.currency

			err := validateAccount(account)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateInterestRuleAppliesDefaults(t *testing.T) {
	rule := validTestInterestRule()
	rule.AccrualFrequency = ""
	rule.CapitalizationFrequency = ""
	rule.DayCountConvention = ""

	if err := validateInterestRule(rule); err != nil {
		t.Fatalf("validate interest rule: %v", err)
	}

	if rule.AccrualFrequency != models.AccrualFrequencyDaily {
		t.Fatalf("accrual frequency = %q, want %q", rule.AccrualFrequency, models.AccrualFrequencyDaily)
	}
	if rule.CapitalizationFrequency != models.CapitalizationFrequencyNone {
		t.Fatalf("capitalization frequency = %q, want %q", rule.CapitalizationFrequency, models.CapitalizationFrequencyNone)
	}
	if rule.DayCountConvention != models.DayCountConventionActual365 {
		t.Fatalf("day count convention = %q, want %q", rule.DayCountConvention, models.DayCountConventionActual365)
	}
}

func TestValidateInterestRuleRejectsInvalidEnums(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*models.InterestRule)
		wantMsg string
	}{
		{
			name: "invalid accrual frequency",
			mutate: func(rule *models.InterestRule) {
				rule.AccrualFrequency = models.AccrualFrequency("weekly")
			},
			wantMsg: "invalid accrual frequency",
		},
		{
			name: "invalid capitalization frequency",
			mutate: func(rule *models.InterestRule) {
				rule.CapitalizationFrequency = models.CapitalizationFrequency("yearly")
			},
			wantMsg: "invalid capitalization frequency",
		},
		{
			name: "invalid day count convention",
			mutate: func(rule *models.InterestRule) {
				rule.DayCountConvention = models.DayCountConvention("30_360")
			},
			wantMsg: "invalid day count convention",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validTestInterestRule()
			tt.mutate(rule)

			err := validateInterestRule(rule)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestValidateInterestRuleRejectsInvalidDateRanges(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*models.InterestRule)
		wantMsg string
	}{
		{
			name: "end date before start date",
			mutate: func(rule *models.InterestRule) {
				endDate := rule.StartDate.AddDate(0, 0, -1)
				rule.EndDate = &endDate
			},
			wantMsg: "end date must be on or after start date",
		},
		{
			name: "promo end date before start date",
			mutate: func(rule *models.InterestRule) {
				promoRate := int64(1_500)
				promoEndDate := rule.StartDate.AddDate(0, 0, -1)
				rule.PromoRateBps = &promoRate
				rule.PromoEndDate = &promoEndDate
			},
			wantMsg: "promo end date must be on or after start date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validTestInterestRule()
			tt.mutate(rule)

			err := validateInterestRule(rule)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestValidateInterestRuleRejectsInvalidPromoConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*models.InterestRule)
		wantMsg string
	}{
		{
			name: "promo rate without promo end date",
			mutate: func(rule *models.InterestRule) {
				promoRate := int64(1_500)
				rule.PromoRateBps = &promoRate
				rule.PromoEndDate = nil
			},
			wantMsg: "promo rate and promo end date must be set together",
		},
		{
			name: "promo end date without promo rate",
			mutate: func(rule *models.InterestRule) {
				promoEndDate := rule.StartDate.AddDate(0, 1, 0)
				rule.PromoRateBps = nil
				rule.PromoEndDate = &promoEndDate
			},
			wantMsg: "promo rate and promo end date must be set together",
		},
		{
			name: "negative promo rate",
			mutate: func(rule *models.InterestRule) {
				promoRate := int64(-1)
				promoEndDate := rule.StartDate.AddDate(0, 1, 0)
				rule.PromoRateBps = &promoRate
				rule.PromoEndDate = &promoEndDate
			},
			wantMsg: "promo rate must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validTestInterestRule()
			tt.mutate(rule)

			err := validateInterestRule(rule)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func validTestAccount() *models.Account {
	now := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	return &models.Account{
		ID:        "account-1",
		Name:      "Main account",
		Type:      models.AccountTypeSavings,
		Currency:  "RUB",
		IsActive:  true,
		OpenedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func validTestInterestRule() *models.InterestRule {
	return &models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC),
	}
}

func TestEnsureRuleBelongsToAccount(t *testing.T) {
	tests := []struct {
		name      string
		rule      *models.InterestRule
		accountID string
		wantErr   bool
	}{
		{
			name: "same account",
			rule: &models.InterestRule{
				ID:        "rule-1",
				AccountID: "account-1",
			},
			accountID: "account-1",
		},
		{
			name: "different account",
			rule: &models.InterestRule{
				ID:        "rule-1",
				AccountID: "account-2",
			},
			accountID: "account-1",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureRuleBelongsToAccount(tt.rule, tt.accountID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLatestApplicableInterestRuleSelectsLatestActiveOnDate(t *testing.T) {
	oldEndDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	rules := []models.InterestRule{
		{
			ID:        "old-rule",
			AccountID: "account-1",
			IsActive:  true,
			StartDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   &oldEndDate,
		},
		{
			ID:        "new-rule",
			AccountID: "account-1",
			IsActive:  true,
			StartDate: time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC),
		},
	}

	rule := latestApplicableInterestRule(
		rules,
		time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
	)

	if rule == nil {
		t.Fatal("expected rule")
	}
	if rule.ID != "new-rule" {
		t.Fatalf("rule id = %q, want new-rule", rule.ID)
	}
}

func TestLatestApplicableInterestRuleIgnoresRulesOutsideAccrualDate(t *testing.T) {
	endDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	rules := []models.InterestRule{
		{
			ID:        "old-rule",
			AccountID: "account-1",
			IsActive:  true,
			StartDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   &endDate,
		},
	}

	rule := latestApplicableInterestRule(
		rules,
		time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
	)

	if rule != nil {
		t.Fatalf("rule = %q, want nil", rule.ID)
	}
}

func TestTransactionsUpToDateFiltersFutureTransactions(t *testing.T) {
	transactions := []models.Transaction{
		{
			ID:          "tx-before",
			AccountID:   "account-1",
			AmountMinor: 100_000,
			OccurredAt:  time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:          "tx-same-day",
			AccountID:   "account-1",
			AmountMinor: 50_000,
			OccurredAt:  time.Date(2026, 5, 5, 23, 59, 59, 0, time.UTC),
		},
		{
			ID:          "tx-after",
			AccountID:   "account-1",
			AmountMinor: 200_000,
			OccurredAt:  time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC),
		},
	}

	filtered := transactionsUpToDate(
		transactions,
		time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	)

	if len(filtered) != 2 {
		t.Fatalf("filtered len = %d, want 2", len(filtered))
	}

	if filtered[0].ID != "tx-before" || filtered[1].ID != "tx-same-day" {
		t.Fatalf("filtered = %+v, want tx-before and tx-same-day", filtered)
	}
}
