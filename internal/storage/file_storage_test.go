package storage

import (
	"path/filepath"
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
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
