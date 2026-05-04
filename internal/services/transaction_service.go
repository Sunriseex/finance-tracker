package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/finance-manager/internal/models"
)

type TransactionService struct{}

func NewTransactionService() *TransactionService {
	return &TransactionService{}
}

type CreateTransactionRequest struct {
	AccountID        string
	RelatedAccountID *string
	Type             models.TransactionType
	AmountMinor      int64
	CategoryID       *string
	Description      string
	OccurredAt       time.Time
}

func (s *TransactionService) Create(ctx context.Context, req CreateTransactionRequest) (*models.Transaction, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create transaction: %w", ctx.Err())
	default:
	}

	if strings.TrimSpace(req.AccountID) == "" {
		return nil, fmt.Errorf("account id is required")
	}
	if !validTransactionType(req.Type) {
		return nil, fmt.Errorf("invalid transaction type: %s", req.Type)
	}
	if req.AmountMinor == 0 {
		return nil, fmt.Errorf("amount must be non-zero")
	}

	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	return &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        strings.TrimSpace(req.AccountID),
		RelatedAccountID: req.RelatedAccountID,
		Type:             req.Type,
		AmountMinor:      req.AmountMinor,
		CategoryID:       req.CategoryID,
		Description:      strings.TrimSpace(req.Description),
		OccurredAt:       occurredAt,
		CreatedAt:        time.Now(),
	}, nil
}

func validTransactionType(transactionType models.TransactionType) bool {
	switch transactionType {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeExpense,
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return true
	default:
		return false
	}
}
