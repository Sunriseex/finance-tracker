package services

import (
	"context"
	"testing"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestInterestRuleServiceCreate(t *testing.T) {
	startDate := time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local)

	rule, err := NewInterestRuleService(nil).Create(t.Context(), &CreateInterestRuleRequest{
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
	rule, err := NewInterestRuleService(nil).Create(t.Context(), &CreateInterestRuleRequest{
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

func TestInterestRuleServiceCreateNormalizesDatePointers(t *testing.T) {
	promoRate := int64(2_400)
	promoEndDate := time.Date(2026, 5, 31, 23, 59, 59, 0, time.Local)
	endDate := time.Date(2026, 12, 31, 23, 59, 59, 0, time.Local)

	rule, err := NewInterestRuleService(nil).Create(t.Context(), &CreateInterestRuleRequest{
		AccountID:     "account-1",
		AnnualRateBps: 1_200,
		PromoRateBps:  &promoRate,
		PromoEndDate:  &promoEndDate,
		EndDate:       &endDate,
	})
	if err != nil {
		t.Fatalf("create interest rule: %v", err)
	}
	if rule.PromoEndDate == nil || rule.PromoEndDate.Format(time.RFC3339) != "2026-05-31T00:00:00Z" {
		t.Fatalf("promo end date = %v, want 2026-05-31 UTC date", rule.PromoEndDate)
	}
	if rule.EndDate == nil || rule.EndDate.Format(time.RFC3339) != "2026-12-31T00:00:00Z" {
		t.Fatalf("end date = %v, want 2026-12-31 UTC date", rule.EndDate)
	}
}

func TestInterestRuleServiceCreateRejectsIncompletePromo(t *testing.T) {
	promoRate := int64(2_400)
	promoEndDate := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		req  CreateInterestRuleRequest
	}{
		{
			name: "promo rate without end date",
			req: CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1_200,
				PromoRateBps:  &promoRate,
			},
		},
		{
			name: "promo end date without rate",
			req: CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1_200,
				PromoEndDate:  &promoEndDate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewInterestRuleService(nil).Create(t.Context(), &tt.req)
			if err == nil {
				t.Fatal("expected error")
			}
		})
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

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
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
	if got.Accrual == nil {
		t.Fatal("accrual is nil")
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
	if got.Accrual.RuleID != "rule-1" {
		t.Fatalf("accrual rule id = %s, want rule-1", got.Accrual.RuleID)
	}
}

func TestInterestRuleServiceAccruePersistsTransactionAndAccrualAtomically(t *testing.T) {
	accruals := &recordingInterestAccrualRepo{}
	transactions := &recordingTransactionRepo{}
	service := NewInterestRuleService(
		NewTransactionService(transactions),
		WithInterestAccrualRepository(accruals),
	)
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := service.Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if got.Transaction == nil || got.Accrual == nil {
		t.Fatal("transaction and accrual must be returned")
	}
	if transactions.createCalls != 0 {
		t.Fatalf("transaction repo create calls = %d, want 0", transactions.createCalls)
	}
	if accruals.createCalls != 0 {
		t.Fatalf("accrual repo create calls = %d, want 0", accruals.createCalls)
	}
	if accruals.createWithTransactionCalls != 1 {
		t.Fatalf("atomic create calls = %d, want 1", accruals.createWithTransactionCalls)
	}
	if accruals.transaction == nil || accruals.accrual == nil {
		t.Fatal("atomic create must receive transaction and accrual")
	}
	if accruals.transaction.ID != got.Transaction.ID {
		t.Fatalf("atomic transaction id = %s, want %s", accruals.transaction.ID, got.Transaction.ID)
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

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
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

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  accrualDate,
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:   "account-1",
				RuleID:      "rule-1",
				AccrualDate: accrualDate,
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
	if got.Accrual != nil {
		t.Fatal("accrual must be nil")
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

	_, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInterestRuleServiceAccrueRejectsUnsupportedCapitalization(t *testing.T) {
	rule := models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyMonthly,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInterestRuleServiceRecalculateDefaultsRange(t *testing.T) {
	accruals := &recordingInterestAccrualRepo{}
	service := NewInterestRuleService(
		NewTransactionService(),
		WithInterestAccrualRepository(accruals),
	)
	rule := validAccrualTestRule()

	got, err := service.Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:          "tx-1",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInitialBalance,
				AmountMinor: 100_000_00,
				OccurredAt:  rule.StartDate,
			},
		},
		Today: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.FromDate.Format(time.DateOnly) != "2026-05-01" {
		t.Fatalf("from date = %s, want 2026-05-01", got.FromDate.Format(time.DateOnly))
	}
	if got.ToDate.Format(time.DateOnly) != "2026-05-03" {
		t.Fatalf("to date = %s, want 2026-05-03", got.ToDate.Format(time.DateOnly))
	}
	if got.CreatedAccruals != 3 {
		t.Fatalf("created accruals = %d, want 3", got.CreatedAccruals)
	}
}

func TestInterestRuleServiceRecalculateRejectsInvalidRange(t *testing.T) {
	_, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule:     validAccrualTestRule(),
		FromDate: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestInterestRuleServiceRecalculateReplacesExistingAccruals(t *testing.T) {
	accruals := &recordingInterestAccrualRepo{replaceDeleted: 1}
	service := NewInterestRuleService(
		NewTransactionService(),
		WithInterestAccrualRepository(accruals),
	)
	rule := validAccrualTestRule()

	got, err := service.Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:          "initial",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInitialBalance,
				AmountMinor: 100_000_00,
				OccurredAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:          "old-interest",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInterestIncome,
				AmountMinor: 99_999,
				OccurredAt:  time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "old-interest",
				AccrualDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.DeletedAccruals != 1 {
		t.Fatalf("deleted accruals = %d, want 1", got.DeletedAccruals)
	}
	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1", got.CreatedAccruals)
	}
	if got.TotalAmountMinor != 3_288 {
		t.Fatalf("total amount = %d, want 3288", got.TotalAmountMinor)
	}
	if accruals.replaceCalls != 1 {
		t.Fatalf("replace calls = %d, want 1", accruals.replaceCalls)
	}
}

func TestInterestRuleServiceRecalculateSkipsNonPositiveBalanceDays(t *testing.T) {
	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: validAccrualTestRule(),
		Transactions: []models.Transaction{
			{
				ID:          "tx-1",
				AccountID:   "account-1",
				Type:        models.TransactionTypeInitialBalance,
				AmountMinor: 0,
				OccurredAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 0 {
		t.Fatalf("created accruals = %d, want 0", got.CreatedAccruals)
	}
	if got.SkippedDays != 1 {
		t.Fatalf("skipped days = %d, want 1", got.SkippedDays)
	}
}

type recordingInterestAccrualRepo struct {
	createCalls                int
	createWithTransactionCalls int
	replaceCalls               int
	replaceDeleted             int64
	transaction                *models.Transaction
	accrual                    *models.InterestAccrual
}

func (r *recordingInterestAccrualRepo) Create(context.Context, *models.InterestAccrual) error {
	r.createCalls++
	return nil
}

func (r *recordingInterestAccrualRepo) CreateWithTransaction(_ context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error {
	r.createWithTransactionCalls++
	r.transaction = transaction
	r.accrual = accrual
	return nil
}

func (r *recordingInterestAccrualRepo) ReplaceRangeWithTransactions(_ context.Context, _, _ string, _, _ time.Time, transactions []models.Transaction, accruals []models.InterestAccrual) (int64, error) {
	r.replaceCalls++
	if len(transactions) > 0 {
		r.transaction = &transactions[0]
	}
	if len(accruals) > 0 {
		r.accrual = &accruals[0]
	}
	return r.replaceDeleted, nil
}

func (r *recordingInterestAccrualRepo) GetByAccountDateRule(context.Context, string, string, string) (*models.InterestAccrual, error) {
	return nil, errNotImplemented
}

func (r *recordingInterestAccrualRepo) ListByAccount(context.Context, string) ([]models.InterestAccrual, error) {
	return nil, nil
}

type recordingTransactionRepo struct {
	createCalls int
}

func (r *recordingTransactionRepo) Create(context.Context, *models.Transaction) error {
	r.createCalls++
	return nil
}

func (r *recordingTransactionRepo) CreateMany(context.Context, []models.Transaction) error {
	return nil
}

func (r *recordingTransactionRepo) GetByID(context.Context, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (r *recordingTransactionRepo) List(context.Context) ([]models.Transaction, error) {
	return nil, nil
}

func (r *recordingTransactionRepo) ListByAccount(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *recordingTransactionRepo) Delete(context.Context, string) error {
	return nil
}

func TestInterestRuleServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *CreateInterestRuleRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "missing account id",
			req: &CreateInterestRuleRequest{
				AnnualRateBps: 1200,
			},
		},
		{
			name: "zero annual rate",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 0,
			},
		},
		{
			name: "negative promo rate",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoRateBps:  ptrInt64(-100),
				PromoEndDate:  ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name: "promo rate without promo end date",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoRateBps:  ptrInt64(1500),
			},
		},
		{
			name: "promo end date without promo rate",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoEndDate:  ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name: "invalid accrual frequency",
			req: &CreateInterestRuleRequest{
				AccountID:        "account-1",
				AnnualRateBps:    1200,
				AccrualFrequency: models.AccrualFrequency("weekly"),
			},
		},
		{
			name: "invalid capitalization frequency",
			req: &CreateInterestRuleRequest{
				AccountID:               "account-1",
				AnnualRateBps:           1200,
				CapitalizationFrequency: models.CapitalizationFrequency("yearly"),
			},
		},
		{
			name: "invalid day count convention",
			req: &CreateInterestRuleRequest{
				AccountID:          "account-1",
				AnnualRateBps:      1200,
				DayCountConvention: models.DayCountConvention("30_360"),
			},
		},
		{
			name: "end date before start date",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				StartDate:     time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
				EndDate:       ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name: "promo end date before start date",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoRateBps:  ptrInt64(1500),
				StartDate:     time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
				PromoEndDate:  ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewInterestRuleService(nil)

			_, err := service.Create(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestInterestRuleServiceAccrueReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *AccrueRuleInterestRequest
	}{
		{"nil request", nil},
		{
			name: "missing rule id",
			req: &AccrueRuleInterestRequest{
				Rule: models.InterestRule{
					AccountID:               "account-1",
					IsActive:                true,
					AnnualRateBps:           1200,
					AccrualFrequency:        models.AccrualFrequencyDaily,
					CapitalizationFrequency: models.CapitalizationFrequencyNone,
					DayCountConvention:      models.DayCountConventionActual365,
					StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
				},
				BalanceMinor: 100_000,
				AccrualDate:  time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "non-positive balance",
			req: &AccrueRuleInterestRequest{
				Rule:         validAccrualTestRule(),
				BalanceMinor: 0,
				AccrualDate:  time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "rule inactive",
			req: &AccrueRuleInterestRequest{
				Rule: func() models.InterestRule {
					rule := validAccrualTestRule()
					rule.IsActive = false
					return rule
				}(),
				BalanceMinor: 100_000,
				AccrualDate:  time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewInterestRuleService(NewTransactionService())

			_, err := service.Accrue(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestInterestRuleServiceRecalculateAllowsInactiveRuleCleanup(t *testing.T) {
	rule := validAccrualTestRule()
	rule.IsActive = false

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:          "initial",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInitialBalance,
				AmountMinor: 100_000_00,
				OccurredAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "old-interest",
				AccrualDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1", got.CreatedAccruals)
	}
}

func TestInterestRuleServiceRecalculateDoesNotUsePriorAccrualsWhenCapitalizationNone(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:          "initial",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInitialBalance,
				AmountMinor: 100_000_00,
				OccurredAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:          "prior-interest",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInterestIncome,
				AmountMinor: 3_288,
				OccurredAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "prior-interest",
				AccrualDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1", got.CreatedAccruals)
	}

	if len(got.Transactions) != 1 {
		t.Fatalf("transactions len = %d, want 1", len(got.Transactions))
	}

	if got.Transactions[0].AmountMinor != 3_288 {
		t.Fatalf("amount = %d, want 3288 without prior interest compounding", got.Transactions[0].AmountMinor)
	}
}

func TestInterestRuleServiceRecalculateCompoundsWhenCapitalizationDaily(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyDaily

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:          "initial",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInitialBalance,
				AmountMinor: 100_000_00,
				OccurredAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 3 {
		t.Fatalf("created accruals = %d, want 3", got.CreatedAccruals)
	}

	if got.Transactions[0].AmountMinor != 3_288 {
		t.Fatalf("day 1 amount = %d, want 3288", got.Transactions[0].AmountMinor)
	}
	if got.Transactions[1].AmountMinor <= got.Transactions[0].AmountMinor {
		t.Fatalf("day 2 amount = %d, want more than day 1 due to daily capitalization", got.Transactions[1].AmountMinor)
	}
}

func TestInterestRuleServiceRecalculateDoesNotCompoundWhenCapitalizationNone(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:          "initial",
				AccountID:   rule.AccountID,
				Type:        models.TransactionTypeInitialBalance,
				AmountMinor: 100_000_00,
				OccurredAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 3 {
		t.Fatalf("created accruals = %d, want 3", got.CreatedAccruals)
	}

	for _, tx := range got.Transactions {
		if tx.AmountMinor != 3_288 {
			t.Fatalf("amount = %d, want 3288 without compounding", tx.AmountMinor)
		}
	}
}

func validAccrualTestRule() models.InterestRule {
	return models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		IsActive:                true,
		AnnualRateBps:           1200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
