package models

import "time"

type AccountType string

const (
	AccountTypeCash        AccountType = "cash"
	AccountTypeCard        AccountType = "card"
	AccountTypeSavings     AccountType = "savings"
	AccountTypeTermDeposit AccountType = "term_deposit"
	AccountTypeBroker      AccountType = "broker"
	AccountTypeOther       AccountType = "other"
)

type Account struct {
	ID        string      `json:"id"`
	LegacyID  *string     `json:"legacy_id,omitempty"`
	Name      string      `json:"name"`
	Bank      string      `json:"bank,omitempty"`
	Type      AccountType `json:"type"`
	Currency  string      `json:"currency"`
	IsActive  bool        `json:"is_active"`
	OpenedAt  time.Time   `json:"opened_at"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}
