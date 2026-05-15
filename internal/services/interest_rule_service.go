package services

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
	Transactions     []models.Transaction
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

type ForecastRuleInterestRequest struct {
	Rule             models.InterestRule
	Transactions     []models.Transaction
	ExistingAccruals []models.InterestAccrual
	FromDate         time.Time
	Days             int
	Today            time.Time
}

type ForecastRuleInterestResponse struct {
	AccountID        string
	RuleID           string
	FromDate         time.Time
	ToDate           time.Time
	Days             int
	ProjectedMinor   int64
	ProjectedBalance int64
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

	if !shouldPostAccrual(&req.Rule, accrualDate) {
		return nil, validationError(fmt.Sprintf("interest rule is not payable on %s", accrualDate.Format(time.DateOnly)))
	}

	if hasInterestAccrual(req.ExistingAccruals, &req.Rule, accrualDate) {
		return &AccrueRuleInterestResponse{Skipped: true}, nil
	}

	periodStart := nextAccrualPeriodStart(&req.Rule, req.ExistingAccruals, accrualDate)
	calculationTransactions := PrincipalTransactionsForRuleAt(req.Transactions, req.ExistingAccruals, &req.Rule, accrualDate)
	amountMinor, balanceMinor, rateBps, err := calculateAccrualAmount(ctx, &req.Rule, calculationTransactions, req.BalanceMinor, periodStart, accrualDate)
	if err != nil {
		return nil, err
	}
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
		BalanceMinor:  balanceMinor,
		AnnualRateBps: rateBps,
		CreatedAt:     time.Now(),
	}

	if s.accruals != nil {
		if err := s.accruals.CreateWithTransaction(ctx, tx, accrual); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				return &AccrueRuleInterestResponse{Skipped: true}, nil
			}
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
	if err := validateRuleForRecalculation(&req.Rule); err != nil {
		return nil, err
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

	calculationFromDate := recalculationStartDate(&req.Rule, fromDate, toDate)
	workingTransactions := excludeAccrualTransactions(req.Transactions, req.ExistingAccruals, &req.Rule, fromDate, toDate)
	workingTransactions = PrincipalTransactionsForRuleAt(workingTransactions, req.ExistingAccruals, &req.Rule, calculationFromDate)
	response := &RecalculateRuleInterestResponse{
		AccountID: req.Rule.AccountID,
		RuleID:    req.Rule.ID,
		FromDate:  fromDate,
		ToDate:    toDate,
	}

	var pendingAmount int64
	var pendingBalance int64
	var pendingRate int64
	var pendingCapitalization []models.Transaction

	for day := calculationFromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
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
		} else {
			rateBps := effectiveRateBps(&req.Rule, day)
			amountMinor := calculateDailyInterestMinor(balance.BalanceMinor, rateBps, req.Rule.DayCountConvention, day)
			if amountMinor <= 0 {
				response.SkippedDays++
			} else {
				pendingAmount += amountMinor
				pendingBalance = balance.BalanceMinor
				pendingRate = rateBps
			}
		}

		if !shouldPostAccrual(&req.Rule, day) || pendingAmount <= 0 {
			continue
		}

		tx, err := buildTransaction(ctx, &CreateTransactionRequest{
			AccountID:   req.Rule.AccountID,
			Type:        models.TransactionTypeInterestIncome,
			AmountMinor: pendingAmount,
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
			AmountMinor:   pendingAmount,
			BalanceMinor:  pendingBalance,
			AnnualRateBps: pendingRate,
			CreatedAt:     time.Now(),
		}

		response.Transactions = append(response.Transactions, *tx)
		response.Accruals = append(response.Accruals, accrual)
		response.CreatedAccruals++
		response.TotalAmountMinor += pendingAmount

		switch {
		case req.Rule.CapitalizationFrequency == models.CapitalizationFrequencyDaily:
			workingTransactions = append(workingTransactions, *tx)
		case shouldCapitalizeOn(&req.Rule, day):
			pendingCapitalization = append(pendingCapitalization, *tx)
			workingTransactions = append(workingTransactions, pendingCapitalization...)
			pendingCapitalization = nil
		case req.Rule.CapitalizationFrequency != models.CapitalizationFrequencyNone &&
			req.Rule.CapitalizationFrequency != "":
			pendingCapitalization = append(pendingCapitalization, *tx)
		}
		pendingAmount = 0
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

func (s *InterestRuleService) Forecast(ctx context.Context, req *ForecastRuleInterestRequest) (*ForecastRuleInterestResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("forecast interest: %w", err)
	}
	if req == nil {
		return nil, validationError("forecast interest request is required")
	}
	if err := validateRuleForRecalculation(&req.Rule); err != nil {
		return nil, err
	}
	if req.Days <= 0 {
		return nil, validationError("forecast days must be positive")
	}

	fromDate := dateOnly(req.FromDate)
	if fromDate.IsZero() {
		fromDate = dateOnly(req.Today)
		if fromDate.IsZero() {
			fromDate = dateOnly(time.Now())
		}
	}
	toDate := fromDate.AddDate(0, 0, req.Days-1)

	forecastRule := req.Rule
	forecastRule.AccrualFrequency = models.AccrualFrequencyDaily
	result, err := NewInterestRuleService(nil).Recalculate(ctx, &RecalculateRuleInterestRequest{
		Rule:             forecastRule,
		Transactions:     req.Transactions,
		ExistingAccruals: req.ExistingAccruals,
		FromDate:         fromDate,
		ToDate:           toDate,
		Today:            req.Today,
	})
	if err != nil {
		return nil, err
	}

	balance, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
		AccountID:    req.Rule.AccountID,
		Transactions: append(transactionsUpToDate(PrincipalTransactionsForRuleAt(req.Transactions, req.ExistingAccruals, &req.Rule, toDate), toDate), result.Transactions...),
	})
	if err != nil {
		return nil, fmt.Errorf("calculate forecast balance: %w", err)
	}

	return &ForecastRuleInterestResponse{
		AccountID:        req.Rule.AccountID,
		RuleID:           req.Rule.ID,
		FromDate:         fromDate,
		ToDate:           toDate,
		Days:             req.Days,
		ProjectedMinor:   result.TotalAmountMinor,
		ProjectedBalance: balance.BalanceMinor,
		Accruals:         result.Accruals,
	}, nil
}

