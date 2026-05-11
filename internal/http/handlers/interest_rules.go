package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) listInterestRules(w http.ResponseWriter, r *http.Request) {
	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if !h.ensureAccountExists(w, r, accountID) {
		return
	}

	rules, err := h.store.InterestRules().ListByAccount(r.Context(), accountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.InterestRulesFromModels(rules))
}

func (h *Handler) createInterestRule(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateInterestRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	startDate, err := parseOptionalDate(req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	promoEndDate, err := parseOptionalDatePtr(req.PromoEndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	endDate, err := parseOptionalDatePtr(req.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if !h.ensureAccountExists(w, r, accountID) {
		return
	}

	service := services.NewInterestRuleService(
		services.NewTransactionService(h.store.Transactions()),
		services.WithInterestRuleRepository(h.store.InterestRules()),
		services.WithInterestAccrualRepository(h.store.InterestAccruals()),
	)
	rule, err := service.Create(r.Context(), &services.CreateInterestRuleRequest{
		AccountID:               accountID,
		AnnualRateBps:           req.AnnualRateBps,
		PromoRateBps:            req.PromoRateBps,
		PromoEndDate:            promoEndDate,
		AccrualFrequency:        req.AccrualFrequency,
		CapitalizationFrequency: req.CapitalizationFrequency,
		DayCountConvention:      req.DayCountConvention,
		StartDate:               startDate,
		EndDate:                 endDate,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.InterestRuleFromModel(rule))
}

func (h *Handler) updateInterestRule(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	ruleID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	rule, err := h.store.InterestRules().GetByID(r.Context(), ruleID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if _, err := h.store.Accounts().GetByIDForUser(r.Context(), rule.AccountID, userID); err != nil {
		writeServiceError(w, err)
		return
	}

	var req dto.UpdateInterestRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	if req.AnnualRateBps != nil {
		rule.AnnualRateBps = *req.AnnualRateBps
	}
	if req.PromoRateBps.Set {
		if !req.PromoRateBps.Valid {
			rule.PromoRateBps = nil
			rule.PromoEndDate = nil
		} else {
			promoRate := req.PromoRateBps.Value
			rule.PromoRateBps = &promoRate
		}
	}

	if req.PromoEndDate.Set {
		if !req.PromoEndDate.Valid || strings.TrimSpace(req.PromoEndDate.Value) == "" {
			rule.PromoEndDate = nil
			rule.PromoRateBps = nil
		} else {
			date, err := parseOptionalDate(req.PromoEndDate.Value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
				return
			}
			rule.PromoEndDate = &date
		}
	}

	if req.AccrualFrequency != nil {
		rule.AccrualFrequency = *req.AccrualFrequency
	}
	if req.CapitalizationFrequency != nil {
		rule.CapitalizationFrequency = *req.CapitalizationFrequency
	}
	if req.DayCountConvention != nil {
		rule.DayCountConvention = *req.DayCountConvention
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	if req.StartDate != nil {
		date, err := parseOptionalDate(*req.StartDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}
		if !date.IsZero() {
			rule.StartDate = date
		}
	}
	if req.EndDate.Set {
		if !req.EndDate.Valid || strings.TrimSpace(req.EndDate.Value) == "" {
			rule.EndDate = nil
		} else {
			date, err := parseOptionalDate(req.EndDate.Value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
				return
			}
			rule.EndDate = &date
		}
	}

	if err := validateInterestRule(rule); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	if err := h.store.InterestRules().Update(r.Context(), rule); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.InterestRuleFromModel(rule))
}

func (h *Handler) accrueInterest(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if !h.ensureAccountExists(w, r, accountID) {
		return
	}
	var req dto.AccrueInterestRequest

	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	accrualDate, err := parseOptionalDate(req.Date)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	rule, err := h.ruleForAccrual(r, accountID, req.RuleID, accrualDate)
	if err != nil {
		if _, ok := err.(validationError); ok {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}

		writeServiceError(w, err)
		return
	}
	transactions, err := h.store.Transactions().ListByAccountForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	accruals, err := h.store.InterestAccruals().ListByAccount(r.Context(), accountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	transactions = transactionsUpToDate(transactions, accrualDate)
	if rule.CapitalizationFrequency == models.CapitalizationFrequencyNone ||
		rule.CapitalizationFrequency == "" {
		transactions = excludeRuleAccrualTransactions(transactions, accruals, rule)
	}

	balance, err := services.NewBalanceService().Calculate(r.Context(), services.CalculateBalanceRequest{
		AccountID:    accountID,
		Transactions: transactions,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	service := services.NewInterestRuleService(
		services.NewTransactionService(h.store.Transactions()),
		services.WithInterestAccrualRepository(h.store.InterestAccruals()),
	)
	result, err := service.Accrue(r.Context(), &services.AccrueRuleInterestRequest{
		Rule:             *rule,
		BalanceMinor:     balance.BalanceMinor,
		AccrualDate:      accrualDate,
		ExistingAccruals: accruals,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}
	if result.Skipped {
		writeJSON(w, http.StatusOK, map[string]any{"skipped": true})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"skipped":     false,
		"transaction": dto.TransactionFromModel(result.Transaction),
		"accrual":     result.Accrual,
	})
}

func (h *Handler) recalculateInterest(w http.ResponseWriter, r *http.Request) {
	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req dto.RecalculateInterestRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	if !validateOptionalUUID(w, req.RuleID, "rule_id") {
		return
	}

	fromDate, err := parseOptionalDate(req.FromDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	toDate, err := parseOptionalDate(req.ToDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	if !h.ensureAccountExists(w, r, accountID) {
		return
	}

	ruleDate := toDate
	if ruleDate.IsZero() {
		ruleDate = dateOnly(time.Now())
	}

	rule, err := h.ruleForRecalculation(r, accountID, req.RuleID, ruleDate)
	if err != nil {
		if _, ok := err.(validationError); ok {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}

		writeServiceError(w, err)
		return
	}

	transactions, err := h.store.Transactions().ListByAccountForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	accruals, err := h.store.InterestAccruals().ListByAccount(r.Context(), accountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	service := services.NewInterestRuleService(
		services.NewTransactionService(h.store.Transactions()),
		services.WithInterestAccrualRepository(h.store.InterestAccruals()),
	)
	result, err := service.Recalculate(r.Context(), &services.RecalculateRuleInterestRequest{
		Rule:             *rule,
		Transactions:     transactions,
		ExistingAccruals: accruals,
		FromDate:         fromDate,
		ToDate:           toDate,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.RecalculateInterestResponse{
		AccountID:        result.AccountID,
		RuleID:           result.RuleID,
		FromDate:         result.FromDate,
		ToDate:           result.ToDate,
		DeletedAccruals:  result.DeletedAccruals,
		CreatedAccruals:  result.CreatedAccruals,
		SkippedDays:      result.SkippedDays,
		TotalAmountMinor: result.TotalAmountMinor,
	})
}

func (h *Handler) ruleForAccrual(r *http.Request, accountID, ruleID string, accrualDate time.Time) (*models.InterestRule, error) {
	ruleID = strings.TrimSpace(ruleID)
	if ruleID != "" {
		if !isValidUUID(ruleID) {
			return nil, errValidation("invalid rule_id")
		}

		rule, err := h.store.InterestRules().GetByID(r.Context(), ruleID)
		if err != nil {
			return nil, fmt.Errorf("get interest rule: %w", err)
		}

		if err := ensureRuleBelongsToAccount(rule, accountID); err != nil {
			return nil, err
		}

		if !interestRuleActiveOn(rule, accrualDate) {
			return nil, errValidation("interest rule is not active on " + dateOnly(accrualDate).Format(time.DateOnly))
		}

		return rule, nil
	}

	rules, err := h.store.InterestRules().ListByAccount(r.Context(), accountID)
	if err != nil {
		return nil, fmt.Errorf("list account interest rules: %w", err)
	}

	rule := latestApplicableInterestRule(rules, accrualDate)
	if rule == nil {
		return nil, repository.ErrNotFound
	}

	return rule, nil
}

func (h *Handler) ruleForRecalculation(r *http.Request, accountID, ruleID string, fallbackDate time.Time) (*models.InterestRule, error) {
	ruleID = strings.TrimSpace(ruleID)
	if ruleID != "" {
		if !isValidUUID(ruleID) {
			return nil, errValidation("invalid rule_id")
		}

		rule, err := h.store.InterestRules().GetByID(r.Context(), ruleID)
		if err != nil {
			return nil, fmt.Errorf("get interest rule: %w", err)
		}

		if err := ensureRuleBelongsToAccount(rule, accountID); err != nil {
			return nil, err
		}

		return rule, nil
	}

	rules, err := h.store.InterestRules().ListByAccount(r.Context(), accountID)
	if err != nil {
		return nil, fmt.Errorf("list account interest rules: %w", err)
	}

	rule := latestApplicableInterestRule(rules, fallbackDate)
	if rule == nil {
		return nil, repository.ErrNotFound
	}

	return rule, nil
}

func validateInterestRule(rule *models.InterestRule) error {
	if rule.AnnualRateBps <= 0 {
		return errValidation("annual rate must be positive")
	}
	if rule.PromoRateBps != nil && *rule.PromoRateBps <= 0 {
		return errValidation("promo rate must be positive")
	}
	if (rule.PromoRateBps == nil) != (rule.PromoEndDate == nil) {
		return errValidation("promo rate and promo end date must be set together")
	}

	if rule.AccrualFrequency == "" {
		rule.AccrualFrequency = models.AccrualFrequencyDaily
	}
	if !validAccrualFrequency(rule.AccrualFrequency) {
		return errValidation("invalid accrual frequency: " + string(rule.AccrualFrequency))
	}

	if rule.CapitalizationFrequency == "" {
		rule.CapitalizationFrequency = models.CapitalizationFrequencyNone
	}
	if !validCapitalizationFrequency(rule.CapitalizationFrequency) {
		return errValidation("invalid capitalization frequency: " + string(rule.CapitalizationFrequency))
	}

	if rule.DayCountConvention == "" {
		rule.DayCountConvention = models.DayCountConventionActual365
	}
	if !validDayCountConvention(rule.DayCountConvention) {
		return errValidation("invalid day count convention: " + string(rule.DayCountConvention))
	}

	startDate := dateOnly(rule.StartDate)
	if startDate.IsZero() {
		return errValidation("start date is required")
	}

	if rule.EndDate != nil && dateOnly(*rule.EndDate).Before(startDate) {
		return errValidation("end date must be on or after start date")
	}

	if rule.PromoEndDate != nil && dateOnly(*rule.PromoEndDate).Before(startDate) {
		return errValidation("promo end date must be on or after start date")
	}

	return nil
}

func parseOptionalDatePtr(input *string) (*time.Time, error) {
	if input == nil {
		//nolint:nilnil // nil date pointer means optional date was not provided.
		return nil, nil
	}
	date, err := parseOptionalDate(*input)
	if err != nil {
		return nil, fmt.Errorf("parse optional date: %w", err)
	}
	if date.IsZero() {
		//nolint:nilnil // empty date string clears optional date.
		return nil, nil
	}
	return &date, nil
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

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func ensureRuleBelongsToAccount(rule *models.InterestRule, accountID string) error {
	if rule.AccountID != accountID {
		return repository.ErrNotFound
	}
	return nil
}

func latestApplicableInterestRule(rules []models.InterestRule, accrualDate time.Time) *models.InterestRule {
	var selected *models.InterestRule

	for i := range rules {
		rule := &rules[i]
		if !rule.IsActive || !interestRuleActiveOn(rule, accrualDate) {
			continue
		}

		if selected == nil || dateOnly(rule.StartDate).After(dateOnly(selected.StartDate)) {
			selected = rule
		}
	}

	return selected
}

func interestRuleActiveOn(rule *models.InterestRule, accrualDate time.Time) bool {
	date := dateOnly(accrualDate)
	if date.IsZero() {
		date = dateOnly(time.Now())
	}

	if date.Before(dateOnly(rule.StartDate)) {
		return false
	}

	if rule.EndDate != nil && date.After(dateOnly(*rule.EndDate)) {
		return false
	}

	return true
}

func transactionsUpToDate(transactions []models.Transaction, accrualDate time.Time) []models.Transaction {
	date := dateOnly(accrualDate)
	if date.IsZero() {
		date = dateOnly(time.Now())
	}

	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if !dateOnly(transactions[i].OccurredAt).After(date) {
			filtered = append(filtered, transactions[i])
		}
	}

	return filtered
}

func excludeRuleAccrualTransactions(
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
