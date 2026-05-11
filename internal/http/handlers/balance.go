package handlers

import (
	"net/http"

	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) getAccountBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if _, err := h.store.Accounts().GetByIDForUser(r.Context(), accountID, userID); err != nil {
		writeServiceError(w, err)
		return
	}

	transactions, err := h.store.Transactions().ListByAccountForUser(r.Context(), accountID, userID)
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