// PrincipalTransactionsForRule excludes this rule's uncapitalized accrual transactions from principal calculations.
func PrincipalTransactionsForRule(
	transactions []models.Transaction,
	accruals []models.InterestAccrual,
	rule *models.InterestRule,
) []models.Transaction {
	return PrincipalTransactionsForRuleAt(transactions, accruals, rule, time.Time{})
}

// PrincipalTransactionsForRuleAt excludes accrual transactions that are not capitalized by asOfDate.
func PrincipalTransactionsForRuleAt(
	transactions []models.Transaction,
	accruals []models.InterestAccrual,
	rule *models.InterestRule,
	asOfDate time.Time,
) []models.Transaction {
	asOfDate = dateOnly(asOfDate)
	capitalizedTransactionIDs := make(map[string]struct{})
	uncapitalizedTransactionIDs := make(map[string]struct{})

	for i := range accruals {
		accrual := &accruals[i]
		if accrual.AccountID != rule.AccountID || accrual.RuleID != rule.ID {
			continue
		}
		if accrualCapitalizedBy(rule, accrual.AccrualDate, asOfDate) {
			capitalizedTransactionIDs[accrual.TransactionID] = struct{}{}
			continue
		}
		uncapitalizedTransactionIDs[accrual.TransactionID] = struct{}{}
	}

	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if _, ok := capitalizedTransactionIDs[transactions[i].ID]; ok {
			filtered = append(filtered, transactions[i])
			continue
		}
		if _, ok := uncapitalizedTransactionIDs[transactions[i].ID]; ok {
			continue
		}
		filtered = append(filtered, transactions[i])
	}

	return filtered
}

