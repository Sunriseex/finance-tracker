package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/repository"
)

type InterestRuleService struct {
	transactions *TransactionService
	rules        repository.InterestRuleRepository
	accruals     repository.InterestAccrualRepository
}

type InterestRuleServiceOption func(*InterestRuleService)

func WithInterestRuleRepository(repo repository.InterestRuleRepository) InterestRuleServiceOption {
	return func(s *InterestRuleService) {
		s.rules = repo
	}
}

func WithInterestAccrualRepository(repo repository.InterestAccrualRepository) InterestRuleServiceOption {
	return func(s *InterestRuleService) {
		s.accruals = repo
	}
}

func NewInterestRuleService(transactions *TransactionService, options ...InterestRuleServiceOption) *InterestRuleService {
	if transactions == nil {
		transactions = NewTransactionService()
	}
	service := &InterestRuleService{transactions: transactions}
	for _, option := range options {
		option(service)
	}
	return service
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
	Rule             models.InterestRule
	BalanceMinor     int64
	AccrualDate      time.Time
	ExistingAccruals []models.InterestAccrual
}

type AccrueRuleInterestResponse struct {
	Transaction *models.Transaction
	Accrual     *models.InterestAccrual
	Skipped     bool
}

type RecalculateRuleInterestRequest struct {
	Rule             models.InterestRule
	Transactions     []models.Transaction
	ExistingAccruals []models.InterestAccrual
	FromDate         time.Time
	ToDate           time.Time
	Today            time.Time
}

type RecalculateRuleInterestResponse struct {
	AccountID        string
	RuleID           string
	FromDate         time.Time
	ToDate           time.Time
	DeletedAccruals  int64
	CreatedAccruals  int64
	SkippedDays      int64
	TotalAmountMinor int64
	Transactions     []models.Transaction
	Accruals         []models.InterestAccrual
}

func (s *InterestRuleService) Create(ctx context.Context, req *CreateInterestRuleRequest) (*models.InterestRule, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create interest rule: %w", ctx.Err())
	default:
	}
	if req == nil {
		return nil, validationError("create interest rule request is required")
	}

	accountID := strings.TrimSpace(req.AccountID)
	if accountID == "" {
		return nil, validationError("account id is required")
	}
	if req.AnnualRateBps <= 0 {
		return nil, validationError("annual rate must be positive")
	}
	if req.PromoRateBps != nil && *req.PromoRateBps <= 0 {
		return nil, validationError("promo rate must be positive")
	}
	if req.PromoRateBps != nil && req.PromoEndDate == nil {
		return nil, validationError("promo end date is required when promo rate is set")
	}
	if req.PromoRateBps == nil && req.PromoEndDate != nil {
		return nil, validationError("promo rate is required when promo end date is set")
	}

	accrualFrequency := req.AccrualFrequency
	if accrualFrequency == "" {
		accrualFrequency = models.AccrualFrequencyDaily
	}
	if !validAccrualFrequency(accrualFrequency) {
		return nil, validationError(fmt.Sprintf("invalid accrual frequency: %s", accrualFrequency))
	}

	capitalizationFrequency := req.CapitalizationFrequency
	if capitalizationFrequency == "" {
		capitalizationFrequency = models.CapitalizationFrequencyNone
	}
	if !validCapitalizationFrequency(capitalizationFrequency) {
		return nil, validationError(fmt.Sprintf("invalid capitalization frequency: %s", capitalizationFrequency))
	}

	dayCountConvention := req.DayCountConvention
	if dayCountConvention == "" {
		dayCountConvention = models.DayCountConventionActual365
	}
	if !validDayCountConvention(dayCountConvention) {
		return nil, validationError(fmt.Sprintf("invalid day count convention: %s", dayCountConvention))
	}

	startDate := dateOnly(req.StartDate)
	if startDate.IsZero() {
		startDate = dateOnly(time.Now())
	}
	if req.EndDate != nil && dateOnly(*req.EndDate).Before(startDate) {
		return nil, validationError("end date must be on or after start date")
	}
	if req.PromoEndDate != nil && dateOnly(*req.PromoEndDate).Before(startDate) {
		return nil, validationError("promo end date must be on or after start date")
	}

	var endDate *time.Time
	if req.EndDate != nil {
		normalized := dateOnly(*req.EndDate)
		endDate = &normalized
	}

	var promoEndDate *time.Time
	if req.PromoEndDate != nil {
		normalized := dateOnly(*req.PromoEndDate)
		promoEndDate = &normalized
	}

	rule := &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               accountID,
		AnnualRateBps:           req.AnnualRateBps,
		PromoRateBps:            req.PromoRateBps,
		PromoEndDate:            promoEndDate,
		AccrualFrequency:        accrualFrequency,
		CapitalizationFrequency: capitalizationFrequency,
		DayCountConvention:      dayCountConvention,
		IsActive:                true,
		StartDate:               startDate,
		EndDate:                 endDate,
	}

	if s.rules != nil {
		if err := s.rules.Create(ctx, rule); err != nil {
			return nil, fmt.Errorf("save interest rule: %w", err)
		}
	}

	return rule, nil
}

