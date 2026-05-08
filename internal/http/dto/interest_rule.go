package dto

import (
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

type InterestRuleResponse struct {
	ID                      string                         `json:"id"`
	AccountID               string                         `json:"account_id"`
	AnnualRateBps           int64                          `json:"annual_rate_bps"`
	PromoRateBps            *int64                         `json:"promo_rate_bps,omitempty"`
	PromoEndDate            *time.Time                     `json:"promo_end_date,omitempty"`
	AccrualFrequency        models.AccrualFrequency        `json:"accrual_frequency"`
	CapitalizationFrequency models.CapitalizationFrequency `json:"capitalization_frequency"`
	DayCountConvention      models.DayCountConvention      `json:"day_count_convention"`
	IsActive                bool                           `json:"is_active"`
	StartDate               time.Time                      `json:"start_date"`
	EndDate                 *time.Time                     `json:"end_date,omitempty"`
}

type CreateInterestRuleRequest struct {
	AnnualRateBps           int64                          `json:"annual_rate_bps"`
	PromoRateBps            *int64                         `json:"promo_rate_bps"`
	PromoEndDate            *string                        `json:"promo_end_date"`
	AccrualFrequency        models.AccrualFrequency        `json:"accrual_frequency"`
	CapitalizationFrequency models.CapitalizationFrequency `json:"capitalization_frequency"`
	DayCountConvention      models.DayCountConvention      `json:"day_count_convention"`
	StartDate               string                         `json:"start_date"`
	EndDate                 *string                        `json:"end_date"`
}

type UpdateInterestRuleRequest struct {
	AnnualRateBps           *int64                          `json:"annual_rate_bps"`
	PromoRateBps            NullableInt64                   `json:"promo_rate_bps"`
	PromoEndDate            NullableString                  `json:"promo_end_date"`
	AccrualFrequency        *models.AccrualFrequency        `json:"accrual_frequency"`
	CapitalizationFrequency *models.CapitalizationFrequency `json:"capitalization_frequency"`
	DayCountConvention      *models.DayCountConvention      `json:"day_count_convention"`
	IsActive                *bool                           `json:"is_active"`
	StartDate               *string                         `json:"start_date"`
	EndDate                 NullableString                  `json:"end_date"`
}

type AccrueInterestRequest struct {
	RuleID string `json:"rule_id"`
	Date   string `json:"date"`
}

func InterestRuleFromModel(rule *models.InterestRule) InterestRuleResponse {
	return InterestRuleResponse{
		ID:                      rule.ID,
		AccountID:               rule.AccountID,
		AnnualRateBps:           rule.AnnualRateBps,
		PromoRateBps:            rule.PromoRateBps,
		PromoEndDate:            rule.PromoEndDate,
		AccrualFrequency:        rule.AccrualFrequency,
		CapitalizationFrequency: rule.CapitalizationFrequency,
		DayCountConvention:      rule.DayCountConvention,
		IsActive:                rule.IsActive,
		StartDate:               rule.StartDate,
		EndDate:                 rule.EndDate,
	}
}

func InterestRulesFromModels(rules []models.InterestRule) []InterestRuleResponse {
	response := make([]InterestRuleResponse, 0, len(rules))
	for i := range rules {
		response = append(response, InterestRuleFromModel(&rules[i]))
	}
	return response
}
