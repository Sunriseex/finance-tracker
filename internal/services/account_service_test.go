package services

import (
	"context"
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestAccountServiceCreate(t *testing.T) {
	account, err := NewAccountService().Create(t.Context(), &CreateAccountRequest{
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
	_, err := NewAccountService().Create(t.Context(), &CreateAccountRequest{
		Name: "Savings",
		Type: "invalid",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAccountServiceCreateValidatesCurrency(t *testing.T) {
	_, err := NewAccountService().Create(t.Context(), &CreateAccountRequest{
		Name:     "Savings",
		Type:     models.AccountTypeSavings,
		Currency: "RUB1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAccountServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *CreateAccountRequest
	}{
		{"nil request", nil},
		{"missing name", &CreateAccountRequest{Type: models.AccountTypeSavings, Currency: "RUB"}},
		{"invalid type", &CreateAccountRequest{Name: "Main", Type: models.AccountType("invalid"), Currency: "RUB"}},
		{"invalid currency", &CreateAccountRequest{Name: "Main", Type: models.AccountTypeSavings, Currency: "12$"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAccountService()

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
