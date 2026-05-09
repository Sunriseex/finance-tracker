package handlers

import (
	"net/http"
	"strings"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) createTransfer(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTransferRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	fromAccountID := strings.TrimSpace(req.FromAccountID)
	toAccountID := strings.TrimSpace(req.ToAccountID)

	if !validateOptionalUUID(w, fromAccountID, "from_account_id") {
		return
	}
	if !validateOptionalUUID(w, toAccountID, "to_account_id") {
		return
	}

	fromAccount, ok := h.accountByID(w, r, fromAccountID, "from_account_id")
	if !ok {
		return
	}
	toAccount, ok := h.accountByID(w, r, toAccountID, "to_account_id")
	if !ok {
		return
	}

	result, err := services.NewTransferService(
		services.NewTransactionService(h.store.Transactions()),
	).Create(r.Context(), &services.CreateTransferRequest{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		FromCurrency:  fromAccount.Currency,
		ToCurrency:    toAccount.Currency,
		AmountMinor:   req.AmountMinor,
		Description:   req.Description,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.TransferResponse{
		Out:          dto.TransactionFromModel(result.Out),
		In:           dto.TransactionFromModel(result.In),
		ExchangeRate: result.ExchangeRate,
	})
}
