// Package commands
package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/storage"
	"github.com/sunriseex/finance-manager/pkg/utils"
)

func getNearestPayment() *models.Payment {
	data, err := storage.LoadPayments(config.AppConfig.DataPath)
	if err != nil {
		slog.Error("Ошибка загрузки данных платежей", "error", err)
		return nil
	}

	if data == nil || len(data.Payments) == 0 {
		slog.Debug("Нет данных о платежах или список пуст")
		return nil
	}

	var nearest *models.Payment
	minDays := 999

	for i := range data.Payments {
		currentPayment := data.Payments[i]

		if currentPayment.PaymentDate != "" {
			continue
		}

		if currentPayment.DueDate == "" {
			slog.Debug("Платеж без даты окончания", "payment_id", currentPayment.ID, "name", currentPayment.Name)
			continue
		}

		days := utils.DaysUntil(currentPayment.DueDate)
		if days < minDays {
			minDays = days
			paymentCopy := currentPayment
			nearest = &paymentCopy
		}
	}
	if nearest != nil {
		slog.Debug("Найден ближайший платеж",
			"name", nearest.Name,
			"due_date", nearest.DueDate,
			"days_until", minDays)
	} else {
		slog.Debug("Ближайший платеж не найден")
	}

	return nearest
}

func DisplayWidget() {
	payment := getNearestPayment()
	if payment == nil {
		fmt.Println("💳 Нет платежей")
		return
	}

	if payment.DueDate == "" {
		slog.Warn("Платеж без даты окончания", "payment_id", payment.ID, "name", payment.Name)
		fmt.Println("💳 Ошибка: нет даты платежа")
		return
	}

	days := utils.DaysUntil(payment.DueDate)
	amount := utils.FormatRubles(payment.Amount)

	name := payment.Name
	if len(name) > 15 {
		name = name[:15] + "…"
	}

	var icon string

	switch {
	case days < 0:
		icon = "🔴"
	case days == 0:
		icon = "🟠"
	case days <= 7:
		icon = "🟡"
	default:
		icon = "🟢"
	}
	intervalInfo := ""
	if payment.DaysInterval > 0 {
		intervalInfo = fmt.Sprintf(" [%dд]", payment.DaysInterval)
	}

	fmt.Printf("%s %s %s₽ · %dд%s\n", icon, name, amount, days, intervalInfo)
}
func MarkPaid() error {

	data, err := storage.LoadPayments(config.AppConfig.DataPath)

	if err != nil {
		slog.Error("Ошибка загрузки данных платежей", "error", err)
		return fmt.Errorf("ошибка загрузки данных: %v", err)
	}

	payment := getNearestPayment()
	if payment == nil {
		slog.Warn("Попытка оплатить несуществующий платеж")
		return fmt.Errorf("нет активных платежей")
	}

	today := time.Now().Format("2006-01-02")
	slog.Debug("Начало обработки оплаты платежа",
		"payment_id", payment.ID,
		"name", payment.Name,
		"amount", payment.Amount)

	if err := storage.RecordPaymentToLedger(*payment, config.AppConfig.LedgerPath); err != nil {
		slog.Error("Ошибка записи платежа в ledger",
			"payment_id", payment.ID,
			"error", err)

		return fmt.Errorf("ошибка записи в ledger: %v", err)
	}

	slog.Debug("Отладочная информация перед обновлением",
		"payment", payment.Name,
		"id", payment.ID,
		"type", payment.Type,
		"due_date", payment.DueDate)

	found := false
	for i := range data.Payments {
		if data.Payments[i].ID == payment.ID {
			found = true

			slog.Debug("Найден платеж для обновления", "name", data.Payments[i].Name)

			oldDueDate := data.Payments[i].DueDate

			if payment.Type == "one-time" {
				data.Payments[i].PaymentDate = today
				slog.Info("Разовый платеж оплачен",
					"payment_id", payment.ID,
					"name", payment.Name)
				fmt.Printf("  ✅ Разовый платеж '%s' помечен как оплаченный\n", payment.Name)
			} else {
				newDueDate := extendPaymentDate(data.Payments[i])

				data.Payments[i].DueDate = newDueDate
				data.Payments[i].PaymentDate = ""

				intervalInfo := ""

				if payment.DaysInterval > 0 {
					intervalInfo = fmt.Sprintf(" (интервал %d дней)", payment.DaysInterval)
				}

				oldDueParsed, _ := time.Parse("2006-01-02", oldDueDate)
				newDueParsed, _ := time.Parse("2006-01-02", newDueDate)
				daysAdded := int(newDueParsed.Sub(oldDueParsed).Hours() / 24)

				slog.Info("Повторяющийся платеж обновлен",
					"payment_id", payment.ID,
					"name", payment.Name,
					"old_due_date", oldDueDate,
					"new_due_date", newDueDate,
					"days_added", daysAdded)

				fmt.Printf("  ✅ Повторяющийся платеж '%s' обновлен.\n",
					payment.Name)
				fmt.Printf("Старая дата: %s\n", oldDueDate)
				fmt.Printf("Новая дата: %s\n", newDueDate)
				fmt.Printf("Добавлено дней: %d%s\n", daysAdded, intervalInfo)
			}
			break
		}
	}

	if !found {
		slog.Error("Платеж не найден в данных", "payment_id", payment.ID)
		return fmt.Errorf("платеж с ID %s не найден в данных", payment.ID)
	}

	if err := storage.SavePayments(data, config.AppConfig.DataPath); err != nil {
		slog.Error("Ошибка сохранения платежей", "error", err)
		return fmt.Errorf("ошибка сохранения данных: %v", err)
	}
	slog.Debug("Данные платежей успешно сохранены")
	fmt.Printf("  Данные успешно сохранены\n")

	DisplayWidget()
	return nil
}
func extendPaymentDate(payment models.Payment) string {
	var baseDate time.Time

	if payment.DueDate != "" {
		due, err := time.Parse("2006-01-02", payment.DueDate)
		if err == nil {
			if due.After(time.Now()) {
				baseDate = due
				slog.Debug("Использована существующая дата как базовая",
					"payment", payment.Name,
					"base_date", due.Format("2006-01-02"))
			} else {
				baseDate = time.Now()
				slog.Debug("Использована текущая дата как базовая (платеж просрочен)",
					"payment", payment.Name)
			}
		} else {
			baseDate = time.Now()
			slog.Warn("Ошибка парсинга даты, использована текущая дата",
				"payment", payment.Name,
				"due_date", payment.DueDate)
		}
	} else {
		baseDate = time.Now()
		slog.Debug("Использована текущая дата как базовая (нет даты окончания)",
			"payment", payment.Name)
	}

	if payment.DaysInterval > 0 {
		newDate := baseDate.AddDate(0, 0, payment.DaysInterval).Format("2006-01-02")

		slog.Debug("Дата платежа продлена по интервалу",
			"payment", payment.Name,
			"interval_days", payment.DaysInterval,
			"new_date", newDate)

		return newDate
	}
	switch payment.Type {
	case "yearly":
		newDate := baseDate.AddDate(1, 0, 0).Format("2006-01-02")
		slog.Debug("Дата платежа продлена на год",
			"payment", payment.Name,
			"new_date", newDate)
		return newDate
	case "monthly":
		newDate := baseDate.AddDate(0, 1, 0).Format("2006-01-02")
		slog.Debug("Дата платежа продлена на месяц",
			"payment", payment.Name,
			"new_daye", newDate)
		return newDate
	default:
		newDate := baseDate.AddDate(0, 1, 0).Format("2006-01-02")
		slog.Debug("Дата платежа продлена на месяц (по умолчанию)",
			"payment", payment.Name,
			"new_date", newDate)
		return newDate
	}
}