func (s *InterestRuleService) Accrue(ctx context.Context, req *AccrueRuleInterestRequest) (*AccrueRuleInterestResponse, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("accrue interest: %w", ctx.Err())
	default:
	}

	if req == nil {
		return nil, validationError("accrue interest request is required")
	}

	if err := validateRuleForAccrual(&req.Rule); err != nil {
		return nil, err
	}

	accrualDate := dateOnly(req.AccrualDate)
	if accrualDate.IsZero() {
		accrualDate = dateOnly(time.Now())
	}

	if !ruleActiveOn(&req.Rule, accrualDate) {
		return nil, validationError(fmt.Sprintf("interest rule is not active on %s", accrualDate.Format(time.DateOnly)))
	}

	if req.Rule.AccrualFrequency != models.AccrualFrequencyDaily {
		return nil, validationError(fmt.Sprintf("unsupported accrual frequency for manual accrual: %s", req.Rule.AccrualFrequency))
	}

	if req.Rule.CapitalizationFrequency != models.CapitalizationFrequencyDaily &&
		req.Rule.CapitalizationFrequency != models.CapitalizationFrequencyNone &&
		req.Rule.CapitalizationFrequency != "" {
		return nil, validationError(fmt.Sprintf("unsupported capitalization frequency: %s", req.Rule.CapitalizationFrequency))
	}

	if hasInterestAccrual(req.ExistingAccruals, &req.Rule, accrualDate) {
		return &AccrueRuleInterestResponse{Skipped: true}, nil
	}

	if req.BalanceMinor <= 0 {
		return nil, validationError("balance must be positive")
	}

	amountMinor := calculateDailyInterestMinor(
		req.BalanceMinor,
		effectiveRateBps(&req.Rule, accrualDate),
		req.Rule.DayCountConvention,
		accrualDate,
	)
	if amountMinor <= 0 {
		return nil, validationError("calculated interest is zero")
	}

	tx, err := buildTransaction(ctx, &CreateTransactionRequest{
		AccountID:   req.Rule.AccountID,
		Type:        models.TransactionTypeInterestIncome,
		AmountMinor: amountMinor,
		Description: interestAccrualDescription(req.Rule.ID, accrualDate),
		OccurredAt:  accrualDate,
	})
	if err != nil {
		return nil, fmt.Errorf("build interest income transaction: %w", err)
	}

	accrual := &models.InterestAccrual{
		ID:            uuid.NewString(),
		AccountID:     req.Rule.AccountID,
		RuleID:        req.Rule.ID,
		TransactionID: tx.ID,
		AccrualDate:   accrualDate,
		AmountMinor:   amountMinor,
		BalanceMinor:  req.BalanceMinor,
		AnnualRateBps: effectiveRateBps(&req.Rule, accrualDate),
		CreatedAt:     time.Now(),
	}

	if s.accruals != nil {
		if err := s.accruals.CreateWithTransaction(ctx, tx, accrual); err != nil {
			return nil, fmt.Errorf("save interest accrual with transaction: %w", err)
		}
	} else if s.transactions.repo != nil {
		if err := s.transactions.repo.Create(ctx, tx); err != nil {
			return nil, fmt.Errorf("save interest transaction: %w", err)
		}
	}

	return &AccrueRuleInterestResponse{
		Transaction: tx,
		Accrual:     accrual,
	}, nil
}

