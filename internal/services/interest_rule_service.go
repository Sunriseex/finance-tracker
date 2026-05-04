package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/sunriseex/finance-manager/internal/models"
)

type InterestRuleService struct {
	transactions *TransactionService
}

func NewInterestRuleService(transactions *TransactionService) *InterestRuleService {
	if transactions == nil {
		transactions = NewTransactionService()
	}
	return &InterestRuleService{transactions: transactions}
}

type CreateInterestRuleRequest struct {
	AccountID               string
	AnnualRateBps           int64
	PromoRateBps            *int64
	PromoEndDate            *time.Time
	AccrualFrequency        models.AccrualFrequency
	CapitalizationFrequency models.CapitalizationFrequency
	DayCountConvention      models.DayCountConvention
	StartDate               time.Time
	EndDate                 *time.Time
}

type AccrueRuleInterestRequest struct {
	Rule                 models.InterestRule
	BalanceMinor         int64
	AccrualDate          time.Time
	ExistingTransactions []models.Transaction
}

type AccrueRuleInterestResponse struct {
	Transaction *models.Transaction
	Skipped     bool
}

func (s *InterestRuleService) Create(ctx context.Context, req CreateInterestRuleRequest) (*models.InterestRule, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create interest rule: %w", ctx.Err())
	default:
	}

	accountID := strings.TrimSpace(req.AccountID)
	if accountID == "" {
		return nil, fmt.Errorf("account id is required")
	}
	if req.AnnualRateBps <= 0 {
		return nil, fmt.Errorf("annual rate must be positive")
	}
	if req.PromoRateBps != nil && *req.PromoRateBps <= 0 {
		return nil, fmt.Errorf("promo rate must be positive")
	}

	accrualFrequency := req.AccrualFrequency
	if accrualFrequency == "" {
		accrualFrequency = models.AccrualFrequencyDaily
	}
	if !validAccrualFrequency(accrualFrequency) {
		return nil, fmt.Errorf("invalid accrual frequency: %s", accrualFrequency)
	}

	capitalizationFrequency := req.CapitalizationFrequency
	if capitalizationFrequency == "" {
		capitalizationFrequency = models.CapitalizationFrequencyNone
	}
	if !validCapitalizationFrequency(capitalizationFrequency) {
		return nil, fmt.Errorf("invalid capitalization frequency: %s", capitalizationFrequency)
	}

	dayCountConvention := req.DayCountConvention
	if dayCountConvention == "" {
		dayCountConvention = models.DayCountConventionActual365
	}
	if !validDayCountConvention(dayCountConvention) {
		return nil, fmt.Errorf("invalid day count convention: %s", dayCountConvention)
	}

	startDate := dateOnly(req.StartDate)
	if startDate.IsZero() {
		startDate = dateOnly(time.Now())
	}
	if req.EndDate != nil && dateOnly(*req.EndDate).Before(startDate) {
		return nil, fmt.Errorf("end date must be on or after start date")
	}
	if req.PromoEndDate != nil && dateOnly(*req.PromoEndDate).Before(startDate) {
		return nil, fmt.Errorf("promo end date must be on or after start date")
	}

	return &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               accountID,
		AnnualRateBps:           req.AnnualRateBps,
		PromoRateBps:            req.PromoRateBps,
		PromoEndDate:            req.PromoEndDate,
		AccrualFrequency:        accrualFrequency,
		CapitalizationFrequency: capitalizationFrequency,
		DayCountConvention:      dayCountConvention,
		IsActive:                true,
		StartDate:               startDate,
		EndDate:                 req.EndDate,
	}, nil
}

func (s *InterestRuleService) Accrue(ctx context.Context, req AccrueRuleInterestRequest) (*AccrueRuleInterestResponse, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("accrue interest: %w", ctx.Err())
	default:
	}

	if err := validateRuleForAccrual(req.Rule); err != nil {
		return nil, err
	}
	if req.BalanceMinor <= 0 {
		return nil, fmt.Errorf("balance must be positive")
	}

	accrualDate := dateOnly(req.AccrualDate)
	if accrualDate.IsZero() {
		accrualDate = dateOnly(time.Now())
	}
	if !ruleActiveOn(req.Rule, accrualDate) {
		return nil, fmt.Errorf("interest rule is not active on %s", accrualDate.Format(time.DateOnly))
	}
	if req.Rule.AccrualFrequency != models.AccrualFrequencyDaily {
		return nil, fmt.Errorf("unsupported accrual frequency for manual accrual: %s", req.Rule.AccrualFrequency)
	}
	if hasInterestAccrual(req.ExistingTransactions, req.Rule, accrualDate) {
		return &AccrueRuleInterestResponse{Skipped: true}, nil
	}

	amountMinor := calculateDailyInterestMinor(req.BalanceMinor, effectiveRateBps(req.Rule, accrualDate), req.Rule.DayCountConvention, accrualDate)
	if amountMinor <= 0 {
		return nil, fmt.Errorf("calculated interest is zero")
	}

	tx, err := s.transactions.Create(ctx, CreateTransactionRequest{
		AccountID:   req.Rule.AccountID,
		Type:        models.TransactionTypeInterestIncome,
		AmountMinor: amountMinor,
		Description: interestAccrualDescription(req.Rule.ID, accrualDate),
		OccurredAt:  accrualDate,
	})
	if err != nil {
		return nil, fmt.Errorf("create interest income transaction: %w", err)
	}

	return &AccrueRuleInterestResponse{Transaction: tx}, nil
}

