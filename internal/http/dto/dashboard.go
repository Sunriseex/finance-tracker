package dto

import (
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

type DashboardAmountResponse struct {
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
}

type DashboardAccountBalanceResponse struct {
	AccountID        string             `json:"account_id"`
	Name             string             `json:"name"`
	Bank             string             `json:"bank,omitempty"`
	Type             models.AccountType `json:"type"`
	Currency         string             `json:"currency"`
	IsActive         bool               `json:"is_active"`
	BalanceMinor     int64              `json:"balance_minor"`
	TransactionCount int                `json:"transaction_count"`
}

type DashboardSummaryResponse struct {
	GeneratedAt                time.Time                         `json:"generated_at"`
	AccountsCount              int                               `json:"accounts_count"`
	ActiveAccountsCount        int                               `json:"active_accounts_count"`
	Balances                   []DashboardAmountResponse         `json:"balances"`
	MonthlyIncome              []DashboardAmountResponse         `json:"monthly_income"`
	MonthlyExpense             []DashboardAmountResponse         `json:"monthly_expense"`
	MonthlyInterestIncome      []DashboardAmountResponse         `json:"monthly_interest_income"`
	AccountBalances            []DashboardAccountBalanceResponse `json:"account_balances"`
	RecentTransactions         []TransactionResponse             `json:"recent_transactions"`
	RecentTransactionsLimit    int                               `json:"recent_transactions_limit"`
	RecentTransactionsReturned int                               `json:"recent_transactions_returned"`
}

type DashboardNetWorthResponse struct {
	GeneratedAt     time.Time                         `json:"generated_at"`
	Balances        []DashboardAmountResponse         `json:"balances"`
	AccountBalances []DashboardAccountBalanceResponse `json:"account_balances"`
}

type DashboardCashflowBucketResponse struct {
	Period           string                    `json:"period"`
	Income           []DashboardAmountResponse `json:"income"`
	Expense          []DashboardAmountResponse `json:"expense"`
	NetCashflow      []DashboardAmountResponse `json:"net_cashflow"`
	TransactionCount int                       `json:"transaction_count"`
}

type DashboardCashflowResponse struct {
	GeneratedAt time.Time                         `json:"generated_at"`
	Months      int                               `json:"months"`
	Buckets     []DashboardCashflowBucketResponse `json:"buckets"`
}

type DashboardInterestIncomeBucketResponse struct {
	Period           string                    `json:"period"`
	InterestIncome   []DashboardAmountResponse `json:"interest_income"`
	TransactionCount int                       `json:"transaction_count"`
}

type DashboardInterestIncomeResponse struct {
	GeneratedAt time.Time                               `json:"generated_at"`
	Months      int                                     `json:"months"`
	Total       []DashboardAmountResponse               `json:"total"`
	Buckets     []DashboardInterestIncomeBucketResponse `json:"buckets"`
}
