package storage

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
	appErrors "github.com/sunriseex/finance-manager/pkg/errors"
)

func TestMutatePaymentsCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "payments.json")

	err := MutatePayments(path, func(data *models.PaymentData) error {
		data.Payments = append(data.Payments, models.Payment{
			ID:      "payment-1",
			Name:    "Internet",
			Amount:  50_000,
			DueDate: "2026-05-10",
			Type:    "monthly",
		})
		return nil
	})
	if err != nil {
		t.Fatalf("mutate payments: %v", err)
	}

	data, err := LoadPayments(path)
	if err != nil {
		t.Fatalf("load payments: %v", err)
	}
	if len(data.Payments) != 1 {
		t.Fatalf("payments count = %d, want 1", len(data.Payments))
	}
}

func TestMutatePaymentsReturnsCallbackErrorUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payments.json")
	wantErr := errors.New("нет активных платежей")

	err := MutatePayments(path, func(_ *models.PaymentData) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("mutate payments error = %v, want %v", err, wantErr)
	}
}

func TestUpdateDepositAmountPreservesDomainErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deposits.json")
	data := models.DepositsData{
		Deposits: []models.Deposit{
			{ID: "deposit-1", Name: "Savings", Bank: "Bank", Amount: 100},
		},
	}
	if err := SaveDeposit(data, path); err != nil {
		t.Fatalf("save deposits: %v", err)
	}

	tests := []struct {
		name      string
		depositID string
		amount    int64
		wantCode  appErrors.ErrorCode
	}{
		{
			name:      "insufficient funds",
			depositID: "deposit-1",
			amount:    -101,
			wantCode:  appErrors.ErrBusinessLogic,
		},
		{
			name:      "missing deposit",
			depositID: "missing",
			amount:    1,
			wantCode:  appErrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateDepositAmount(tt.depositID, tt.amount, path)
			if appErrors.GetErrorCode(err) != tt.wantCode {
				t.Fatalf("error code = %s, want %s; err = %v", appErrors.GetErrorCode(err), tt.wantCode, err)
			}
		})
	}
}