func (s *InterestRuleService) Recalculate(ctx context.Context, req *RecalculateRuleInterestRequest) (*RecalculateRuleInterestResponse, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("recalculate interest: %w", ctx.Err())
	default:
	}

	if req == nil {
		return nil, validationError("recalculate interest request is required")
	}
	if err := validateRuleForAccrual(&req.Rule); err != nil {
		return nil, err
	}
	if req.Rule.AccrualFrequency != models.AccrualFrequencyDaily {
		return nil, validationError(fmt.Sprintf("unsupported accrual frequency for recalculation: %s", req.Rule.AccrualFrequency))
	}
	if req.Rule.CapitalizationFrequency != models.CapitalizationFrequencyDaily &&
		req.Rule.CapitalizationFrequency != models.CapitalizationFrequencyNone &&
		req.Rule.CapitalizationFrequency != "" {
		return nil, validationError(fmt.Sprintf("unsupported capitalization frequency: %s", req.Rule.CapitalizationFrequency))
	}

	fromDate := dateOnly(req.FromDate)
	if fromDate.IsZero() {
		fromDate = dateOnly(req.Rule.StartDate)
	}
	toDate := dateOnly(req.ToDate)
	if toDate.IsZero() {
		toDate = dateOnly(req.Today)
		if toDate.IsZero() {
			toDate = dateOnly(time.Now())
		}
	}
	if fromDate.IsZero() {
		return nil, validationError("from date is required")
	}
	if toDate.Before(fromDate) {
		return nil, validationError("to date must be on or after from date")
	}

	workingTransactions := excludeAccrualTransactions(req.Transactions, req.ExistingAccruals, &req.Rule, fromDate, toDate)

	if req.Rule.CapitalizationFrequency == models.CapitalizationFrequencyNone ||
		req.Rule.CapitalizationFrequency == "" {
		workingTransactions = excludeAllRuleAccrualTransactions(workingTransactions, req.ExistingAccruals, &req.Rule)
	}
	response := &RecalculateRuleInterestResponse{
		AccountID: req.Rule.AccountID,
		RuleID:    req.Rule.ID,
		FromDate:  fromDate,
		ToDate:    toDate,
	}

	for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("recalculate interest: %w", ctx.Err())
		default:
		}

		if !ruleActiveOn(&req.Rule, day) {
			response.SkippedDays++
			continue
		}

		balance, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
			AccountID:    req.Rule.AccountID,
			Transactions: transactionsUpToDate(workingTransactions, day),
		})
		if err != nil {
			return nil, fmt.Errorf("calculate balance for interest recalculation: %w", err)
		}
		if balance.BalanceMinor <= 0 {
			response.SkippedDays++
			continue
		}

		rateBps := effectiveRateBps(&req.Rule, day)
		amountMinor := calculateDailyInterestMinor(balance.BalanceMinor, rateBps, req.Rule.DayCountConvention, day)
		if amountMinor <= 0 {
			response.SkippedDays++
			continue
		}

		tx, err := buildTransaction(ctx, &CreateTransactionRequest{
			AccountID:   req.Rule.AccountID,
			Type:        models.TransactionTypeInterestIncome,
			AmountMinor: amountMinor,
			Description: interestAccrualDescription(req.Rule.ID, day),
			OccurredAt:  day,
		})
		if err != nil {
			return nil, fmt.Errorf("build recalculated interest transaction: %w", err)
		}

		accrual := models.InterestAccrual{
			ID:            uuid.NewString(),
			AccountID:     req.Rule.AccountID,
			RuleID:        req.Rule.ID,
			TransactionID: tx.ID,
			AccrualDate:   day,
			AmountMinor:   amountMinor,
			BalanceMinor:  balance.BalanceMinor,
			AnnualRateBps: rateBps,
			CreatedAt:     time.Now(),
		}

		response.Transactions = append(response.Transactions, *tx)
		response.Accruals = append(response.Accruals, accrual)
		response.CreatedAccruals++
		response.TotalAmountMinor += amountMinor

		if req.Rule.CapitalizationFrequency == models.CapitalizationFrequencyDaily {
			workingTransactions = append(workingTransactions, *tx)
		}
	}

	if s.accruals != nil {
		deleted, err := s.accruals.ReplaceRangeWithTransactions(ctx, req.Rule.AccountID, req.Rule.ID, fromDate, toDate, response.Transactions, response.Accruals)
		if err != nil {
			return nil, fmt.Errorf("replace recalculated interest accruals: %w", err)
		}
		response.DeletedAccruals = deleted
	}

	return response, nil
}

