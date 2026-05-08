package handlers

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/sunriseex/finance-manager/internal/http/dto"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/services"
)

const dashboardRecentTransactionsLimit = 10
const defaultDashboardMonths = 6

func (h *Handler) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.store.Accounts().List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	transactions, err := h.store.Transactions().List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	summary, err := buildDashboardSummary(r.Context(), time.Now(), accounts, transactions, dashboardRecentTransactionsLimit)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) getDashboardNetWorth(w http.ResponseWriter, r *http.Request) {
	accounts, transactions, ok := h.dashboardData(w, r)
	if !ok {
		return
	}

	response, err := buildDashboardNetWorth(r.Context(), time.Now(), accounts, transactions)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getDashboardCashflow(w http.ResponseWriter, r *http.Request) {
	months, ok := dashboardMonthsParam(w, r)
	if !ok {
		return
	}

	accounts, transactions, ok := h.dashboardData(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, buildDashboardCashflow(time.Now(), accounts, transactions, months))
}

func (h *Handler) getDashboardInterestIncome(w http.ResponseWriter, r *http.Request) {
	months, ok := dashboardMonthsParam(w, r)
	if !ok {
		return
	}

	accounts, transactions, ok := h.dashboardData(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, buildDashboardInterestIncome(time.Now(), accounts, transactions, months))
}

func (h *Handler) dashboardData(w http.ResponseWriter, r *http.Request) ([]models.Account, []models.Transaction, bool) {
	accounts, err := h.store.Accounts().List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return nil, nil, false
	}

	transactions, err := h.store.Transactions().List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return nil, nil, false
	}

	return accounts, transactions, true
}

func dashboardMonthsParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("months")
	if raw == "" {
		return defaultDashboardMonths, true
	}

	months, err := strconv.Atoi(raw)
	if err != nil || months <= 0 || months > 60 {
		writeError(w, http.StatusBadRequest, "validation_error", "months must be between 1 and 60", nil)
		return 0, false
	}

	return months, true
}

func buildDashboardSummary(ctx context.Context, now time.Time, accounts []models.Account, transactions []models.Transaction, recentLimit int) (*dto.DashboardSummaryResponse, error) {
	if recentLimit < 0 {
		recentLimit = 0
	}

	accountByID := make(map[string]models.Account, len(accounts))
	for i := range accounts {
		accountByID[accounts[i].ID] = accounts[i]
	}

	summary := &dto.DashboardSummaryResponse{
		GeneratedAt:                now,
		AccountsCount:              len(accounts),
		RecentTransactionsLimit:    recentLimit,
		RecentTransactionsReturned: min(recentLimit, len(transactions)),
	}

	balances := make(map[string]int64)
	income := make(map[string]int64)
	expense := make(map[string]int64)
	interestIncome := make(map[string]int64)

	for i := range accounts {
		account := &accounts[i]
		if account.IsActive {
			summary.ActiveAccountsCount++
		}

		balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
			AccountID:    account.ID,
			Transactions: transactions,
		})
		if err != nil {
			return nil, fmt.Errorf("calculate dashboard account balance: %w", err)
		}

		if account.IsActive {
			balances[account.Currency] += balance.BalanceMinor
		}

		summary.AccountBalances = append(summary.AccountBalances, dto.DashboardAccountBalanceResponse{
			AccountID:        account.ID,
			Name:             account.Name,
			Bank:             account.Bank,
			Type:             account.Type,
			Currency:         account.Currency,
			IsActive:         account.IsActive,
			BalanceMinor:     balance.BalanceMinor,
			TransactionCount: balance.Count,
		})
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonthStart := monthStart.AddDate(0, 1, 0)
	for i := range transactions {
		tx := &transactions[i]
		if tx.OccurredAt.Before(monthStart) || !tx.OccurredAt.Before(nextMonthStart) {
			continue
		}

		account, ok := accountByID[tx.AccountID]
		if !ok {
			continue
		}

		switch tx.Type {
		case models.TransactionTypeIncome:
			income[account.Currency] += tx.AmountMinor
		case models.TransactionTypeExpense:
			expense[account.Currency] += tx.AmountMinor
		case models.TransactionTypeInterestIncome:
			income[account.Currency] += tx.AmountMinor
			interestIncome[account.Currency] += tx.AmountMinor
		}
	}

	recent := slices.Clone(transactions)
	slices.SortFunc(recent, func(a, b models.Transaction) int {
		if byOccurredAt := b.OccurredAt.Compare(a.OccurredAt); byOccurredAt != 0 {
			return byOccurredAt
		}
		if byCreatedAt := b.CreatedAt.Compare(a.CreatedAt); byCreatedAt != 0 {
			return byCreatedAt
		}
		return cmp.Compare(b.ID, a.ID)
	})
	if len(recent) > recentLimit {
		recent = recent[:recentLimit]
	}

	summary.Balances = dashboardAmountsFromMap(balances)
	summary.MonthlyIncome = dashboardAmountsFromMap(income)
	summary.MonthlyExpense = dashboardAmountsFromMap(expense)
	summary.MonthlyInterestIncome = dashboardAmountsFromMap(interestIncome)
	summary.RecentTransactions = dto.TransactionsFromModels(recent)
	summary.RecentTransactionsReturned = len(recent)

	return summary, nil
}

