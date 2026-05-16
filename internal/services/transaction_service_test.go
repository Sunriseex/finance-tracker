package services

import (
	"context"
	"errors"
	"testing"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestTransactionServiceCreate(t *testing.T) {
	tx, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 10_000,
		Description: " Salary ",
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if tx.ID == "" {
		t.Fatal("id is empty")
	}
	if tx.Description != "Salary" {
		t.Fatalf("description = %q, want Salary", tx.Description)
	}
	if tx.OccurredAt.IsZero() {
		t.Fatal("occurred at is zero")
	}
}

func TestTransactionServiceCreateForUser(t *testing.T) {
	repo := &recordingCreateForUserRepo{}
	tx, err := NewTransactionService(repo).CreateForUser(t.Context(), " user-1 ", &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 10_000,
		Description: " Salary ",
	})
	if err != nil {
		t.Fatalf("create transaction for user: %v", err)
	}
	if tx.ID == "" {
		t.Fatal("id is empty")
	}
	if repo.createCalls != 0 {
		t.Fatalf("old Create calls = %d, want 0", repo.createCalls)
	}
	if repo.createForUserCalls != 1 {
		t.Fatalf("CreateForUser calls = %d, want 1", repo.createForUserCalls)
	}
	if repo.userID != "user-1" {
		t.Fatalf("userID = %q, want user-1", repo.userID)
	}
	if repo.transaction == nil || repo.transaction.ID != tx.ID {
		t.Fatal("repo did not receive created transaction")
	}
}

func TestTransactionServiceCreateValidatesInput(t *testing.T) {
	_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 0,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTransactionServiceCreateRejectsNegativeNonAdjustmentAmounts(t *testing.T) {
	tests := []models.TransactionType{
		models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeExpense,
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
		models.TransactionTypeInterestIncome,
	}

	for _, transactionType := range tests {
		t.Run(string(transactionType), func(t *testing.T) {
			_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
				AccountID:   "account-1",
				Type:        transactionType,
				AmountMinor: -1,
			})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestTransactionServiceCreateAllowsNegativeAdjustments(t *testing.T) {
	tx, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeAdjustment,
		AmountMinor: -1_000,
	})
	if err != nil {
		t.Fatalf("create adjustment transaction: %v", err)
	}
	if tx.AmountMinor != -1_000 {
		t.Fatalf("amount = %d, want -1000", tx.AmountMinor)
	}
}

func TestTransactionServiceCreateValidatesAmountBounds(t *testing.T) {
	tests := []struct {
		name        string
		transaction models.TransactionType
		amount      int64
		wantErr     bool
	}{
		{name: "allows positive boundary", transaction: models.TransactionTypeIncome, amount: maxTransactionAmountMinor},
		{name: "rejects positive above boundary", transaction: models.TransactionTypeIncome, amount: maxTransactionAmountMinor + 1, wantErr: true},
		{name: "allows negative adjustment boundary", transaction: models.TransactionTypeAdjustment, amount: -maxTransactionAmountMinor},
		{name: "rejects negative adjustment below boundary", transaction: models.TransactionTypeAdjustment, amount: -maxTransactionAmountMinor - 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
				AccountID:   "account-1",
				Type:        tt.transaction,
				AmountMinor: tt.amount,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !IsValidationError(err) {
					t.Fatalf("expected validation error, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("create transaction: %v", err)
			}
		})
	}
}

func TestTransactionServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *CreateTransactionRequest
	}{
		{
			name: "missing account id",
			req: &CreateTransactionRequest{
				Type:        models.TransactionTypeIncome,
				AmountMinor: 100,
			},
		},
		{
			name: "invalid transaction type",
			req: &CreateTransactionRequest{
				AccountID:   "account-1",
				Type:        models.TransactionType("unknown"),
				AmountMinor: 100,
			},
		},
		{
			name: "zero amount",
			req: &CreateTransactionRequest{
				AccountID:   "account-1",
				Type:        models.TransactionTypeIncome,
				AmountMinor: 0,
			},
		},
		{
			name: "negative income amount",
			req: &CreateTransactionRequest{
				AccountID:   "account-1",
				Type:        models.TransactionTypeIncome,
				AmountMinor: -100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTransactionService()

			_, err := service.Create(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

type failingTransactionRepo struct {
	err error
}

func (r failingTransactionRepo) Create(_ context.Context, _ *models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) CreateForUser(_ context.Context, _ string, _ *models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) CreateMany(_ context.Context, _ []models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) CreateTransfer(_ context.Context, _, _, _, _, _ string, _ []models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) GetByID(_ context.Context, _ string) (*models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) GetByIDForUser(_ context.Context, _, _ string) (*models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) List(_ context.Context) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) ListByUser(_ context.Context, _ string) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) ListByAccount(_ context.Context, _ string) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) ListByAccountForUser(_ context.Context, _, _ string) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) GetBalanceByAccountForUser(context.Context, string, string) (balanceMinor, transactionCount int64, err error) {
	return 0, 0, r.err
}

func (r failingTransactionRepo) Delete(_ context.Context, _ string) error {
	return r.err
}

func (r failingTransactionRepo) DeleteForUser(_ context.Context, _, _ string) error {
	return r.err
}

func TestTransactionServiceCreateDoesNotClassifyRepositoryErrorAsValidation(t *testing.T) {
	repoErr := errors.New("database failed")
	service := NewTransactionService(failingTransactionRepo{err: repoErr})

	_, err := service.Create(context.Background(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 100,
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if IsValidationError(err) {
		t.Fatalf("expected repository/internal error, got validation error: %v", err)
	}
}

func TestTransactionServiceCreateForUserDoesNotClassifyRepositoryErrorAsValidation(t *testing.T) {
	repoErr := errors.New("database failed")
	service := NewTransactionService(failingTransactionRepo{err: repoErr})

	_, err := service.CreateForUser(context.Background(), "user-1", &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 100,
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if IsValidationError(err) {
		t.Fatalf("expected repository/internal error, got validation error: %v", err)
	}
}

type recordingCreateForUserRepo struct {
	failingTransactionRepo
	createCalls        int
	createForUserCalls int
	userID             string
	transaction        *models.Transaction
}

func (r *recordingCreateForUserRepo) Create(context.Context, *models.Transaction) error {
	r.createCalls++
	return nil
}

func (r *recordingCreateForUserRepo) CreateForUser(_ context.Context, userID string, transaction *models.Transaction) error {
	r.createForUserCalls++
	r.userID = userID
	r.transaction = transaction
	return nil
}