func ListPayments() error {
	data, err := storage.LoadPayments(config.AppConfig.DataPath)
	if err != nil {
		slog.Error("Ошибка загрузки данных платежей", "error", err)
		return fmt.Errorf("ошибка загрузки данных: %v", err)
	}
	var activePayments []models.Payment

	var totalAmount int64
	for _, p := range data.Payments {
		if p.PaymentDate == "" {
			activePayments = append(activePayments, p)
			totalAmount += p.Amount
		}
	}

	slog.Debug("Загружены активные платежи",
		"total_payments", len(data.Payments),
		"active_payments", len(activePayments),
		"total_amount", totalAmount)

	if len(activePayments) == 0 {
		slog.Info("Нет активных платежей для отображения")
		fmt.Println("Нет активных платежей")
		return nil
	}
	sort.Slice(activePayments, func(i, j int) bool {
		return utils.DaysUntil(activePayments[i].DueDate) < utils.DaysUntil(activePayments[j].DueDate)
	})

	var overdue, urgent, upcoming []models.Payment
	for _, p := range activePayments {
		days := utils.DaysUntil(p.DueDate)
		switch {
		case days < 0:
			overdue = append(overdue, p)
		case days <= 7:
			urgent = append(urgent, p)
		default:
			upcoming = append(upcoming, p)
		}
	}
	slog.Debug("Категоризированы платежи",
		"overdue", len(overdue),
		"urgent", len(urgent),
		"upcoming", len(upcoming))

	fmt.Println("АКТИВНЫЕ ПЛАТЕЖИ:")
	fmt.Println("-----------------")
	fmt.Println("")
	if len(overdue) > 0 {
		fmt.Println("🔴 СРОЧНО (просрочено):")
		for _, p := range overdue {
			days := utils.DaysUntil(p.DueDate)
			amountRubles := utils.FormatRubles(p.Amount)
			fmt.Printf("   • %s: %s₽ (%d дней) [%s]", p.Name, amountRubles, -days, p.Type)
			if p.LedgerAccount != "" {
				fmt.Printf(" [%s]", p.LedgerAccount)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	if len(urgent) > 0 {
		fmt.Println("🟡 БЛИЖАЙШИЕ:")
		for _, p := range urgent {
			days := utils.DaysUntil(p.DueDate)
			amountRubles := utils.FormatRubles(p.Amount)
			fmt.Printf("   • %s: %s₽ (%d дней) [%s]", p.Name, amountRubles, days, p.Type)
			if p.LedgerAccount != "" {
				fmt.Printf(" [%s]", p.PaymentDate)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	if len(upcoming) > 0 {
		fmt.Println("🟢 ОЖИДАЕМЫЕ:")
		for _, p := range upcoming {
			days := utils.DaysUntil(p.DueDate)
			amountRubles := utils.FormatRubles(p.Amount)
			fmt.Printf("   • %s: %s₽ (%d дней) [%s]", p.Name, amountRubles, days, p.Type)
			if p.LedgerAccount != "" {
				fmt.Printf(" [%s]", p.LedgerAccount)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	totalRubles := utils.FormatRubles(totalAmount)
	fmt.Printf("📊 ИТОГО: %d платежей на %s₽\n", len(activePayments), totalRubles)

	return nil
}

func AddPayment() error {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	name := addCmd.String("name", "", "Название платежа")
	amountStr := addCmd.String("amount", "", "Сумма платежа в рублях (например: 349.90)")
	dueDate := addCmd.String("date", "", "Дата окончания (ГГГГ-ММ-ДД)")
	days := addCmd.Int("days", 0, "Количество дней (альтернатива дате)")
	paymentType := addCmd.String("type", "monthly", "Тип: monthly, yearly, one-time")
	category := addCmd.String("category", "", "Категория")
	ledgerAccount := addCmd.String("ledger-account", "", "Счет для ledger")

	addCmd.Parse(os.Args[2:])

	if *name == "" || *amountStr == "" {
		slog.Warn("Попытка добавления платежа без имени или суммы")
		return fmt.Errorf("необходимо указать --name и --amount")
	}
	amount, err := utils.RublesToKopecks(*amountStr)
	if err != nil {
		slog.Error("Ошибка конвертации суммы платежа",
			"amount_string", *amountStr,
			"error", err)

		return fmt.Errorf("ошибка конвертации суммы: %v", err)
	}
	var finalDueDate string
	if *days > 0 {
		finalDueDate = time.Now().AddDate(0, 0, *days).Format("2006-01-02")
	} else if *dueDate != "" {
		_, err = time.Parse("2006-01-02", *dueDate)
		if err != nil {
			slog.Error("Ошибка парсинга даты платежа",
				"date_string", *dueDate,
				"error", err)

			return fmt.Errorf("некорректная дата. Используйте формат YYYY-MM-DD: %v", err)
		}
		finalDueDate = *dueDate
	} else {
		slog.Warn("Попытка добавления платежа без даты или дней")
		return fmt.Errorf("необходимая указать либо --date, либо --days")
	}
	validTypes := map[string]bool{
		"monthly":  true,
		"yearly":   true,
		"one-time": true,
	}
	if !validTypes[*paymentType] {
		slog.Warn("Попытка добавления платежа с некорректным типом",
			"payment_type", paymentType)

		return fmt.Errorf("некорректный тип. Допустимые: monthly, yearly, one-time")
	}
	data, err := storage.LoadPayments(config.AppConfig.DataPath)
	if err != nil {
		slog.Warn("Файл платежей не найден, создается новый", "error", err)
		data = &models.PaymentData{Payments: []models.Payment{}}
	}
	id := uuid.New().String()
	newPayment := models.Payment{
		ID:            id,
		Name:          *name,
		Amount:        amount,
		DueDate:       finalDueDate,
		Type:          *paymentType,
		Category:      *category,
		LedgerAccount: *ledgerAccount,
		DaysInterval:  *days,
	}
	data.Payments = append(data.Payments, newPayment)
	if err := storage.SavePayments(data, config.AppConfig.DataPath); err != nil {
		slog.Error("Ошибка сохранения нового платежа",
			"payment_id", id,
			"error", err)

		return fmt.Errorf("ошибка сохранения платежа: %v", err)
	}

	slog.Info("Новый платеж добавлен",
		"payment_id", id,
		"name", *name,
		"amount", amount,
		"due_date", finalDueDate,
		"type", *paymentType)

	intervalInfo := ""
	if *days > 0 {
		intervalInfo = fmt.Sprintf(" [интервал %d дней]", *days)
	}
	accountInfo := ""
	if *ledgerAccount != "" {
		accountInfo = fmt.Sprintf(" -> %s", *ledgerAccount)
	}
	amountRubles := utils.FormatRubles(amount)
	fmt.Printf("Платеж добавлен: %s - %s₽ - %s [%s]%s%s\n", *name, amountRubles, finalDueDate, *paymentType, intervalInfo, accountInfo)

	return nil
}

func ShowLedger() error {
	ledgerPath := storage.ExpandPath(config.AppConfig.LedgerPath)
	if _, err := os.Stat(ledgerPath); os.IsNotExist(err) {
		slog.Warn("Ошибка чтения файла ledger", "path", ledgerPath)

		return fmt.Errorf("ledger файл не существует")
	}
	content, err := os.ReadFile(ledgerPath)
	if err != nil {
		slog.Error("Ошибка чтения файла ledger", "path", ledgerPath, "error", err)
		return fmt.Errorf("ошибка чтения ledger файл: %v", err)
	}
	slog.Debug("Ledger файл прочитан", "size_bytes", len(content))

	lines := strings.Split(string(content), "\n")
	recentLines := lines[len(lines)-10:]
	fmt.Println("Послдение записи в Ledger:")
	for _, line := range recentLines {
		if strings.TrimSpace(line) != "" {
			fmt.Println(line)
		}
	}
	return nil
}

func CleanupPayments() error {
	data, err := storage.LoadPayments(config.AppConfig.DataPath)
	if err != nil {
		slog.Error("Ошибка загрузки платежей для очистки", "error", err)
		return fmt.Errorf("ошибка загрузки данных: %v", err)
	}
	initialCount := len(data.Payments)
	cleanedData := cleanupOldPayments(*data)

	if err := storage.SavePayments(&cleanedData, config.AppConfig.DataPath); err != nil {
		slog.Error("Ошибка сохранения очищенных платежей", "error", err)
		return fmt.Errorf("ошибка сохранения данных: %v", err)
	}

	removedCount := initialCount - len(cleanedData.Payments)
	slog.Info("Очистка платежей завершена",
		"initial_count", initialCount,
		"final_count", len(cleanedData.Payments),
		"removed_count", removedCount)

	fmt.Printf("Очистка завершена. Удалено %d старых платежей\n", initialCount-len(cleanedData.Payments))
	return nil
}

func cleanupOldPayments(data models.PaymentData) models.PaymentData {
	var validPayments []models.Payment
	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -7)

	slog.Debug("Начало очистки старых платежей", "cutoff_date", cutoffDate.Format("2006-01-02"))

	for _, payment := range data.Payments {
		if payment.PaymentDate != "" {
			validPayments = append(validPayments, payment)
			continue
		}
		due, err := time.Parse("2006-01-02", payment.DueDate)
		if err != nil {
			slog.Warn("Ошибка парсинга даты платежа при очистке",
				"payment_id", payment.ID,
				"due_date", payment.DueDate)
			validPayments = append(validPayments, payment)
			continue
		}
		if due.After(cutoffDate) {
			validPayments = append(validPayments, payment)
		} else {
			slog.Debug("Платеж удален при очистке",
				"payment_id", payment.ID,
				"name", payment.Name,
				"due_date", payment.DueDate)
		}
	}

	data.Payments = validPayments
	return data
}

func ShowHelp() {
	fmt.Println(`Payments CLI - Управление платежами

Команды:
  payments-manager                    - Показать виджет с ближайшим платежом
  payments-manager paid               - Отметить ближайший платеж как оплаченный
  payments-manager list               - Показать все активные платежи
  payments-manager add                - Добавить новый платеж
  payments-manager ledger             - Показать последние записи Ledger
  payments-manager cleanup            - Очистить старые платежи

Команда add (примеры):
  # С указанием даты
  payments-manager add --name "Yandex Plus" --amount 349.90 --date 2024-10-22 --type monthly --category subscriptions
  
  # С указанием дней
  payments-manager add --name "Хостинг" --amount 1500.00 --days 40 --type one-time --category hosting
  
  # С указанием счета Ledger
  payments-manager add --name "Интернет" --amount 500.00 --date 2024-11-01 --type monthly --category utilities --ledger-account Liabilities:AlfaBank`)
}
