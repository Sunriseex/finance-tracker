package dto

import (
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

type AccountResponse struct {
	ID        string             `json:"id"`
	LegacyID  *string            `json:"legacy_id,omitempty"`
	Name      string             `json:"name"`
	Bank      string             `json:"bank,omitempty"`
	Type      models.AccountType `json:"type"`
	Currency  string             `json:"currency"`
	IsActive  bool               `json:"is_active"`
	OpenedAt  time.Time          `json:"opened_at"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type CreateAccountRequest struct {
	Name     string             `json:"name"`
	Bank     string             `json:"bank"`
	Type     models.AccountType `json:"type"`
	Currency string             `json:"currency"`
	OpenedAt string             `json:"opened_at"`
}

type UpdateAccountRequest struct {
	Name     *string             `json:"name"`
	Bank     *string             `json:"bank"`
	Type     *models.AccountType `json:"type"`
	Currency *string             `json:"currency"`
	OpenedAt *string             `json:"opened_at"`
	IsActive *bool               `json:"is_active"`
}

func AccountFromModel(account *models.Account) AccountResponse {
	return AccountResponse{
		ID:        account.ID,
		LegacyID:  account.LegacyID,
		Name:      account.Name,
		Bank:      account.Bank,
		Type:      account.Type,
		Currency:  account.Currency,
		IsActive:  account.IsActive,
		OpenedAt:  account.OpenedAt,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}
}

func AccountsFromModels(accounts []models.Account) []AccountResponse {
	response := make([]AccountResponse, 0, len(accounts))
	for i := range accounts {
		response = append(response, AccountFromModel(&accounts[i]))
	}
	return response
}
