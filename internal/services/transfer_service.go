package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/sunriseex/finance-manager/internal/models"
)

type TransferService struct {
	transactions *TransactionService
}

func NewTransferService(transactions *TransactionService) *TransferService {
	if transactions == nil {
		transactions = NewTransactionService()
	}
	return &TransferService{transactions: transactions}
}

type CreateTransferRequest struct {
	FromAccountID string
	ToAccountID   string
	AmountMinor   int64
	Description   string
}

type CreateTransferResponse struct {
	Out *models.Transaction
	In  *models.Transaction
}

func (s *TransferService) Create(ctx context.Context, req CreateTransferRequest) (*CreateTransferResponse, error) {
	fromAccountID := strings.TrimSpace(req.FromAccountID)
	toAccountID := strings.TrimSpace(req.ToAccountID)
	if fromAccountID == "" {
		return nil, fmt.Errorf("from account id is required")
	}
	if toAccountID == "" {
		return nil, fmt.Errorf("to account id is required")
	}
	if fromAccountID == toAccountID {
		return nil, fmt.Errorf("transfer accounts must be different")
	}
	if req.AmountMinor <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	inRelatedID := fromAccountID
	outRelatedID := toAccountID
	created, err := s.transactions.CreateMany(ctx, &CreateTransactionRequest{
		AccountID:        fromAccountID,
		RelatedAccountID: &outRelatedID,
		Type:             models.TransactionTypeTransferOut,
		AmountMinor:      req.AmountMinor,
		Description:      req.Description,
	}, &CreateTransactionRequest{
		AccountID:        toAccountID,
		RelatedAccountID: &inRelatedID,
		Type:             models.TransactionTypeTransferIn,
		AmountMinor:      req.AmountMinor,
		Description:      req.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("create transfer transactions: %w", err)
	}

	return &CreateTransferResponse{Out: &created[0], In: &created[1]}, nil
}
