package models

import "time"

type TransactionType string

const (
	TransactionTypeInitialBalance TransactionType = "initial_balance"
	TransactionTypeIncome         TransactionType = "income"
	TransactionTypeExpense        TransactionType = "expense"
	TransactionTypeTransferIn     TransactionType = "transfer_in"
	TransactionTypeTransferOut    TransactionType = "transfer_out"
	TransactionTypeInterestIncome TransactionType = "interest_income"
	TransactionTypeAdjustment     TransactionType = "adjustment"
)

type Transaction struct {
	ID               string          `json:"id"`
	AccountID        string          `json:"account_id"`
	RelatedAccountID *string         `json:"related_account_id,omitempty"`
	TransferID       *string         `json:"transfer_id,omitempty"`
	Type             TransactionType `json:"type"`
	AmountMinor      int64           `json:"amount_minor"`
	CategoryID       *string         `json:"category_id,omitempty"`
	Description      string          `json:"description,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
	CreatedAt        time.Time       `json:"created_at"`
}

type Transfer struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	FromAccountID        string    `json:"from_account_id"`
	ToAccountID          string    `json:"to_account_id"`
	FromTransactionID    string    `json:"from_transaction_id"`
	ToTransactionID      string    `json:"to_transaction_id"`
	FromAmountMinor      int64     `json:"from_amount_minor"`
	ToAmountMinor        int64     `json:"to_amount_minor"`
	FromCurrency         string    `json:"from_currency"`
	ToCurrency           string    `json:"to_currency"`
	ExchangeRate         string    `json:"exchange_rate"`
	ExchangeRateProvider string    `json:"exchange_rate_provider"`
	ExchangeRateDate     time.Time `json:"exchange_rate_date"`
	CreatedAt            time.Time `json:"created_at"`
}
