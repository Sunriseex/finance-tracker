package handlers

import (
	"net/http"
	"strings"

	"github.com/sunriseex/finance-manager/internal/http/dto"
	"github.com/sunriseex/finance-manager/internal/services"
)

func (h *Handler) createTransfer(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTransferRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	fromAccountID := strings.TrimSpace(req.FromAccountID)
	toAccountID := strings.TrimSpace(req.ToAccountID)

	if fromAccountID != "" {
		if !h.ensureAccountExists(w, r, fromAccountID) {
			return
		}
	}
	if toAccountID != "" {
		if !h.ensureAccountExists(w, r, toAccountID) {
			return
		}
	}

	result, err := services.NewTransferService(
		services.NewTransactionService(h.store.Transactions()),
	).Create(r.Context(), services.CreateTransferRequest{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		AmountMinor:   req.AmountMinor,
		Description:   req.Description,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.TransferResponse{
		Out: dto.TransactionFromModel(result.Out),
		In:  dto.TransactionFromModel(result.In),
	})
}
