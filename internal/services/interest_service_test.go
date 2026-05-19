package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/config"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/storage"
	appErrors "github.com/sunriseex/capitalflow/pkg/errors"
)

func TestInterestServiceProcessDepositAccrual(t *testing.T) {
	tests := []struct {
		name          string
		stored        []models.Deposit
		deposit       models.Deposit
		wantSuccess   bool
		wantIncome    decimal.Decimal
		wantErrorCode appErrors.ErrorCode
		wantAmount    *int64
	}{
		{
			name: "savings deposit accrues daily interest and updates amount",
			stored: []models.Deposit{
				legacyInterestTestDeposit("deposit-1", "savings", 100_000, 36.5),
			},
			deposit:     legacyInterestTestDeposit("deposit-1", "savings", 100_000, 36.5),
			wantSuccess: true,
			wantIncome:  decimal.NewFromInt(1),
			wantAmount:  int64Ptr(100_100),
		},
		{
			name: "term deposit is skipped",
			stored: []models.Deposit{
				legacyInterestTestDeposit("deposit-2", "term", 100_000, 36.5),
			},
			deposit:     legacyInterestTestDeposit("deposit-2", "term", 100_000, 36.5),
			wantSuccess: true,
			wantIncome:  decimal.Zero,
			wantAmount:  int64Ptr(100_000),
		},
		{
			name: "zero amount savings deposit is skipped",
			stored: []models.Deposit{
				legacyInterestTestDeposit("deposit-3", "savings", 0, 36.5),
			},
			deposit:     legacyInterestTestDeposit("deposit-3", "savings", 0, 36.5),
			wantSuccess: true,
			wantIncome:  decimal.Zero,
			wantAmount:  int64Ptr(0),
		},
		{
			name: "negative amount savings deposit is skipped",
			stored: []models.Deposit{
				legacyInterestTestDeposit("deposit-4", "savings", -100_000, 36.5),
			},
			deposit:     legacyInterestTestDeposit("deposit-4", "savings", -100_000, 36.5),
			wantSuccess: true,
			wantIncome:  decimal.Zero,
			wantAmount:  int64Ptr(-100_000),
		},
		{
			name:          "storage update failure is reported",
			stored:        []models.Deposit{},
			deposit:       legacyInterestTestDeposit("missing", "savings", 100_000, 36.5),
			wantSuccess:   false,
			wantIncome:    decimal.NewFromInt(1),
			wantErrorCode: appErrors.ErrStorage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "deposits.json")
			writeLegacyInterestDeposits(t, path, tt.stored)
			withDepositsDataPath(t, path)

			result := NewInterestService().processDepositAccrual(&tt.deposit)

			if result.Success != tt.wantSuccess {
				t.Fatalf("success = %v, want %v", result.Success, tt.wantSuccess)
			}
			if !result.Income.Equal(tt.wantIncome) {
				t.Fatalf("income = %s, want %s", result.Income, tt.wantIncome)
			}
			if gotCode := appErrors.GetErrorCode(result.Error); gotCode != tt.wantErrorCode {
				t.Fatalf("error code = %s, want %s; err = %v", gotCode, tt.wantErrorCode, result.Error)
			}
			if tt.wantAmount != nil {
				deposit, err := storage.GetDepositByID(tt.deposit.ID, path)
				if err != nil {
					t.Fatalf("get deposit: %v", err)
				}
				if deposit.Amount != *tt.wantAmount {
					t.Fatalf("amount = %d, want %d", deposit.Amount, *tt.wantAmount)
				}
			}
		})
	}
}

func TestInterestServiceCalculateProjectedIncome(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deposits.json")
	writeLegacyInterestDeposits(t, path, []models.Deposit{
		legacyInterestTestDeposit("deposit-1", "savings", 100_000, 36.5),
	})
	withDepositsDataPath(t, path)

	t.Run("calculates projected income", func(t *testing.T) {
		got, err := NewInterestService().CalculateProjectedIncome(&CalculateProjectedIncomeRequest{
			DepositID: "deposit-1",
			Days:      10,
		})
		if err != nil {
			t.Fatalf("calculate projected income: %v", err)
		}
		if !got.Success {
			t.Fatal("success = false, want true")
		}
		if got.DepositName != "Test deposit-1" {
			t.Fatalf("deposit name = %q, want Test deposit-1", got.DepositName)
		}
		if got.Amount != 1000 {
			t.Fatalf("amount = %v, want 1000", got.Amount)
		}
		if got.ProjectedIncome != 10 {
			t.Fatalf("projected income = %v, want 10", got.ProjectedIncome)
		}
		if got.TotalAmount != 1010 {
			t.Fatalf("total amount = %v, want 1010", got.TotalAmount)
		}
	})

	tests := []struct {
		name     string
		deposit  string
		days     int
		wantCode appErrors.ErrorCode
	}{
		{
			name:     "zero days",
			deposit:  "deposit-1",
			days:     0,
			wantCode: appErrors.ErrValidation,
		},
		{
			name:     "negative days",
			deposit:  "deposit-1",
			days:     -1,
			wantCode: appErrors.ErrValidation,
		},
		{
			name:     "missing deposit",
			deposit:  "missing",
			days:     1,
			wantCode: appErrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewInterestService().CalculateProjectedIncome(&CalculateProjectedIncomeRequest{
				DepositID: tt.deposit,
				Days:      tt.days,
			})
			if gotCode := appErrors.GetErrorCode(err); gotCode != tt.wantCode {
				t.Fatalf("error code = %s, want %s; err = %v", gotCode, tt.wantCode, err)
			}
		})
	}
}

func legacyInterestTestDeposit(id, depositType string, amount int64, rate float64) models.Deposit {
	return models.Deposit{
		ID:             id,
		Name:           "Test " + id,
		Bank:           "Test Bank",
		Type:           depositType,
		Amount:         amount,
		InitialAmount:  amount,
		InterestRate:   rate,
		Capitalization: "none",
	}
}

func withDepositsDataPath(t *testing.T, path string) {
	t.Helper()

	previous := config.AppConfig
	if previous == nil {
		config.AppConfig = &config.Config{}
	} else {
		next := *previous
		config.AppConfig = &next
	}
	config.AppConfig.DepositsDataPath = path

	t.Cleanup(func() {
		config.AppConfig = previous
	})
}

func writeLegacyInterestDeposits(t *testing.T, path string, deposits []models.Deposit) {
	t.Helper()

	data, err := json.Marshal(models.DepositsData{Deposits: deposits})
	if err != nil {
		t.Fatalf("marshal deposits: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write deposits: %v", err)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
