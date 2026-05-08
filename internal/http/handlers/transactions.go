package handlers

import (
	"net/http"
	"strings"

	"github.com/sunriseex/finance-manager/internal/http/dto"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/services"
)

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))

	if !validateOptionalUUID(w, accountID, "account_id") {
		return
	}

	var (
		transactions []models.Transaction
		err          error
	)
	if accountID == "" {
		transactions, err = h.store.Transactions().List(r.Context())
	} else {
		transactions, err = h.store.Transactions().ListByAccount(r.Context(), accountID)
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionsFromModels(transactions))
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	occurredAt, err := parseOptionalDate(req.OccurredAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	if rejectDirectTransferTransaction(w, req.Type) {
		return
	}

	accountID := strings.TrimSpace(req.AccountID)
	if !validateOptionalUUID(w, accountID, "account_id") {
		return
	}

	var relatedAccountID *string
	if req.RelatedAccountID != nil {
		normalized := strings.TrimSpace(*req.RelatedAccountID)
		if !validateOptionalUUID(w, normalized, "related_account_id") {
			return
		}

		if normalized != "" {
			if !h.ensureAccountExists(w, r, normalized) {
				return
			}
			relatedAccountID = &normalized
		}
	}

	var categoryID *string
	if req.CategoryID != nil {
		normalized := strings.TrimSpace(*req.CategoryID)
		if !validateOptionalUUID(w, normalized, "category_id") {
			return
		}

		if normalized != "" {
			if _, err := h.store.Categories().GetByID(r.Context(), normalized); err != nil {
				writeServiceError(w, err)
				return
			}
			categoryID = &normalized
		}
	}

	if accountID != "" {
		if !h.ensureAccountExists(w, r, accountID) {
			return
		}
	}

	transaction, err := services.NewTransactionService(h.store.Transactions()).Create(r.Context(), &services.CreateTransactionRequest{
		AccountID:        accountID,
		RelatedAccountID: relatedAccountID,
		Type:             req.Type,
		AmountMinor:      req.AmountMinor,
		CategoryID:       categoryID,
		Description:      req.Description,
		OccurredAt:       occurredAt,
	})
	if err != nil {
		writeValidationOrServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.TransactionFromModel(transaction))
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	transaction, err := h.store.Transactions().GetByID(r.Context(), transactionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionFromModel(transaction))
}

func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	transaction, err := h.store.Transactions().GetByID(r.Context(), transactionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	if isTransferTransaction(transaction.Type) {
		writeError(
			w,
			http.StatusConflict,
			"transfer_transaction_delete_forbidden",
			"Transfer transactions cannot be deleted through the transaction endpoint",
			nil,
		)
		return
	}

	if err := h.store.Transactions().Delete(r.Context(), transactionID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func isTransferTransaction(transactionType models.TransactionType) bool {
	return transactionType == models.TransactionTypeTransferIn ||
		transactionType == models.TransactionTypeTransferOut
}

func rejectDirectTransferTransaction(w http.ResponseWriter, transactionType models.TransactionType) bool {
	if !isTransferTransaction(transactionType) {
		return false
	}

	writeError(
		w,
		http.StatusBadRequest,
		"validation_error",
		"Transfer transactions must be created through the transfer endpoint",
		nil,
	)
	return true
}