func validateRuleForAccrual(rule models.InterestRule) error {
	if strings.TrimSpace(rule.ID) == "" {
		return fmt.Errorf("interest rule id is required")
	}
	if strings.TrimSpace(rule.AccountID) == "" {
		return fmt.Errorf("account id is required")
	}
	if !rule.IsActive {
		return fmt.Errorf("interest rule is inactive")
	}
	if rule.AnnualRateBps <= 0 {
		return fmt.Errorf("annual rate must be positive")
	}
	if !validAccrualFrequency(rule.AccrualFrequency) {
		return fmt.Errorf("invalid accrual frequency: %s", rule.AccrualFrequency)
	}
	if !validDayCountConvention(rule.DayCountConvention) {
		return fmt.Errorf("invalid day count convention: %s", rule.DayCountConvention)
	}
	return nil
}

func calculateDailyInterestMinor(balanceMinor, rateBps int64, convention models.DayCountConvention, date time.Time) int64 {
	amount := decimal.NewFromInt(balanceMinor)
	rate := decimal.NewFromInt(rateBps).Div(decimal.NewFromInt(10_000))
	days := decimal.NewFromInt(int64(daysInYear(convention, date)))

	return amount.Mul(rate).Div(days).Round(0).IntPart()
}

func daysInYear(convention models.DayCountConvention, date time.Time) int {
	switch convention {
	case models.DayCountConventionActual366:
		return 366
	case models.DayCountConventionActualActual:
		if isLeapYear(date.Year()) {
			return 366
		}
		return 365
	default:
		return 365
	}
}

func effectiveRateBps(rule models.InterestRule, date time.Time) int64 {
	if rule.PromoRateBps != nil && rule.PromoEndDate != nil && !date.After(dateOnly(*rule.PromoEndDate)) {
		return *rule.PromoRateBps
	}
	return rule.AnnualRateBps
}

func ruleActiveOn(rule models.InterestRule, date time.Time) bool {
	if date.Before(dateOnly(rule.StartDate)) {
		return false
	}
	if rule.EndDate != nil && date.After(dateOnly(*rule.EndDate)) {
		return false
	}
	return true
}

func hasInterestAccrual(transactions []models.Transaction, rule models.InterestRule, date time.Time) bool {
	description := interestAccrualDescription(rule.ID, date)
	for _, tx := range transactions {
		if tx.AccountID == rule.AccountID &&
			tx.Type == models.TransactionTypeInterestIncome &&
			dateOnly(tx.OccurredAt).Equal(date) &&
			tx.Description == description {
			return true
		}
	}
	return false
}

func interestAccrualDescription(ruleID string, date time.Time) string {
	return fmt.Sprintf("interest accrual rule=%s date=%s", ruleID, date.Format(time.DateOnly))
}

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func validAccrualFrequency(frequency models.AccrualFrequency) bool {
	switch frequency {
	case models.AccrualFrequencyDaily,
		models.AccrualFrequencyMonthly,
		models.AccrualFrequencyEndOfTerm:
		return true
	default:
		return false
	}
}

func validCapitalizationFrequency(frequency models.CapitalizationFrequency) bool {
	switch frequency {
	case models.CapitalizationFrequencyDaily,
		models.CapitalizationFrequencyMonthly,
		models.CapitalizationFrequencyEndOfTerm,
		models.CapitalizationFrequencyNone:
		return true
	default:
		return false
	}
}

func validDayCountConvention(convention models.DayCountConvention) bool {
	switch convention {
	case models.DayCountConventionActual365,
		models.DayCountConventionActual366,
		models.DayCountConventionActualActual:
		return true
	default:
		return false
	}
}
