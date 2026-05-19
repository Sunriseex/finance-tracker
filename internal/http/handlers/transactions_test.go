package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestIsTransferTransaction(t *testing.T) {
	tests := []struct {
		name            string
		transactionType models.TransactionType
		want            bool
	}{
		{
			name:            "transfer in",
			transactionType: models.TransactionTypeTransferIn,
			want:            true,
		},
		{
			name:            "transfer out",
			transactionType: models.TransactionTypeTransferOut,
			want:            true,
		},
		{
			name:            "income",
			transactionType: models.TransactionTypeIncome,
			want:            false,
		},
		{
			name:            "expense",
			transactionType: models.TransactionTypeExpense,
			want:            false,
		},
		{
			name:            "interest income",
			transactionType: models.TransactionTypeInterestIncome,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransferTransaction(tt.transactionType)
			if got != tt.want {
				t.Fatalf("isTransferTransaction(%q) = %t, want %t", tt.transactionType, got, tt.want)
			}
		})
	}
}

func TestApplyTransactionListFilter(t *testing.T) {
	categoryID := "11111111-1111-1111-1111-111111111111"
	transactions := []models.Transaction{
		{
			ID:          "old-income",
			Type:        models.TransactionTypeIncome,
			AmountMinor: 10_000,
			CategoryID:  &categoryID,
			Description: "Salary May",
			OccurredAt:  time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          "expense",
			Type:        models.TransactionTypeExpense,
			AmountMinor: 3_000,
			Description: "Food",
			OccurredAt:  time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          "new-income",
			Type:        models.TransactionTypeIncome,
			AmountMinor: 20_000,
			CategoryID:  &categoryID,
			Description: "Salary June",
			OccurredAt:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	got := applyTransactionListFilter(transactions, &repository.TransactionListFilter{
		CategoryID: categoryID,
		Type:       models.TransactionTypeIncome,
		FromDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:     time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC),
		Search:     "salary",
		Limit:      1,
		Page:       2,
	})

	if len(got) != 1 {
		t.Fatalf("filtered len = %d, want 1", len(got))
	}
	if got[0].ID != "new-income" {
		t.Fatalf("filtered transaction = %s, want new-income", got[0].ID)
	}
}

func TestParseTransactionListFilterRejectsInvalidQuery(t *testing.T) {
	tests := []string{
		"/api/v1/transactions?account_id=bad",
		"/api/v1/transactions?category_id=bad",
		"/api/v1/transactions?type=bad",
		"/api/v1/transactions?from_date=2026-13-01",
		"/api/v1/transactions?from_date=2026-06-01&to_date=2026-05-01",
		"/api/v1/transactions?limit=0",
		"/api/v1/transactions?page=-1",
	}

	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
			rec := httptest.NewRecorder()

			_, ok := parseTransactionListFilter(rec, req)

			if ok {
				t.Fatal("filter parse succeeded, want rejection")
			}
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestListTransactionsUsesRepositoryFiltering(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	categoryID := "22222222-2222-2222-2222-222222222222"
	transactions := &testTransactionRepo{
		listFilteredTransactions: []models.Transaction{
			{
				ID:          "33333333-3333-3333-3333-333333333333",
				AccountID:   "11111111-1111-1111-1111-111111111111",
				Type:        models.TransactionTypeIncome,
				AmountMinor: 100,
				CategoryID:  &categoryID,
				Description: "Salary June",
				OccurredAt:  time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
				CreatedAt:   time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	store.transactions = transactions
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/transactions?account_id=11111111-1111-1111-1111-111111111111&category_id="+categoryID+"&type=income&from_date=2026-05-01&to_date=2026-06-30&search=salary&limit=10&page=2",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if transactions.listFilteredCalls != 1 {
		t.Fatalf("ListByUserFiltered calls = %d, want 1", transactions.listFilteredCalls)
	}
	if transactions.listFilteredUserID != "user-1" {
		t.Fatalf("filtered user id = %q, want user-1", transactions.listFilteredUserID)
	}
	filter := transactions.listFilteredFilter
	if filter.AccountID != "11111111-1111-1111-1111-111111111111" ||
		filter.CategoryID != categoryID ||
		filter.Type != models.TransactionTypeIncome ||
		filter.Search != "salary" ||
		filter.Limit != 10 ||
		filter.Page != 2 ||
		filter.FromDate.IsZero() ||
		filter.ToDate.IsZero() {
		t.Fatalf("unexpected filter: %+v", filter)
	}
}

func TestCreateTransactionRejectsTransferTypes(t *testing.T) {
	tests := []models.TransactionType{
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
	}

	for _, transactionType := range tests {
		t.Run(string(transactionType), func(t *testing.T) {
			if !isTransferTransaction(transactionType) {
				t.Fatalf("expected %q to be recognized as transfer transaction", transactionType)
			}
		})
	}
}

func TestRejectDirectTransferTransaction(t *testing.T) {
	tests := []struct {
		name            string
		transactionType models.TransactionType
		wantRejected    bool
		wantStatus      int
	}{
		{
			name:            "transfer in",
			transactionType: models.TransactionTypeTransferIn,
			wantRejected:    true,
			wantStatus:      http.StatusBadRequest,
		},
		{
			name:            "transfer out",
			transactionType: models.TransactionTypeTransferOut,
			wantRejected:    true,
			wantStatus:      http.StatusBadRequest,
		},
		{
			name:            "income",
			transactionType: models.TransactionTypeIncome,
			wantRejected:    false,
			wantStatus:      http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			gotRejected := rejectDirectTransferTransaction(rec, tt.transactionType)

			if gotRejected != tt.wantRejected {
				t.Fatalf("rejected = %t, want %t", gotRejected, tt.wantRejected)
			}

			if tt.wantRejected && rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestCreateTransactionUsesUserScopedCreate(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	transactions := &testTransactionRepo{}
	store.transactions = transactions
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transactions", strings.NewReader(`{
		"account_id":"11111111-1111-1111-1111-111111111111",
		"type":"income",
		"amount_minor":100
	}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "create-transaction-user-scoped")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if transactions.oldCreateCalls != 0 {
		t.Fatalf("old Create calls = %d, want 0", transactions.oldCreateCalls)
	}
	if transactions.createForUserCalls != 1 {
		t.Fatalf("CreateForUser calls = %d, want 1", transactions.createForUserCalls)
	}
}

func TestCreateTransactionForOtherUsersAccountReturnsNotFound(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.transactions = &testTransactionRepo{createForUserErr: repository.ErrNotFound}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/transactions", strings.NewReader(`{
		"account_id":"22222222-2222-2222-2222-222222222222",
		"type":"income",
		"amount_minor":100
	}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "create-transaction-other-account")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestRecalculateInterestRejectsInvalidRequestBeforeStoreAccess(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		body      string
	}{
		{
			name:      "invalid account id",
			accountID: "not-a-uuid",
			body:      `{}`,
		},
		{
			name:      "invalid body",
			accountID: "11111111-1111-1111-1111-111111111111",
			body:      `{`,
		},
		{
			name:      "invalid rule id",
			accountID: "11111111-1111-1111-1111-111111111111",
			body:      `{"rule_id":"not-a-uuid"}`,
		},
		{
			name:      "invalid date",
			accountID: "11111111-1111-1111-1111-111111111111",
			body:      `{"from_date":"2026-13-01"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				"/api/v1/accounts/"+tt.accountID+"/recalculate-interest",
				strings.NewReader(tt.body),
			)
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("id", tt.accountID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
			rec := httptest.NewRecorder()

			new(Handler).recalculateInterest(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}
