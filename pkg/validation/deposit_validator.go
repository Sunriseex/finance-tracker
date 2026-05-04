package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

type DepositValidator struct {
	allowedBanks    map[string]bool
	allowedTypes    map[string]bool
	allowedCapTypes map[string]bool
	minAmount       int64
	maxAmount       int64
}

func NewDepositValidator() *DepositValidator {
	return &DepositValidator{
		allowedBanks: map[string]bool{
			"Яндекс Банк": true,
			"Yandex":      true,
			"Альфа Банк":  true,
			"Alfa":        true,
			"Тинькофф":    true,
			"Tinkoff":     true,
			"Сбербанк":    true,
			"Sberbank":    true,
			"ВТБ":         true,
			"VTB":         true,
		},
		allowedTypes: map[string]bool{
			"savings": true,
			"term":    true,
		},
		allowedCapTypes: map[string]bool{
			"daily":     true,
			"monthly":   true,
			"end":       true,
			"quarterly": true,
		},
		minAmount: 1000,
		maxAmount: 100000000,
	}
}

func (v *DepositValidator) Validate(deposit *models.Deposit) error {
	var errors []string

	if err := v.validateBasicFields(deposit, &errors); err != nil {
		return err
	}

	if err := v.validateFinancialFields(deposit, &errors); err != nil {
		return err
	}

	if err := v.validateDates(deposit, &errors); err != nil {
		return err
	}

	if err := v.validateBusinessRules(deposit, &errors); err != nil {
		return err
	}

	if len(errors) > 0 {
		return fmt.Errorf("deposit validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (v *DepositValidator) validateBasicFields(deposit *models.Deposit, errors *[]string) error {
	if strings.TrimSpace(deposit.Name) == "" {
		*errors = append(*errors, "deposit name cannot be empty")
	}

	if len(deposit.Name) > 100 {
		*errors = append(*errors, "deposit name too long (max 100 characters)")
	}

	if !v.allowedBanks[deposit.Bank] {
		*errors = append(*errors, fmt.Sprintf("invalid bank: %s. Allowed banks: %v",
			deposit.Bank, v.getAllowedBanksList()))
	}

	if !v.allowedTypes[deposit.Type] {
		*errors = append(*errors, fmt.Sprintf("invalid type: %s. Allowed types: savings, term", deposit.Type))
	}

	if !v.allowedCapTypes[deposit.Capitalization] {
		*errors = append(*errors, fmt.Sprintf("invalid capitalization: %s. Allowed: daily, monthly, end, quarterly",
			deposit.Capitalization))
	}

	return nil
}

func (v *DepositValidator) validateFinancialFields(deposit *models.Deposit, errors *[]string) error {
	if deposit.Amount < v.minAmount {
		*errors = append(*errors, fmt.Sprintf("amount below minimum: %d kopecks (%.2f rub)",
			v.minAmount, float64(v.minAmount)/100))
	}

	if deposit.Amount > v.maxAmount {
		*errors = append(*errors, fmt.Sprintf("amount above maximum: %d kopecks (%.2f rub)",
			v.maxAmount, float64(v.maxAmount)/100))
	}

	if deposit.InterestRate <= 0 {
		*errors = append(*errors, "interest rate must be positive")
	}

	if deposit.InterestRate > 100 {
		*errors = append(*errors, "interest rate cannot exceed 100%")
	}

	if deposit.PromoRate != nil {
		if *deposit.PromoRate <= 0 {
			*errors = append(*errors, "promo rate must be positive if set")
		}
		if *deposit.PromoRate > 100 {
			*errors = append(*errors, "promo rate cannot exceed 100%")
		}
		if deposit.PromoRate != nil && *deposit.PromoRate <= deposit.InterestRate {
			*errors = append(*errors, "promo rate must be higher than base rate")
		}
	}

	return nil
}

func (v *DepositValidator) validateDates(deposit *models.Deposit, errors *[]string) error {
	if _, err := time.Parse("2006-01-02", deposit.StartDate); err != nil {
		*errors = append(*errors, fmt.Sprintf("invalid start date format: %s, must be YYYY-MM-DD", deposit.StartDate))
	}

	if deposit.PromoEndDate != "" {
		if _, err := time.Parse("2006-01-02", deposit.PromoEndDate); err != nil {
			*errors = append(*errors, fmt.Sprintf("invalid promo end date format: %s, must be YYYY-MM-DD", deposit.PromoEndDate))
		} else {
			promoEnd, _ := time.Parse("2006-01-02", deposit.PromoEndDate)
			startDate, _ := time.Parse("2006-01-02", deposit.StartDate)
			if promoEnd.Before(startDate) {
				*errors = append(*errors, "promo end date cannot be before start date")
			}
		}
	}

	if deposit.Type == "term" {
		if deposit.EndDate == "" {
			*errors = append(*errors, "term deposit must have end date")
		} else if _, err := time.Parse("2006-01-02", deposit.EndDate); err != nil {
			*errors = append(*errors, fmt.Sprintf("invalid end date format: %s, must be YYYY-MM-DD", deposit.EndDate))
		} else {
			endDate, _ := time.Parse("2006-01-02", deposit.EndDate)
			startDate, _ := time.Parse("2006-01-02", deposit.StartDate)
			if endDate.Before(startDate) || endDate.Equal(startDate) {
				*errors = append(*errors, "end date must be after start date")
			}
		}
	}

	return nil
}

func (v *DepositValidator) validateBusinessRules(deposit *models.Deposit, errors *[]string) error {
	if deposit.Type == "term" {
		if deposit.TermMonths <= 0 {
			*errors = append(*errors, "term deposits must have positive term in months")
		}
		if deposit.TermMonths > 60 {
			*errors = append(*errors, "term cannot exceed 60 months (5 years)")
		}
		if deposit.EndDate == "" {
			*errors = append(*errors, "term deposit must have end date")
		}
	}

	if deposit.PromoRate != nil && deposit.PromoEndDate == "" {
		*errors = append(*errors, "promo rate requires promo end date")
	}

	if deposit.PromoEndDate != "" && deposit.PromoRate == nil {
		*errors = append(*errors, "promo end date requires promo rate")
	}

	if deposit.InitialAmount > deposit.Amount {
		*errors = append(*errors, "current amount cannot be less than initial amount")
	}

	return nil
}

func (v *DepositValidator) getAllowedBanksList() []string {
	banks := make([]string, 0, len(v.allowedBanks))
	for bank := range v.allowedBanks {
		banks = append(banks, bank)
	}
	return banks
}

func (v *DepositValidator) ValidateCreateRequest(name, bank, depositType string, amount int64, interestRate float64, termMonths int, promoRate *float64, promoEndDate string) error {
	testDeposit := &models.Deposit{
		Name:           name,
		Bank:           bank,
		Type:           depositType,
		Amount:         amount,
		InitialAmount:  amount,
		InterestRate:   interestRate,
		PromoRate:      promoRate,
		PromoEndDate:   promoEndDate,
		StartDate:      time.Now().Format("2006-01-02"),
		Capitalization: "daily",
	}

	if depositType == "term" {
		testDeposit.TermMonths = termMonths
		endDate, err := time.Parse("2006-01-02", testDeposit.StartDate)
		if err == nil {
			testDeposit.EndDate = endDate.AddDate(0, termMonths, 0).Format("2006-01-02")
		}
	}

	return v.Validate(testDeposit)
}
