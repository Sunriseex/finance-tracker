package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	filter, ok := parseTransactionListFilter(w, r)
	if !ok {
		return
	}

	transactionsRepo := h.store.Transactions()
	transactions, err := listTransactionsForUser(r.Context(), transactionsRepo, userID, &filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionsFromModels(transactions))
}

type filteredTransactionLister interface {
	ListByUserFiltered(ctx context.Context, userID string, filter *repository.TransactionListFilter) ([]models.Transaction, error)
}

func listTransactionsForUser(ctx context.Context, transactions repository.TransactionRepository, userID string, filter *repository.TransactionListFilter) ([]models.Transaction, error) {
	if filtered, ok := transactions.(filteredTransactionLister); ok {
		listed, err := filtered.ListByUserFiltered(ctx, userID, filter)
		if err != nil {
			return nil, fmt.Errorf("list filtered transactions: %w", err)
		}
		return listed, nil
	}

	var (
		listed []models.Transaction
		err    error
	)
	if filter.AccountID == "" {
		listed, err = transactions.ListByUser(ctx, userID)
	} else {
		listed, err = transactions.ListByAccountForUser(ctx, filter.AccountID, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return applyTransactionListFilter(listed, filter), nil
}

func parseTransactionListFilter(w http.ResponseWriter, r *http.Request) (repository.TransactionListFilter, bool) {
	query := r.URL.Query()
	filter := repository.TransactionListFilter{
		AccountID:  strings.TrimSpace(query.Get("account_id")),
		CategoryID: strings.TrimSpace(query.Get("category_id")),
		Type:       models.TransactionType(strings.TrimSpace(query.Get("type"))),
		Search:     strings.ToLower(strings.TrimSpace(query.Get("search"))),
		Page:       1,
	}

	if !validateOptionalUUID(w, filter.AccountID, "account_id") ||
		!validateOptionalUUID(w, filter.CategoryID, "category_id") {
		return repository.TransactionListFilter{}, false
	}

	if filter.Type != "" && !validTransactionFilterType(filter.Type) {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid type: "+string(filter.Type), nil)
		return repository.TransactionListFilter{}, false
	}

	var err error
	filter.FromDate, err = parseOptionalDate(query.Get("from_date"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	filter.ToDate, err = parseOptionalDate(query.Get("to_date"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	if !filter.FromDate.IsZero() && !filter.ToDate.IsZero() && filter.ToDate.Before(filter.FromDate) {
		writeError(w, http.StatusBadRequest, "validation_error", "to_date must be on or after from_date", nil)
		return repository.TransactionListFilter{}, false
	}

	filter.Limit, err = parseOptionalPositiveInt(query.Get("limit"), "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	filter.Page, err = parseOptionalPositiveInt(query.Get("page"), "page")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	if filter.Page == 0 {
		filter.Page = 1
	}

	return filter, true
}

func validTransactionFilterType(transactionType models.TransactionType) bool {
	switch transactionType {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeExpense,
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return true
	default:
		return false
	}
}

func parseOptionalPositiveInt(input, field string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(input)
	if err != nil || value <= 0 {
		return 0, errValidation(field + " must be a positive integer")
	}
	return value, nil
}

func applyTransactionListFilter(transactions []models.Transaction, filter *repository.TransactionListFilter) []models.Transaction {
	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		transaction := transactions[i]
		if filter.CategoryID != "" && (transaction.CategoryID == nil || *transaction.CategoryID != filter.CategoryID) {
			continue
		}
		if filter.Type != "" && transaction.Type != filter.Type {
			continue
		}
		occurredAt := dateOnly(transaction.OccurredAt)
		if !filter.FromDate.IsZero() && occurredAt.Before(dateOnly(filter.FromDate)) {
			continue
		}
		if !filter.ToDate.IsZero() && occurredAt.After(dateOnly(filter.ToDate)) {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(transaction.Description), filter.Search) {
			continue
		}
		filtered = append(filtered, transaction)
	}

	if filter.Limit <= 0 {
		return filtered
	}

	start := (filter.Page - 1) * filter.Limit
	if start >= len(filtered) {
		return []models.Transaction{}
	}
	end := min(start+filter.Limit, len(filtered))
	return filtered[start:end]
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

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

	transaction, err := h.transactions.CreateForUser(r.Context(), userID, &services.CreateTransactionRequest{
		AccountID:        accountID,
		RelatedAccountID: relatedAccountID,
		Type:             req.Type,
		AmountMinor:      req.AmountMinor,
		CategoryID:       categoryID,
		Description:      req.Description,
		OccurredAt:       occurredAt,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.TransactionFromModel(transaction))
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	transactionID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	transaction, err := h.store.Transactions().GetByIDForUser(r.Context(), transactionID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionFromModel(transaction))
}

func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	transactionID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	transaction, err := h.store.Transactions().GetByIDForUser(r.Context(), transactionID, userID)
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

	if err := h.store.Transactions().DeleteForUser(r.Context(), transactionID, userID); err != nil {
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
