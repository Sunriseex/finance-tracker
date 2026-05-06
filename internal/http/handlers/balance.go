package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/finance-manager/internal/services"
)

func (h *Handler) getAccountBalance(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")

	if _, err := h.store.Accounts().GetByID(r.Context(), accountID); err != nil {
		writeServiceError(w, err)
		return
	}

	transactions, err := h.store.Transactions().ListByAccount(r.Context(), accountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	balance, err := services.NewBalanceService().Calculate(r.Context(), services.CalculateBalanceRequest{
		AccountID:    accountID,
		Transactions: transactions,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account_id":        balance.AccountID,
		"balance_minor":     balance.BalanceMinor,
		"transaction_count": balance.Count,
	})
}