func accrualCapitalizedBy(rule *models.InterestRule, accrualDate, asOfDate time.Time) bool {
	if asOfDate.IsZero() {
		return shouldCapitalizeOn(rule, accrualDate)
	}
	capitalizationDate, ok := capitalizationDateForAccrual(rule, accrualDate)
	return ok && !capitalizationDate.After(asOfDate)
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
	if !validCapitalizationFrequency(rule.CapitalizationFrequency) {
		return validationError(fmt.Sprintf("invalid capitalization frequency: %s", rule.CapitalizationFrequency))
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

func calculateAccrualAmount(ctx context.Context, rule *models.InterestRule, transactions []models.Transaction, balanceMinor int64, fromDate, toDate time.Time) (amountMinor, finalBalanceMinor, finalRateBps int64, err error) {
	if toDate.Before(fromDate) {
		return 0, 0, 0, validationError("accrual period is empty")
	}
	if len(transactions) == 0 {
		if balanceMinor <= 0 {
			return 0, 0, 0, validationError("balance must be positive")
		}
		rateBps := effectiveRateBps(rule, toDate)
		return calculateDailyInterestMinor(balanceMinor, rateBps, rule.DayCountConvention, toDate), balanceMinor, rateBps, nil
	}

	var total int64
	var lastBalance int64
	var lastRate int64
	for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		select {
		case <-ctx.Done():
			return 0, 0, 0, fmt.Errorf("calculate accrual amount: %w", ctx.Err())
		default:
		}

		if !ruleActiveOn(rule, day) {
			continue
		}
		balance, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
			AccountID:    rule.AccountID,
			Transactions: transactionsUpToDate(transactions, day),
		})
		if err != nil {
			return 0, 0, 0, fmt.Errorf("calculate accrual balance: %w", err)
		}
		if balance.BalanceMinor <= 0 {
			continue
		}
		rateBps := effectiveRateBps(rule, day)
		total += calculateDailyInterestMinor(balance.BalanceMinor, rateBps, rule.DayCountConvention, day)
		lastBalance = balance.BalanceMinor
		lastRate = rateBps
	}
	return total, lastBalance, lastRate, nil
}

func calculateDailyInterestMinor(balanceMinor, rateBps int64, convention models.DayCountConvention, date time.Time) int64 {
	amount := decimal.NewFromInt(balanceMinor)
	rate := decimal.NewFromInt(rateBps).Div(decimal.NewFromInt(10_000))
	days := decimal.NewFromInt(int64(daysInYear(convention, date)))

	return amount.Mul(rate).Div(days).Round(0).IntPart()
}

func nextAccrualPeriodStart(rule *models.InterestRule, accruals []models.InterestAccrual, accrualDate time.Time) time.Time {
	if rule.AccrualFrequency == models.AccrualFrequencyDaily {
		return dateOnly(accrualDate)
	}

	start := dateOnly(rule.StartDate)
	for i := range accruals {
		accrual := &accruals[i]
		if accrual.AccountID != rule.AccountID || accrual.RuleID != rule.ID {
			continue
		}
		date := dateOnly(accrual.AccrualDate)
		if date.Before(accrualDate) && !date.Before(start) {
			start = date.AddDate(0, 0, 1)
		}
	}
	return start
}

func recalculationStartDate(rule *models.InterestRule, fromDate, toDate time.Time) time.Time {
	fromDate = dateOnly(fromDate)
	toDate = dateOnly(toDate)
	if rule.AccrualFrequency == models.AccrualFrequencyDaily {
		return fromDate
	}
	for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		if shouldPostAccrual(rule, day) {
			return minDate(fromDate, accrualPeriodStart(rule, day))
		}
	}
	return fromDate
}

