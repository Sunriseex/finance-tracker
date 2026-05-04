package services

import (
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestTransferServiceCreate(t *testing.T) {
	got, err := NewTransferService(nil).Create(t.Context(), CreateTransferRequest{
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
	_, err := NewTransferService(nil).Create(t.Context(), CreateTransferRequest{
		FromAccountID: "account-1",
		ToAccountID:   "account-1",
		AmountMinor:   25_000,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
