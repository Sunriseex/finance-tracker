package services

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestTransferServiceCreate(t *testing.T) {
	got, err := NewTransferService(nil).Create(t.Context(), &CreateTransferRequest{
		FromAccountID: "account-1",
		ToAccountID:   "account-2",
		AmountMinor:   25_000,
		Description:   "Move savings",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if got.Out.Type != models.TransactionTypeTransferOut {
		t.Fatalf("out type = %s", got.Out.Type)
	}
	if got.In.Type != models.TransactionTypeTransferIn {
		t.Fatalf("in type = %s", got.In.Type)
	}
	if got.Out.AmountMinor != got.In.AmountMinor {
		t.Fatalf("amount mismatch: out=%d in=%d", got.Out.AmountMinor, got.In.AmountMinor)
	}
	if got.Out.RelatedAccountID == nil || *got.Out.RelatedAccountID != "account-2" {
		t.Fatalf("out related account = %v", got.Out.RelatedAccountID)
	}
	if got.In.RelatedAccountID == nil || *got.In.RelatedAccountID != "account-1" {
		t.Fatalf("in related account = %v", got.In.RelatedAccountID)
	}
}

func TestTransferServiceCreateRejectsSameAccount(t *testing.T) {
	_, err := NewTransferService(nil).Create(t.Context(), &CreateTransferRequest{
		FromAccountID: "account-1",
		ToAccountID:   "account-1",
		AmountMinor:   25_000,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTransferServiceCreatePersistsTransactionsAsBatch(t *testing.T) {
	repo := &batchTransactionRepo{}
	got, err := NewTransferService(NewTransactionService(repo)).Create(t.Context(), &CreateTransferRequest{
		FromAccountID: "account-1",
		ToAccountID:   "account-2",
		AmountMinor:   25_000,
		Description:   "Move savings",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if got.Out == nil || got.In == nil {
		t.Fatal("transfer transactions must be returned")
	}
	if repo.createCalls != 0 {
		t.Fatalf("single create calls = %d, want 0", repo.createCalls)
	}
	if len(repo.batches) != 1 {
		t.Fatalf("batch count = %d, want 1", len(repo.batches))
	}
	if len(repo.batches[0]) != 2 {
		t.Fatalf("batch size = %d, want 2", len(repo.batches[0]))
	}
}

func TestTransferServiceCreateConvertsCrossCurrencyAmount(t *testing.T) {
	repo := &batchTransactionRepo{}
	service := NewTransferService(NewTransactionService(repo))
	service.currency = NewCurrencyService(staticExchangeRateProvider{
		rates: &ExchangeRates{
			Base: "RUB",
			Rates: map[string]decimal.Decimal{
				"KRW": decimal.RequireFromString("16.25"),
			},
		},
	})

	got, err := service.Create(t.Context(), &CreateTransferRequest{
		FromAccountID: "rub-account",
		ToAccountID:   "krw-account",
		FromCurrency:  "RUB",
		ToCurrency:    "KRW",
		AmountMinor:   1_000_000,
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if got.Out.AmountMinor != 1_000_000 {
		t.Fatalf("out amount = %d, want 1000000", got.Out.AmountMinor)
	}
	if got.In.AmountMinor != 16_250_000 {
		t.Fatalf("in amount = %d, want 16250000", got.In.AmountMinor)
	}
	if got.ExchangeRate != "16.25" {
		t.Fatalf("exchange rate = %s, want 16.25", got.ExchangeRate)
	}
	if repo.fromCurrency != "RUB" || repo.toCurrency != "KRW" {
		t.Fatalf("repo currencies = %s/%s, want RUB/KRW", repo.fromCurrency, repo.toCurrency)
	}
}

type batchTransactionRepo struct {
	createCalls  int
	batches      [][]models.Transaction
	fromCurrency string
	toCurrency   string
}

func (r *batchTransactionRepo) Create(context.Context, *models.Transaction) error {
	r.createCalls++
	return nil
}

func (r *batchTransactionRepo) CreateForUser(context.Context, string, *models.Transaction) error {
	return errors.New("unexpected user-scoped create")
}

func (r *batchTransactionRepo) CreateMany(_ context.Context, transactions []models.Transaction) error {
	r.batches = append(r.batches, append([]models.Transaction(nil), transactions...))
	return nil
}

func (r *batchTransactionRepo) CreateTransfer(ctx context.Context, _, _, _, fromCurrency, toCurrency string, transactions []models.Transaction) error {
	r.fromCurrency = fromCurrency
	r.toCurrency = toCurrency
	return r.CreateMany(ctx, transactions)
}

func (r *batchTransactionRepo) GetByID(context.Context, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (r *batchTransactionRepo) GetByIDForUser(context.Context, string, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (r *batchTransactionRepo) List(context.Context) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) ListByUser(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) ListByAccount(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) ListByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) GetBalanceByAccountForUser(context.Context, string, string) (balanceMinor, transactionCount int64, err error) {
	return 0, 0, nil
}

func (r *batchTransactionRepo) Delete(context.Context, string) error {
	return nil
}

func (r *batchTransactionRepo) DeleteForUser(context.Context, string, string) error {
	return nil
}

var errNotImplemented = errors.New("not implemented")

func TestTransferServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  CreateTransferRequest
	}{
		{
			name: "missing from account id",
			req: CreateTransferRequest{
				ToAccountID: "account-2",
				AmountMinor: 100,
			},
		},
		{
			name: "missing to account id",
			req: CreateTransferRequest{
				FromAccountID: "account-1",
				AmountMinor:   100,
			},
		},
		{
			name: "same accounts",
			req: CreateTransferRequest{
				FromAccountID: "account-1",
				ToAccountID:   "account-1",
				AmountMinor:   100,
			},
		},
		{
			name: "zero amount",
			req: CreateTransferRequest{
				FromAccountID: "account-1",
				ToAccountID:   "account-2",
				AmountMinor:   0,
			},
		},
		{
			name: "negative amount",
			req: CreateTransferRequest{
				FromAccountID: "account-1",
				ToAccountID:   "account-2",
				AmountMinor:   -100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTransferService(NewTransactionService())

			_, err := service.Create(context.Background(), &tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestTransferServiceCreateDoesNotClassifyRepositoryErrorAsValidation(t *testing.T) {
	repoErr := errors.New("database failed")
	txService := NewTransactionService(failingTransactionRepo{err: repoErr})
	service := NewTransferService(txService)

	_, err := service.Create(context.Background(), &CreateTransferRequest{
		FromAccountID: "account-1",
		ToAccountID:   "account-2",
		AmountMinor:   100,
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if IsValidationError(err) {
		t.Fatalf("expected repository/internal error, got validation error: %v", err)
	}
}
