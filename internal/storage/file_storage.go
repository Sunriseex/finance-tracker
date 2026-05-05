package storage

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/pkg/errors"
	"github.com/sunriseex/finance-manager/pkg/security"
)

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func LoadPayments(dataPath string) (*models.PaymentData, error) {

	slog.Debug("Загрузка платежей из файла", "path", dataPath)

	expandedPath := ExpandPath(dataPath)
	var data models.PaymentData

	if err := security.SafeReadJSON(expandedPath, &data); err != nil {
		slog.Error("Ошибка чтения файла платежей", "path", expandedPath, "error", err)
		return nil, errors.NewStorageError("чтение файла платежей", err)
	}

	if data.Payments == nil {
		data.Payments = []models.Payment{}
	}
	slog.Debug("Платежи загружены", "count", len(data.Payments))
	return &data, nil
}

func SavePayments(data *models.PaymentData, dataPath string) error {

	slog.Debug("Сохранение платежей", "count", len(data.Payments), "path", dataPath)

	expandedPath := ExpandPath(dataPath)
	return security.WithFileLock(expandedPath, func() error {
		return savePaymentsUnlocked(data, expandedPath)
	})
}

func MutatePayments(dataPath string, fn func(*models.PaymentData) error) error {
	expandedPath := ExpandPath(dataPath)
	return security.WithFileLock(expandedPath, func() error {
		data, err := loadPaymentsOrEmpty(expandedPath)
		if err != nil {
			return err
		}
		if data.Payments == nil {
			data.Payments = []models.Payment{}
		}
		if err := fn(data); err != nil {
			return err
		}
		return savePaymentsUnlocked(data, expandedPath)
	})
}

func loadPaymentsOrEmpty(dataPath string) (*models.PaymentData, error) {
	data, err := LoadPayments(dataPath)
	if err == nil {
		return data, nil
	}

	expandedPath := ExpandPath(dataPath)
	if _, statErr := os.Stat(expandedPath); os.IsNotExist(statErr) {
		return &models.PaymentData{
			Payments: []models.Payment{},
		}, nil
	}

	return nil, err
}

func savePaymentsUnlocked(data *models.PaymentData, expandedPath string) error {
	if _, err := security.BackupFile(expandedPath); err != nil {
		slog.Error("Ошибка создания резервной копии платежей", "path", expandedPath, "error", err)
		return errors.NewStorageError("резервная копия платежей", err)
	}
	if err := security.AtomicWriteJSON(data, expandedPath); err != nil {
		slog.Error("Ошибка сохранения платежей", "path", expandedPath, "error", err)
		return errors.NewStorageError("сохранение платежей", err)
	}

	slog.Debug("Платежи успешно сохранены", "count", len(data.Payments))

	return nil
}

func InitializeDepositsFile(dataPath string) error {
	expandedPath := ExpandPath(dataPath)
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		initialData := &models.DepositsData{
			Deposits: []models.Deposit{},
		}
		if err := security.AtomicWriteJSON(initialData, expandedPath); err != nil {
			return errors.NewStorageError("инициализация файла вкладов", err)
		}
		return nil
	}
	return nil
}

func InitializePaymentFile(dataPath string) error {
	expandedPath := ExpandPath(dataPath)
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		initialData := &models.PaymentData{
			Payments: []models.Payment{},
		}
		if err := security.AtomicWriteJSON(initialData, expandedPath); err != nil {
			return errors.NewStorageError("инициализация файла платежей", err)
		}
		return nil
	}
	return nil
}
