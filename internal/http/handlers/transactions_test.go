package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/finance-manager/internal/models"
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
				"/api/accounts/"+tt.accountID+"/recalculate-interest",
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
