package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type TransactionService struct {
	repo repository.TransactionRepository
}

const maxTransactionAmountMinor int64 = 100_000_000_000_000

func NewTransactionService(repos ...repository.TransactionRepository) *TransactionService {
	var repo repository.TransactionRepository
	if len(repos) > 0 {
		repo = repos[0]
	}
	return &TransactionService{repo: repo}
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

func (s *TransactionService) Create(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
	transaction, err := buildTransaction(ctx, req)
	if err != nil {
		return nil, err
	}

	if s.repo != nil {
		if err := s.repo.Create(ctx, transaction); err != nil {
			return nil, fmt.Errorf("save transaction: %w", err)
		}
	}

	return transaction, nil
}

func (s *TransactionService) CreateForUser(ctx context.Context, userID string, req *CreateTransactionRequest) (*models.Transaction, error) {
	transaction, err := buildTransaction(ctx, req)
	if err != nil {
		return nil, err
	}

	if s.repo != nil {
		if err := s.repo.CreateForUser(ctx, strings.TrimSpace(userID), transaction); err != nil {
			return nil, fmt.Errorf("save transaction: %w", err)
		}
	}

	return transaction, nil
}

func (s *TransactionService) CreateMany(ctx context.Context, reqs ...*CreateTransactionRequest) ([]models.Transaction, error) {
	transactions := make([]models.Transaction, 0, len(reqs))
	for _, req := range reqs {
		transaction, err := buildTransaction(ctx, req)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *transaction)
	}

	if s.repo != nil {
		if err := s.repo.CreateMany(ctx, transactions); err != nil {
			return nil, fmt.Errorf("save transactions: %w", err)
		}
	}

	return transactions, nil
}

func (s *TransactionService) CreateTransfer(ctx context.Context, userID, fromAccountID, toAccountID, fromCurrency, toCurrency string, reqs ...*CreateTransactionRequest) ([]models.Transaction, error) {
	transactions := make([]models.Transaction, 0, len(reqs))
	for _, req := range reqs {
		transaction, err := buildTransaction(ctx, req)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *transaction)
	}

	if s.repo != nil {
		if err := s.repo.CreateTransfer(ctx, userID, fromAccountID, toAccountID, fromCurrency, toCurrency, transactions); err != nil {
			return nil, fmt.Errorf("save transfer transactions: %w", err)
		}
	}

	return transactions, nil
}

func buildTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create transaction: %w", ctx.Err())
	default:
	}
	if req == nil {
		return nil, fmt.Errorf("create transaction request is required")
	}

	if strings.TrimSpace(req.AccountID) == "" {
		return nil, validationError("account id is required")
	}
	if !validTransactionType(req.Type) {
		return nil, validationError(fmt.Sprintf("invalid transaction type: %s", req.Type))
	}
	if req.AmountMinor == 0 {
		return nil, validationError("amount must be non-zero")
	}
	if req.AmountMinor < -maxTransactionAmountMinor || req.AmountMinor > maxTransactionAmountMinor {
		return nil, validationError(fmt.Sprintf("amount must be between %d and %d minor units", -maxTransactionAmountMinor, maxTransactionAmountMinor))
	}
	if req.Type != models.TransactionTypeAdjustment && req.AmountMinor < 0 {
		return nil, validationError(fmt.Sprintf("amount must be positive for %s transactions", req.Type))
	}

	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	transaction := &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        strings.TrimSpace(req.AccountID),
		RelatedAccountID: req.RelatedAccountID,
		Type:             req.Type,
		AmountMinor:      req.AmountMinor,
		CategoryID:       req.CategoryID,
		Description:      strings.TrimSpace(req.Description),
		OccurredAt:       occurredAt,
		CreatedAt:        time.Now(),
	}

	return transaction, nil
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
