package services

import (
	"testing"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestInterestRuleServiceCreate(t *testing.T) {
	startDate := time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local)

	rule, err := NewInterestRuleService(nil).Create(t.Context(), CreateInterestRuleRequest{
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyDaily,
		DayCountConvention:      models.DayCountConventionActual365,
		StartDate:               startDate,
	})
	if err != nil {
		t.Fatalf("create interest rule: %v", err)
	}
	if rule.ID == "" {
		t.Fatal("id is empty")
	}
	if rule.AccountID != "account-1" {
		t.Fatalf("account id = %s, want account-1", rule.AccountID)
	}
	if !rule.StartDate.Equal(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start date = %s, want date only UTC", rule.StartDate)
	}
	if !rule.IsActive {
		t.Fatal("rule must be active")
	}
}

func TestInterestRuleServiceCreateDefaults(t *testing.T) {
	rule, err := NewInterestRuleService(nil).Create(t.Context(), CreateInterestRuleRequest{
		AccountID:     "account-1",
		AnnualRateBps: 1_200,
	})
	if err != nil {
		t.Fatalf("create interest rule: %v", err)
	}
	if rule.AccrualFrequency != models.AccrualFrequencyDaily {
		t.Fatalf("accrual frequency = %s, want daily", rule.AccrualFrequency)
	}
	if rule.CapitalizationFrequency != models.CapitalizationFrequencyNone {
		t.Fatalf("capitalization frequency = %s, want none", rule.CapitalizationFrequency)
	}
	if rule.DayCountConvention != models.DayCountConventionActual365 {
		t.Fatalf("day count convention = %s, want actual_365", rule.DayCountConvention)
	}
}

func TestInterestRuleServiceAccrue(t *testing.T) {
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 15, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if got.Skipped {
		t.Fatal("accrual must not be skipped")
	}
	if got.Transaction == nil {
		t.Fatal("transaction is nil")
	}
	if got.Transaction.Type != models.TransactionTypeInterestIncome {
		t.Fatalf("transaction type = %s, want interest_income", got.Transaction.Type)
	}
	if got.Transaction.AmountMinor != 3_288 {
		t.Fatalf("amount = %d, want 3288", got.Transaction.AmountMinor)
	}
	if got.Transaction.OccurredAt.Format(time.DateOnly) != "2026-05-04" {
		t.Fatalf("occurred at = %s, want 2026-05-04", got.Transaction.OccurredAt.Format(time.DateOnly))
	}
}

func TestInterestRuleServiceAccrueUsesPromoRate(t *testing.T) {
	promoRate := int64(2_400)
	promoEndDate := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		PromoRateBps:       &promoRate,
		PromoEndDate:       &promoEndDate,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if got.Transaction.AmountMinor != 6_575 {
		t.Fatalf("amount = %d, want 6575", got.Transaction.AmountMinor)
	}
}

func TestInterestRuleServiceAccrueSkipsDuplicate(t *testing.T) {
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	accrualDate := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  accrualDate,
		ExistingTransactions: []models.Transaction{
			{
				AccountID:   "account-1",
				Type:        models.TransactionTypeInterestIncome,
				AmountMinor: 3_288,
				Description: "interest accrual rule=rule-1 date=2026-05-04",
				OccurredAt:  accrualDate,
			},
		},
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if !got.Skipped {
		t.Fatal("accrual must be skipped")
	}
	if got.Transaction != nil {
		t.Fatal("transaction must be nil")
	}
}

func TestInterestRuleServiceAccrueValidatesRuleDate(t *testing.T) {
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}

	_, err := NewInterestRuleService(nil).Accrue(t.Context(), AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