func buildDashboardNetWorth(ctx context.Context, now time.Time, accounts []models.Account, transactions []models.Transaction) (*dto.DashboardNetWorthResponse, error) {
	balances := make(map[string]int64)
	response := &dto.DashboardNetWorthResponse{
		GeneratedAt: now,
	}

	for i := range accounts {
		account := &accounts[i]
		balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
			AccountID:    account.ID,
			Transactions: transactions,
		})
		if err != nil {
			return nil, fmt.Errorf("calculate dashboard net worth account balance: %w", err)
		}

		if account.IsActive {
			balances[account.Currency] += balance.BalanceMinor
		}

		response.AccountBalances = append(response.AccountBalances, dto.DashboardAccountBalanceResponse{
			AccountID:        account.ID,
			Name:             account.Name,
			Bank:             account.Bank,
			Type:             account.Type,
			Currency:         account.Currency,
			IsActive:         account.IsActive,
			BalanceMinor:     balance.BalanceMinor,
			TransactionCount: balance.Count,
		})
	}

	response.Balances = dashboardAmountsFromMap(balances)
	return response, nil
}

func buildDashboardCashflow(now time.Time, accounts []models.Account, transactions []models.Transaction, months int) dto.DashboardCashflowResponse {
	accountByID := dashboardAccountByID(accounts)
	buckets := dashboardMonthBuckets(now, months)
	bucketByPeriod := make(map[string]*dto.DashboardCashflowBucketResponse, len(buckets))
	for i := range buckets {
		bucketByPeriod[buckets[i].Period] = &buckets[i]
	}

	for i := range transactions {
		tx := &transactions[i]
		period := tx.OccurredAt.Format("2006-01")
		bucket, ok := bucketByPeriod[period]
		if !ok {
			continue
		}

		account, ok := accountByID[tx.AccountID]
		if !ok {
			continue
		}

		switch tx.Type {
		case models.TransactionTypeIncome, models.TransactionTypeInterestIncome:
			addDashboardAmount(&bucket.Income, account.Currency, tx.AmountMinor)
			addDashboardAmount(&bucket.NetCashflow, account.Currency, tx.AmountMinor)
			bucket.TransactionCount++
		case models.TransactionTypeExpense:
			addDashboardAmount(&bucket.Expense, account.Currency, tx.AmountMinor)
			addDashboardAmount(&bucket.NetCashflow, account.Currency, -tx.AmountMinor)
			bucket.TransactionCount++
		}
	}

	return dto.DashboardCashflowResponse{
		GeneratedAt: now,
		Months:      months,
		Buckets:     normalizeCashflowBuckets(buckets),
	}
}

