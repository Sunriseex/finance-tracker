package handlers

import (
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
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
