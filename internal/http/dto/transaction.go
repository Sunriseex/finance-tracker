package dto

import (
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

type TransactionResponse struct {
	ID               string                 `json:"id"`
	AccountID        string                 `json:"account_id"`
	RelatedAccountID *string                `json:"related_account_id,omitempty"`
	Type             models.TransactionType `json:"type"`
	AmountMinor      int64                  `json:"amount_minor"`
	CategoryID       *string                `json:"category_id,omitempty"`
	Description      string                 `json:"description,omitempty"`
	OccurredAt       time.Time              `json:"occurred_at"`
	CreatedAt        time.Time              `json:"created_at"`
}

type CreateTransactionRequest struct {
	AccountID        string                 `json:"account_id"`
	RelatedAccountID *string                `json:"related_account_id"`
	Type             models.TransactionType `json:"type"`
	AmountMinor      int64                  `json:"amount_minor"`
	CategoryID       *string                `json:"category_id"`
	Description      string                 `json:"description"`
	OccurredAt       string                 `json:"occurred_at"`
}

func TransactionFromModel(transaction *models.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:               transaction.ID,
		AccountID:        transaction.AccountID,
		RelatedAccountID: transaction.RelatedAccountID,
		Type:             transaction.Type,
		AmountMinor:      transaction.AmountMinor,
		CategoryID:       transaction.CategoryID,
		Description:      transaction.Description,
		OccurredAt:       transaction.OccurredAt,
		CreatedAt:        transaction.CreatedAt,
	}
}

func TransactionsFromModels(transactions []models.Transaction) []TransactionResponse {
	response := make([]TransactionResponse, 0, len(transactions))
	for i := range transactions {
		response = append(response, TransactionFromModel(&transactions[i]))
	}
	return response
}
