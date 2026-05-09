package dto

import "time"

type CurrencyRateResponse struct {
	Base      string             `json:"base"`
	Date      string             `json:"date"`
	Provider  string             `json:"provider"`
	FetchedAt time.Time          `json:"fetched_at"`
	Rates     map[string]float64 `json:"rates"`
}