func excludeAllRuleAccrualTransactions(
	transactions []models.Transaction,
	accruals []models.InterestAccrual,
	rule *models.InterestRule,
) []models.Transaction {
	excludedTransactionIDs := make(map[string]struct{})

	for i := range accruals {
		accrual := &accruals[i]
		if accrual.AccountID == rule.AccountID && accrual.RuleID == rule.ID {
			excludedTransactionIDs[accrual.TransactionID] = struct{}{}
		}
	}

	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if _, ok := excludedTransactionIDs[transactions[i].ID]; ok {
			continue
		}

		filtered = append(filtered, transactions[i])
	}

	return filtered
}

func validateRuleForAccrual(rule *models.InterestRule) error {
	if strings.TrimSpace(rule.ID) == "" {
		return validationError("interest rule id is required")
	}
	if strings.TrimSpace(rule.AccountID) == "" {
		return validationError("account id is required")
	}
	if !rule.IsActive {
		return validationError("interest rule is inactive")
	}
	if rule.AnnualRateBps <= 0 {
		return validationError("annual rate must be positive")
	}
	if !validAccrualFrequency(rule.AccrualFrequency) {
		return validationError(fmt.Sprintf("invalid accrual frequency: %s", rule.AccrualFrequency))
	}
	if !validDayCountConvention(rule.DayCountConvention) {
		return validationError(fmt.Sprintf("invalid day count convention: %s", rule.DayCountConvention))
	}
	return nil
}

func excludeAccrualTransactions(transactions []models.Transaction, accruals []models.InterestAccrual, rule *models.InterestRule, fromDate, toDate time.Time) []models.Transaction {
	replacedTransactionIDs := make(map[string]struct{})
	for i := range accruals {
		accrual := &accruals[i]
		accrualDate := dateOnly(accrual.AccrualDate)
		if accrual.AccountID == rule.AccountID &&
			accrual.RuleID == rule.ID &&
			!accrualDate.Before(fromDate) &&
			!accrualDate.After(toDate) {
			replacedTransactionIDs[accrual.TransactionID] = struct{}{}
		}
	}

	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if _, ok := replacedTransactionIDs[transactions[i].ID]; ok {
			continue
		}
		filtered = append(filtered, transactions[i])
	}
	return filtered
}

func transactionsUpToDate(transactions []models.Transaction, date time.Time) []models.Transaction {
	date = dateOnly(date)
	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if !dateOnly(transactions[i].OccurredAt).After(date) {
			filtered = append(filtered, transactions[i])
		}
	}
	return filtered
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

func effectiveRateBps(rule *models.InterestRule, date time.Time) int64 {
	if rule.PromoRateBps != nil && rule.PromoEndDate != nil && !date.After(dateOnly(*rule.PromoEndDate)) {
		return *rule.PromoRateBps
	}
	return rule.AnnualRateBps
}

func ruleActiveOn(rule *models.InterestRule, date time.Time) bool {
	if date.Before(dateOnly(rule.StartDate)) {
		return false
	}
	if rule.EndDate != nil && date.After(dateOnly(*rule.EndDate)) {
		return false
	}
	return true
}

func hasInterestAccrual(accruals []models.InterestAccrual, rule *models.InterestRule, date time.Time) bool {
	for i := range accruals {
		accrual := &accruals[i]
		if accrual.AccountID == rule.AccountID &&
			accrual.RuleID == rule.ID &&
			dateOnly(accrual.AccrualDate).Equal(date) {
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