func accrualPeriodStart(rule *models.InterestRule, date time.Time) time.Time {
	date = dateOnly(date)
	start := dateOnly(rule.StartDate)
	switch rule.AccrualFrequency {
	case models.AccrualFrequencyMonthly:
		monthStart := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
		return maxDate(start, monthStart)
	case models.AccrualFrequencyEndOfTerm:
		return start
	default:
		return date
	}
}

func capitalizationDateForAccrual(rule *models.InterestRule, accrualDate time.Time) (time.Time, bool) {
	accrualDate = dateOnly(accrualDate)
	if !ruleActiveOn(rule, accrualDate) {
		return time.Time{}, false
	}
	switch rule.CapitalizationFrequency {
	case models.CapitalizationFrequencyDaily:
		return accrualDate, true
	case models.CapitalizationFrequencyMonthly:
		return lastActiveDayOfMonth(rule, accrualDate), true
	case models.CapitalizationFrequencyEndOfTerm:
		if rule.EndDate == nil {
			return time.Time{}, false
		}
		return dateOnly(*rule.EndDate), true
	default:
		return time.Time{}, false
	}
}

func shouldPostAccrual(rule *models.InterestRule, date time.Time) bool {
	date = dateOnly(date)
	if !ruleActiveOn(rule, date) {
		return false
	}

	switch rule.AccrualFrequency {
	case models.AccrualFrequencyMonthly:
		return isLastActiveDayOfMonth(rule, date)
	case models.AccrualFrequencyEndOfTerm:
		return isEndOfTerm(rule, date)
	default:
		return true
	}
}

func shouldCapitalizeOn(rule *models.InterestRule, date time.Time) bool {
	switch rule.CapitalizationFrequency {
	case models.CapitalizationFrequencyDaily:
		return true
	case models.CapitalizationFrequencyMonthly:
		return isLastActiveDayOfMonth(rule, date)
	case models.CapitalizationFrequencyEndOfTerm:
		return isEndOfTerm(rule, date)
	default:
		return false
	}
}

func isLastActiveDayOfMonth(rule *models.InterestRule, date time.Time) bool {
	return lastActiveDayOfMonth(rule, date).Equal(dateOnly(date))
}

func lastActiveDayOfMonth(rule *models.InterestRule, date time.Time) time.Time {
	date = dateOnly(date)
	monthEnd := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	if rule.EndDate != nil {
		return minDate(monthEnd, dateOnly(*rule.EndDate))
	}
	return monthEnd
}

func isEndOfTerm(rule *models.InterestRule, date time.Time) bool {
	return rule.EndDate != nil && dateOnly(*rule.EndDate).Equal(dateOnly(date))
}

func minDate(a, b time.Time) time.Time {
	a = dateOnly(a)
	b = dateOnly(b)
	if a.Before(b) {
		return a
	}
	return b
}

func maxDate(a, b time.Time) time.Time {
	a = dateOnly(a)
	b = dateOnly(b)
	if a.After(b) {
		return a
	}
	return b
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
	case "",
		models.CapitalizationFrequencyDaily,
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

func validateRuleForRecalculation(rule *models.InterestRule) error {
	if strings.TrimSpace(rule.ID) == "" {
		return validationError("interest rule id is required")
	}
	if strings.TrimSpace(rule.AccountID) == "" {
		return validationError("account id is required")
	}
	if rule.AnnualRateBps <= 0 {
		return validationError("annual rate must be positive")
	}
	if !validAccrualFrequency(rule.AccrualFrequency) {
		return validationError(fmt.Sprintf("invalid accrual frequency: %s", rule.AccrualFrequency))
	}
	if !validCapitalizationFrequency(rule.CapitalizationFrequency) {
		return validationError(fmt.Sprintf("invalid capitalization frequency: %s", rule.CapitalizationFrequency))
	}
	if !validDayCountConvention(rule.DayCountConvention) {
		return validationError(fmt.Sprintf("invalid day count convention: %s", rule.DayCountConvention))
	}
	return nil
}
