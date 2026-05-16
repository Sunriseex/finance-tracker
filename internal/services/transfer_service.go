package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/sunriseex/capitalflow/internal/models"
)

type TransferService struct {
	transactions *TransactionService
	currency     *CurrencyService
}

func NewTransferService(transactions *TransactionService) *TransferService {
	if transactions == nil {
		transactions = NewTransactionService()
	}
	return &TransferService{transactions: transactions, currency: NewCurrencyService(nil)}
}

type CreateTransferRequest struct {
	UserID        string
	FromAccountID string
	ToAccountID   string
	FromCurrency  string
	ToCurrency    string
	AmountMinor   int64
	Description   string
}

type CreateTransferResponse struct {
	Out          *models.Transaction
	In           *models.Transaction
	ExchangeRate string
}

func (s *TransferService) Create(ctx context.Context, req *CreateTransferRequest) (*CreateTransferResponse, error) {
	if req == nil {
		return nil, validationError("transfer request is required")
	}

	fromAccountID := strings.TrimSpace(req.FromAccountID)
	toAccountID := strings.TrimSpace(req.ToAccountID)
	if fromAccountID == "" {
		return nil, validationError("from account id is required")
	}
	if toAccountID == "" {
		return nil, validationError("to account id is required")
	}
	if fromAccountID == toAccountID {
		return nil, validationError("transfer accounts must be different")
	}
	if req.AmountMinor <= 0 {
		return nil, validationError("transfer amount must be positive")
	}

	inAmountMinor := req.AmountMinor
	exchangeRate := "1"
	fromCurrency := strings.TrimSpace(req.FromCurrency)
	toCurrency := strings.TrimSpace(req.ToCurrency)
	if fromCurrency != "" || toCurrency != "" {
		convertedAmountMinor, rate, err := s.currency.ConvertMinor(ctx, req.AmountMinor, fromCurrency, toCurrency)
		if err != nil {
			return nil, fmt.Errorf("convert transfer amount: %w", err)
		}
		inAmountMinor = convertedAmountMinor
		exchangeRate = rate.String()
	}

	inRelatedID := fromAccountID
	outRelatedID := toAccountID
	created, err := s.transactions.CreateTransfer(ctx, strings.TrimSpace(req.UserID), fromAccountID, toAccountID, fromCurrency, toCurrency, &CreateTransactionRequest{
		AccountID:        fromAccountID,
		RelatedAccountID: &outRelatedID,
		Type:             models.TransactionTypeTransferOut,
		AmountMinor:      req.AmountMinor,
		Description:      req.Description,
	}, &CreateTransactionRequest{
		AccountID:        toAccountID,
		RelatedAccountID: &inRelatedID,
		Type:             models.TransactionTypeTransferIn,
		AmountMinor:      inAmountMinor,
		Description:      req.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("create transfer transactions: %w", err)
	}

	return &CreateTransferResponse{Out: &created[0], In: &created[1], ExchangeRate: exchangeRate}, nil
}
