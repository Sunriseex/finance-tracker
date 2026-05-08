package handlers

import (
	"testing"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

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

func TestExcludeRuleAccrualTransactions(t *testing.T) {
	rule := &models.InterestRule{
		ID:        "rule-1",
		AccountID: "account-1",
	}

	transactions := []models.Transaction{
		{
			ID:          "initial",
			AccountID:   "account-1",
			Type:        models.TransactionTypeInitialBalance,
			AmountMinor: 100_000_00,
		},
		{
			ID:          "rule-1-interest",
			AccountID:   "account-1",
			Type:        models.TransactionTypeInterestIncome,
			AmountMinor: 3_288,
		},
		{
			ID:          "rule-2-interest",
			AccountID:   "account-1",
			Type:        models.TransactionTypeInterestIncome,
			AmountMinor: 1_000,
		},
	}

	accruals := []models.InterestAccrual{
		{
			AccountID:     "account-1",
			RuleID:        "rule-1",
			TransactionID: "rule-1-interest",
		},
		{
			AccountID:     "account-1",
			RuleID:        "rule-2",
			TransactionID: "rule-2-interest",
		},
	}

	got := excludeRuleAccrualTransactions(transactions, accruals, rule)

	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	for _, tx := range got {
		if tx.ID == "rule-1-interest" {
			t.Fatal("transaction for current rule accrual must be excluded")
		}
	}

	if got[0].ID != "initial" {
		t.Fatalf("first transaction id = %s, want initial", got[0].ID)
	}
	if got[1].ID != "rule-2-interest" {
		t.Fatalf("second transaction id = %s, want rule-2-interest", got[1].ID)
	}
}