func buildDashboardInterestIncome(now time.Time, accounts []models.Account, transactions []models.Transaction, months int) dto.DashboardInterestIncomeResponse {
	accountByID := dashboardAccountByID(accounts)
	buckets := dashboardInterestMonthBuckets(now, months)
	bucketByPeriod := make(map[string]*dto.DashboardInterestIncomeBucketResponse, len(buckets))
	for i := range buckets {
		bucketByPeriod[buckets[i].Period] = &buckets[i]
	}

	total := make(map[string]int64)
	for i := range transactions {
		tx := &transactions[i]
		if tx.Type != models.TransactionTypeInterestIncome {
			continue
		}

		period := tx.OccurredAt.Format("2006-01")
		bucket, ok := bucketByPeriod[period]
		if !ok {
			continue
		}

		account, ok := accountByID[tx.AccountID]
		if !ok {
			continue
		}

		addDashboardAmount(&bucket.InterestIncome, account.Currency, tx.AmountMinor)
		total[account.Currency] += tx.AmountMinor
		bucket.TransactionCount++
	}

	return dto.DashboardInterestIncomeResponse{
		GeneratedAt: now,
		Months:      months,
		Total:       dashboardAmountsFromMap(total),
		Buckets:     normalizeInterestIncomeBuckets(buckets),
	}
}

func dashboardAmountsFromMap(amounts map[string]int64) []dto.DashboardAmountResponse {
	currencies := make([]string, 0, len(amounts))
	for currency := range amounts {
		currencies = append(currencies, currency)
	}
	slices.Sort(currencies)

	response := make([]dto.DashboardAmountResponse, 0, len(currencies))
	for _, currency := range currencies {
		response = append(response, dto.DashboardAmountResponse{
			Currency:    currency,
			AmountMinor: amounts[currency],
		})
	}
	return response
}

func dashboardAccountByID(accounts []models.Account) map[string]models.Account {
	accountByID := make(map[string]models.Account, len(accounts))
	for i := range accounts {
		accountByID[accounts[i].ID] = accounts[i]
	}
	return accountByID
}

func dashboardMonthBuckets(now time.Time, months int) []dto.DashboardCashflowBucketResponse {
	if months <= 0 {
		months = defaultDashboardMonths
	}

	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -(months - 1), 0)
	buckets := make([]dto.DashboardCashflowBucketResponse, 0, months)
	for i := range months {
		buckets = append(buckets, dto.DashboardCashflowBucketResponse{
			Period: start.AddDate(0, i, 0).Format("2006-01"),
		})
	}
	return buckets
}

func dashboardInterestMonthBuckets(now time.Time, months int) []dto.DashboardInterestIncomeBucketResponse {
	if months <= 0 {
		months = defaultDashboardMonths
	}

	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -(months - 1), 0)
	buckets := make([]dto.DashboardInterestIncomeBucketResponse, 0, months)
	for i := range months {
		buckets = append(buckets, dto.DashboardInterestIncomeBucketResponse{
			Period: start.AddDate(0, i, 0).Format("2006-01"),
		})
	}
	return buckets
}

func addDashboardAmount(amounts *[]dto.DashboardAmountResponse, currency string, delta int64) {
	for i := range *amounts {
		if (*amounts)[i].Currency == currency {
			(*amounts)[i].AmountMinor += delta
			return
		}
	}

	*amounts = append(*amounts, dto.DashboardAmountResponse{
		Currency:    currency,
		AmountMinor: delta,
	})
}

func normalizeCashflowBuckets(buckets []dto.DashboardCashflowBucketResponse) []dto.DashboardCashflowBucketResponse {
	for i := range buckets {
		slices.SortFunc(buckets[i].Income, compareDashboardAmount)
		slices.SortFunc(buckets[i].Expense, compareDashboardAmount)
		slices.SortFunc(buckets[i].NetCashflow, compareDashboardAmount)
	}
	return buckets
}

func normalizeInterestIncomeBuckets(buckets []dto.DashboardInterestIncomeBucketResponse) []dto.DashboardInterestIncomeBucketResponse {
	for i := range buckets {
		slices.SortFunc(buckets[i].InterestIncome, compareDashboardAmount)
	}
	return buckets
}

func compareDashboardAmount(a, b dto.DashboardAmountResponse) int {
	return cmp.Compare(a.Currency, b.Currency)
}
