package main

import (
	"context"
	"strings"
	"testing"

	"github.com/sunriseex/finance-manager/internal/config"
)

func TestRunTransactionsCreateRejectsTransferTypes(t *testing.T) {
	oldConfig := config.AppConfig
	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
	}
	t.Cleanup(func() {
		config.AppConfig = oldConfig
	})

	tests := []struct {
		name            string
		transactionType string
	}{
		{
			name:            "transfer in",
			transactionType: "transfer_in",
		},
		{
			name:            "transfer out",
			transactionType: "transfer_out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runTransactionsCreate(context.Background(), []string{
				"--account", "account-1",
				"--type", tt.transactionType,
				"--amount", "100.00",
			})

			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), "transfer transactions") {
				t.Fatalf("error = %q, want transfer rejection", err.Error())
			}
		})
	}
}
