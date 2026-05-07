package handlers

import (
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
