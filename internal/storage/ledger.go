package storage

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

func RecordPaymentToLedger(payment models.Payment, ledgerPath string) error {

	slog.Debug("Запись платежа в ledger",
		"payment_name", payment.Name,
		"amount", payment.Amount,
		"ledger_path", ledgerPath)

	expandedPath := ExpandPath(ledgerPath)
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("Ошибка создания директории для ledger", "path", dir, "error", err)
		return fmt.Errorf("create ledger directory: %w", err)
	}
	file, err := os.OpenFile(expandedPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("Ошибка открытия файла ledger", "path", expandedPath, "error", err)
		return fmt.Errorf("open ledger file: %w", err)
	}
	defer file.Close()

	today := time.Now().Format("2006/01/02")

	amount := fmt.Sprintf("₽%.2f", float64(payment.Amount)/100.0)

	expenseAccount := "e:Subcriptions"

	if payment.Category != "" {
		expenseAccount = "e:" + payment.Category
	}
	paymentAccount := payment.LedgerAccount
	if paymentAccount == "" {
		paymentAccount = "l:YandexPay"
	}
	entry := fmt.Sprintf("%s %s\n  %s %s\n  %s\n\n",
		today, payment.Name, expenseAccount, amount, paymentAccount,
	)
	_, err = file.WriteString(entry)
	if err != nil {
		slog.Error("Ошибка записи в ledger", "path", expandedPath, "error", err)
		return fmt.Errorf("write payment ledger entry: %w", err)
	}

	slog.Info("Платеж записан в ledger",
		"payment_name", payment.Name,
		"amount", payment.Amount)

	return nil
}

func RecordDepositToLedger(deposit models.Deposit, operationType string, amount int64, description, ledgerPath string) error {

	slog.Debug("Запись операции по вкладу в ledger",
		"deposit_name", deposit.Name,
		"operation_type", operationType,
		"amount", amount,
		"ledger_path", ledgerPath)

	expandedPath := ExpandPath(ledgerPath)
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Error("Ошибка создания директории для ledger", "path", dir, "error", err)
		return fmt.Errorf("create ledger directory: %w", err)
	}

	file, err := os.OpenFile(expandedPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("Ошибка открытия файла ledger", "path", expandedPath, "error", err)
		return fmt.Errorf("open ledger file: %w", err)
	}
	defer file.Close()

	today := time.Now().Format("2006/01/02")
	amountRubles := fmt.Sprintf("₽%.2f", float64(amount)/100.0)

	var creditAccount, debitAccount string
	bankShort := formatBankNameForLedger(deposit.Bank)

	switch operationType {
	case "create":
		creditAccount = fmt.Sprintf("b:%s", bankShort)
		debitAccount = fmt.Sprintf("b:%s:Savings", bankShort)
		if description == "" {
			description = fmt.Sprintf("Открытие вклада %s", deposit.Name)
		}
	case "topup":
		creditAccount = fmt.Sprintf("b:%s", bankShort)
		debitAccount = fmt.Sprintf("b:%s:Savings", bankShort)
		if description == "" {
			description = fmt.Sprintf("Пополнение вклада %s", deposit.Name)
		}
	case "interest":
		creditAccount = "i:Interest:Bank"
		debitAccount = fmt.Sprintf("b:%s:Savings", bankShort)
		if description == "" {
			description = "Выплата процентов"
		}
	default:
		return fmt.Errorf("unknown deposit operation type: %s", operationType)
	}

	entry := fmt.Sprintf("%s %s\n  %s %s\n  %s\n\n",
		today, description,
		debitAccount, amountRubles,
		creditAccount,
	)

	_, err = file.WriteString(entry)
	if err != nil {
		slog.Error("Ошибка записи в ledger", "path", expandedPath, "error", err)
		return fmt.Errorf("write deposit ledger entry: %w", err)
	}

	slog.Info("Операция по вкладу записана в ledger",
		"deposit_name", deposit.Name,
		"operation_type", operationType,
		"amount", amount)

	return nil
}

func formatBankNameForLedger(bank string) string {
	switch bank {
	case "Яндекс Банк", "Yandex":
		return "Yandex"
	case "Альфа Банк", "Alfa":
		return "AlfaBank"
	case "Тинькофф", "Tinkoff":
		return "Tbank"
	default:
		safeName := strings.ReplaceAll(bank, " ", "")
		safeName = strings.ReplaceAll(safeName, "-", "")
		return safeName
	}
}
