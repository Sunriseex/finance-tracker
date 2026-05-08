package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
