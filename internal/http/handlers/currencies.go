package handlers

import (
	"net/http"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) getCurrencyRates(w http.ResponseWriter, r *http.Request) {
	base := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("base")))
	if base == "" {
		base = "RUB"
	}
	if !validCurrency(base) {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid base currency: "+base, nil)
		return
	}

	rates, err := services.NewCurrencyService(nil).Latest(r.Context(), base)
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.CurrencyRateResponse{
		Base:      rates.Base,
		Date:      rates.Date,
		Provider:  rates.Provider,
		FetchedAt: rates.FetchedAt,
		Rates:     decimalRatesToFloat(rates.Rates),
	})
}

func decimalRatesToFloat(rates map[string]decimal.Decimal) map[string]float64 {
	response := make(map[string]float64, len(rates))
	for currency, rate := range rates {
		value, _ := rate.Float64()
		response[currency] = value
	}
	return response
}
