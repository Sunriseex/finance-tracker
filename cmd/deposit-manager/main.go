package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sunriseex/finance-manager/internal/commands"
	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/storage"
	"github.com/sunriseex/finance-manager/pkg/errors"
)

const version = "0.1.0-dev"

func main() {
	if err := config.Init(); err != nil {
		slog.Error("Ошибка инициализации конфигурации", "error", err)
		os.Exit(1)
	}

	if err := initializeDataFiles(); err != nil {
		slog.Warn("не удалось инициализировать файл данных", "error", err)
	}

	if len(os.Args) > 1 {
		executeCommandWithArgs()
		return
	}

	executeDefaultCommand()
}

func executeCommandWithArgs() {
	if len(os.Args) == 1 {
		executeDefaultCommand()
		return
	}

	command := os.Args[1]
	startedAt := time.Now()
	slog.Info("deposit-manager command started", "command", command, "args_count", len(os.Args)-2)

	if err := executeCommand(command, os.Args[2:]); err != nil {
		userMsg := errors.GetUserFriendlyMessage(err)
		slog.Error("deposit-manager command failed",
			"command", command,
			"error", userMsg,
			"details", err,
			"duration", time.Since(startedAt))
		os.Exit(1)
	}

	slog.Info("deposit-manager command completed", "command", command, "duration", time.Since(startedAt))
}

func initializeDataFiles() error {
	if err := storage.InitializeDepositsFile(config.AppConfig.DepositsDataPath); err != nil {
		return fmt.Errorf("инициализация файла вкладов: %w", err)
	}
	if err := storage.InitializePaymentFile(config.AppConfig.DataPath); err != nil {
		return fmt.Errorf("инициализация файла платежей: %w", err)
	}
	return nil
}

func executeDefaultCommand() {
	startedAt := time.Now()
	slog.Info("deposit-manager command started", "command", "list", "default", true)
	if err := commands.DepositList(); err != nil {
		slog.Error("Ошибка выполнения команды list", "error", err)
		os.Exit(1)
	}
	fmt.Println()
	slog.Info("deposit-manager command completed", "command", "list", "default", true, "duration", time.Since(startedAt))
}

func executeCommand(command string, args []string) error {
	switch command {
	case "list":
		return commands.DepositList()
	case "topup":
		return handleTopUpCommand(args)
	case "calculate":
		return handleCalculateCommand(args)
	case "create":
		return handleCreateCommand(args)
	case "update":
		return handleUpdateCommand(args)
	case "accrue-interest":
		return commands.DepositAccrueInterest()
	case "version":
		fmt.Println(version)
		return nil
	case "doctor":
		return commands.DepositDoctor()
	case "find":
		return handleFindCommand(args)
	case "export":
		return handleExportCommand(args)
	case "backup":
		return commands.DepositBackup()
	case "help", "-h", "--help":
		showHelp()
		return nil
	default:
		return fmt.Errorf("неизвестная команда: %s\n\nИспользуйте 'deposit-manager help' для справки", command)
	}
}

func handleTopUpCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("использование: deposit-manager topup <deposit_id> <amount>")
	}

	amount, err := commands.ParseRubles(args[1])
	if err != nil {
		return err
	}
	slog.Debug("Пополнение вклада", "deposit id", args[0], "amount", amount)
	return commands.DepositTopUp(args[0], amount)
}

func handleCalculateCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("использование: deposit-manager calculate <deposit_id> <days>")
	}

	days, err := commands.ParseDays(args[1])
	if err != nil {
		return err
	}
	slog.Debug("Расчет дохода", "deposit id", args[0], "days", days)
	return commands.DepositCalculateIncome(args[0], days)
}

func handleUpdateCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("использование: deposit-manager update <deposit_id>")
	}
	slog.Debug("Обновление вклада", "deposit id", args[0])
	return commands.DepositUpdate(args[0])
}

func handleFindCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("использование: deposit-manager find <name> <bank>")
	}
	slog.Debug("Поиск вклада", "name", args[0], "bank", args[1])
	return commands.DepositFind(args[0], args[1])
}

func handleExportCommand(args []string) error {
	outputPath := ""
	if len(args) > 0 {
		outputPath = args[0]
	}
	return commands.DepositExport(outputPath)
}

