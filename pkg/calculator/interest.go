package calculator

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/pkg/dates"
)

func CalculateIncome(deposit models.Deposit, days int) decimal.Decimal {
	if deposit.Amount <= 0 || days <= 0 {
		return decimal.Zero
	}

	amount := decimal.NewFromInt(deposit.Amount).Div(decimal.NewFromInt(100))

	if deposit.Type == "term" && deposit.StartDate != "" && deposit.EndDate != "" {
		totalDays, err := dates.DaysBetween(deposit.StartDate, deposit.EndDate)
		if err == nil && days == totalDays {
			return calculateTermDepositTotalIncome(deposit, amount, days)
		}
	}

	return calculateStandardIncome(deposit, amount, days)
}

func calculateTermDepositTotalIncome(deposit models.Deposit, amount decimal.Decimal, totalDays int) decimal.Decimal {
	effectiveRate := getEffectiveRateForTerm(deposit, totalDays)

	if deposit.Bank == "Yandex" {
		return calculateEndTerm(amount, effectiveRate, totalDays)
	}

	return calculateByCapitalization(amount, effectiveRate, totalDays, deposit.Capitalization)
}

func getEffectiveRateForTerm(deposit models.Deposit, totalDays int) decimal.Decimal {
	if deposit.PromoRate != nil && isPromoActiveForTerm(deposit, totalDays) {
		return decimal.NewFromFloat(*deposit.PromoRate)
	}

	return decimal.NewFromFloat(deposit.InterestRate)
}

func isPromoActiveForTerm(deposit models.Deposit, totalDays int) bool {
	if deposit.PromoRate == nil || deposit.PromoEndDate == "" || deposit.StartDate == "" {
		return false
	}

	startDate, err := time.Parse("2006-01-02", deposit.StartDate)
	if err != nil {
		return false
	}

	promoEnd, err := time.Parse("2006-01-02", deposit.PromoEndDate)
	if err != nil {
		return false
	}

	endDate := startDate.AddDate(0, 0, totalDays)

	return !promoEnd.Before(endDate)
}

func calculateStandardIncome(deposit models.Deposit, amount decimal.Decimal, days int) decimal.Decimal {
	active, daysUntilPromoEnd := CheckPromoStatus(deposit)

	if active && deposit.PromoRate != nil && daysUntilPromoEnd > 0 {
		if days > daysUntilPromoEnd {
			return calculateIncomeWithPromoTransition(deposit, amount, days, daysUntilPromoEnd)
		}
		return calculateByBankMethod(amount, decimal.NewFromFloat(*deposit.PromoRate), days, deposit)
	}

	return calculateByBankMethod(amount, decimal.NewFromFloat(deposit.InterestRate), days, deposit)
}

func calculateByBankMethod(amount, rate decimal.Decimal, days int, deposit models.Deposit) decimal.Decimal {
	if deposit.Bank == "Yandex" {
		return calculateEndTerm(amount, rate, days)
	}

	return calculateByCapitalization(amount, rate, days, deposit.Capitalization)
}

func CheckPromoStatus(deposit models.Deposit) (active bool, daysRemaining int) {
	if deposit.PromoRate == nil || deposit.PromoEndDate == "" {
		return false, 0
	}

	promoEnd, err := time.Parse("2006-01-02", deposit.PromoEndDate)
	if err != nil {
		return false, 0
	}

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	promoEnd = time.Date(promoEnd.Year(), promoEnd.Month(), promoEnd.Day(), 0, 0, 0, 0, time.UTC)

	daysUntilEnd := int(promoEnd.Sub(today).Hours() / 24)

	return daysUntilEnd >= 0, daysUntilEnd
}

func CalculateTotalTermIncome(deposit models.Deposit) decimal.Decimal {
	if deposit.Type != "term" || deposit.StartDate == "" || deposit.EndDate == "" {
		return decimal.Zero
	}

	start, err := time.Parse("2006-01-02", deposit.StartDate)
	if err != nil {
		return decimal.Zero
	}

	end, err := time.Parse("2006-01-02", deposit.EndDate)
	if err != nil {
		return decimal.Zero
	}

	totalDays := int(end.Sub(start).Hours() / 24)
	if totalDays <= 0 {
		return decimal.Zero
	}

	return CalculateIncome(deposit, totalDays)
}

func calculateIncomeWithPromoTransition(deposit models.Deposit, amount decimal.Decimal, totalDays, promoDaysRemaining int) decimal.Decimal {
	var incomePromo decimal.Decimal

	if deposit.PromoRate != nil {
		promoRate := decimal.NewFromFloat(*deposit.PromoRate)
		incomePromo = calculateByBankMethod(amount, promoRate, promoDaysRemaining, deposit)
	} else {
		incomePromo = decimal.Zero
	}

	amountAfterPromo := amount.Add(incomePromo)
	remainingDays := totalDays - promoDaysRemaining
	baseRate := decimal.NewFromFloat(deposit.InterestRate)

	incomeRemaining := calculateByBankMethod(amountAfterPromo, baseRate, remainingDays, deposit)

	return incomePromo.Add(incomeRemaining)
}

func calculateByCapitalization(amount, rate decimal.Decimal, days int, capitalization string) decimal.Decimal {
	switch capitalization {
	case "daily":
		return calculateDailyCapitalization(amount, rate, days)
	case "monthly":
		return calculateMonthlyCapitalization(amount, rate, days)
	case "end":
		return calculateEndTerm(amount, rate, days)
	default:
		return calculateEndTerm(amount, rate, days)
	}
}

func calculateDailyCapitalization(amount, rate decimal.Decimal, days int) decimal.Decimal {
	daysInYear := decimal.NewFromInt(365)
	dailyRate := rate.Div(decimal.NewFromInt(100)).Div(daysInYear)
	one := decimal.NewFromInt(1)

	factor := one.Add(dailyRate).Pow(decimal.NewFromInt(int64(days)))
	result := factor.Sub(one).Mul(amount)
	return result
}

func calculateMonthlyCapitalization(amount, rate decimal.Decimal, days int) decimal.Decimal {
	daysInMonth := decimal.NewFromFloat(30.44)
	months := decimal.NewFromInt(int64(days)).Div(daysInMonth)
	monthlyRate := rate.Div(decimal.NewFromInt(1200))
	one := decimal.NewFromInt(1)

	monthsInt := int(months.IntPart())
	if monthsInt > 0 {
		factor := one.Add(monthlyRate).Pow(decimal.NewFromInt(int64(monthsInt)))
		result := factor.Sub(one).Mul(amount)
		return result
	}

	return decimal.Zero
}

func calculateEndTerm(amount, rate decimal.Decimal, days int) decimal.Decimal {
	daysDecimal := decimal.NewFromInt(int64(days))
	daysInYear := decimal.NewFromInt(365)

	result := amount.Mul(rate).Div(decimal.NewFromInt(100))
	result = result.Mul(daysDecimal).Div(daysInYear)

	return result
}
