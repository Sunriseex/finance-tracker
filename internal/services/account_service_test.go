package services

import (
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestAccountServiceCreate(t *testing.T) {
	account, err := NewAccountService().Create(t.Context(), CreateAccountRequest{
		Name:     "Savings",
		Bank:     "Yandex",
		Type:     models.AccountTypeSavings,
		Currency: "rub",
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if account.ID == "" {
		t.Fatal("id is empty")
	}
	if account.Currency != "RUB" {
		t.Fatalf("currency = %s, want RUB", account.Currency)
	}
	if !account.IsActive {
		t.Fatal("account must be active")
	}
}

func TestAccountServiceCreateValidatesInput(t *testing.T) {
	_, err := NewAccountService().Create(t.Context(), CreateAccountRequest{
		Name: "Savings",
		Type: "invalid",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
