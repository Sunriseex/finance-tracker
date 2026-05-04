package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/storage"
	"github.com/sunriseex/finance-manager/pkg/errors"
	"github.com/sunriseex/finance-manager/pkg/security"
)

type ExportSnapshot struct {
	ExportedAt time.Time            `json:"exported_at"`
	Source     ExportSnapshotSource `json:"source"`
	Deposits   []models.Deposit     `json:"deposits"`
	Payments   []models.Payment     `json:"payments"`
	Ledger     string               `json:"ledger,omitempty"`
}

type ExportSnapshotSource struct {
	DepositsPath string `json:"deposits_path"`
	PaymentsPath string `json:"payments_path"`
	LedgerPath   string `json:"ledger_path"`
}

func DepositExport(outputPath string) error {
	if outputPath == "" {
		outputPath = "finance-manager-export-" + time.Now().UTC().Format("20060102T150405Z") + ".json"
	}

	deposits, err := storage.LoadDeposits(config.AppConfig.DepositsDataPath)
	if err != nil {
		return errors.NewStorageError("экспорт вкладов", err)
	}

	payments, err := storage.LoadPayments(config.AppConfig.DataPath)
	if err != nil {
		return errors.NewStorageError("экспорт платежей", err)
	}

	ledger, err := readOptionalTextFile(config.AppConfig.LedgerPath)
	if err != nil {
		return errors.NewStorageError("экспорт ledger", err)
	}

	snapshot := ExportSnapshot{
		ExportedAt: time.Now().UTC(),
		Source: ExportSnapshotSource{
			DepositsPath: config.AppConfig.DepositsDataPath,
			PaymentsPath: config.AppConfig.DataPath,
			LedgerPath:   config.AppConfig.LedgerPath,
		},
		Deposits: deposits.Deposits,
		Payments: payments.Payments,
		Ledger:   ledger,
	}

	if err := security.AtomicWriteJSON(snapshot, outputPath); err != nil {
		return errors.NewStorageError("запись export файла", err)
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		absPath = outputPath
	}
	fmt.Printf("✅ Export saved: %s\n", absPath)
	return nil
}

func DepositBackup() error {
	targets := []struct {
		name string
		path string
	}{
		{name: "deposits", path: config.AppConfig.DepositsDataPath},
		{name: "payments", path: config.AppConfig.DataPath},
		{name: "ledger", path: config.AppConfig.LedgerPath},
	}

	created := 0
	for _, target := range targets {
		backupPath, err := security.BackupFile(target.path)
		if err != nil {
			return errors.NewStorageError("backup "+target.name, err)
		}
		if backupPath == "" {
			fmt.Printf("SKIP %s: source not found\n", target.name)
			continue
		}
		created++
		fmt.Printf("OK   %s: %s\n", target.name, backupPath)
	}

	if created == 0 {
		fmt.Println("No files were backed up")
		return nil
	}

	fmt.Printf("✅ Backup completed: %d file(s)\n", created)
	return nil
}

func readOptionalTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
