package models

import "time"

type Deposit struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Bank           string    `json:"bank"`
	Type           string    `json:"type"`
	Amount         int64     `json:"amount"`
	InitialAmount  int64     `json:"initial_amount"`
	InterestRate   float64   `json:"interest_rate"`
	PromoRate      *float64  `json:"promo_rate,omitempty"`
	PromoEndDate   string    `json:"promo_end_date,omitempty"`
	StartDate      string    `json:"start_date"`
	EndDate        string    `json:"end_date,omitempty"`
	TermMonths     int       `json:"term_months,omitempty"`
	Capitalization string    `json:"capitalization"`
	AutoRenewal    bool      `json:"auto_renewal"`
	TopUpEndDate   string    `json:"top_up_end_date,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DepositsData struct {
	Deposits []Deposit `json:"deposits"`
}