func handleCreateCommand(args []string) error {
	if len(args) < 6 {
		showCreateUsage()
		return fmt.Errorf("недостаточно аргументов")
	}

	var createParams struct {
		name, bank, depositType, promoEndDate string
		amount                                int64
		rate                                  float64
		termMonths                            int
		promoRate                             *float64
	}

	flagSet := flag.NewFlagSet("create", flag.ContinueOnError)
	flagSet.StringVar(&createParams.name, "name", "", "Название вклада")
	flagSet.StringVar(&createParams.bank, "bank", "", "Банк")
	flagSet.StringVar(&createParams.depositType, "type", "", "Тип вклада (savings|term)")
	flagSet.StringVar(&createParams.promoEndDate, "promo-end", "", "Дата окончания промо-ставки")

	var amountStr, rateStr, termStr, promoRateStr string
	flagSet.StringVar(&amountStr, "amount", "", "Сумма вклада")
	flagSet.StringVar(&rateStr, "rate", "", "Процентная ставка")
	flagSet.StringVar(&termStr, "term", "", "Срок в месяцах")
	flagSet.StringVar(&promoRateStr, "promo-rate", "", "Промо-ставка")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if err := validateAndParseCreateParams(&createParams, amountStr, rateStr, termStr, promoRateStr); err != nil {
		return err
	}

	slog.Debug("Создание вклада",
		"name", createParams.name,
		"bank", createParams.bank,
		"type", createParams.depositType,
		"amount", createParams.amount)

	return commands.DepositCreate(
		createParams.name,
		createParams.bank,
		createParams.depositType,
		createParams.amount,
		createParams.rate,
		createParams.termMonths,
		createParams.promoRate,
		createParams.promoEndDate,
	)
}

func validateAndParseCreateParams(params *struct {
	name, bank, depositType, promoEndDate string
	amount                                int64
	rate                                  float64
	termMonths                            int
	promoRate                             *float64
}, amountStr, rateStr, termStr, promoRateStr string) error {
	if params.name == "" {
		return fmt.Errorf("необходимо указать название вклада (--name)")
	}
	if params.bank == "" {
		return fmt.Errorf("необходимо указать банк (--bank)")
	}
	if params.depositType == "" {
		return fmt.Errorf("необходимо указать тип вклада (--type savings|term)")
	}

	amount, err := commands.ParseRubles(amountStr)
	if err != nil {
		return err
	}
	params.amount = amount

	rate, err := commands.ParseRate(rateStr)
	if err != nil {
		return err
	}
	params.rate = rate

	if params.depositType == "term" {
		term, err := commands.ParseTerm(termStr)
		if err != nil {
			return err
		}
		params.termMonths = term
	}

	if promoRateStr != "" {
		promoRate, err := commands.ParseRate(promoRateStr)
		if err != nil {
			return err
		}
		params.promoRate = &promoRate
	}

	return nil
}

func showCreateUsage() {
	fmt.Println("Использование: deposit-manager create --name <name> --bank <bank> --type <savings|term> --amount <amount> --rate <interest_rate> [--term <months>] [--promo-rate <rate> --promo-end <date>]")
	fmt.Println()
	fmt.Println("Примеры:")
	fmt.Println("  deposit-manager create --name \"Яндекс Сейв\" --bank \"Яндекс Банк\" --type savings --amount 50000 --rate 17.0")
	fmt.Println("  deposit-manager create --name \"Яндекс Срочный\" --bank \"Яндекс Банк\" --type term --amount 100000 --rate 17.0 --term 3")
	fmt.Println("  deposit-manager create --name \"Яндекс Промо\" --bank \"Яндекс Банк\" --type savings --amount 50000 --rate 12.0 --promo-rate 17.0 --promo-end 2024-12-31")
}

func showHelp() {
	fmt.Println(`Deposit Manager - Управление банковскими вкладами

Команды:
  deposit-manager                    - Показать список вкладов
  deposit-manager list              - Показать список всех вкладов
  deposit-manager topup <id> <amount> - Пополнить вклад
  deposit-manager calculate <id> <days> - Рассчитать доход по вкладу
  deposit-manager create            - Создать новый вклад
  deposit-manager update <id>       - Обновить даты вклада (пролонгация)
  deposit-manager accrue-interest   - Автоматическое начисление процентов
  deposit-manager find <name> <bank> - Найти вклад по имени и банку
  deposit-manager doctor            - Проверить конфиг и файлы данных
  deposit-manager export [path]     - Экспортировать данные в JSON
  deposit-manager backup            - Создать резервные копии файлов данных
  deposit-manager version           - Показать версию
  deposit-manager help              - Показать эту справку

Примеры:
  deposit-manager create --name "Яндекс Сейв" --bank "Яндекс Банк" --type savings --amount 50000 --rate 17.0
  deposit-manager create --name "Яндекс Срочный" --bank "Яндекс Банк" --type term --amount 100000 --rate 17.0 --term 3
  deposit-manager accrue-interest`)
}
