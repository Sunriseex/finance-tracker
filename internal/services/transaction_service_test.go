package services

import (
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestTransactionServiceCreate(t *testing.T) {
	tx, err := NewTransactionService().Create(t.Context(), CreateTransactionRequest{
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

func TestTransactionServiceCreateValidatesInput(t *testing.T) {
	_, err := NewTransactionService().Create(t.Context(), CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 0,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
