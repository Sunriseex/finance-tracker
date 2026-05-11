package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/config"
	"github.com/sunriseex/capitalflow/internal/migration"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/postgres"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
	"github.com/sunriseex/capitalflow/internal/storage"
	"github.com/sunriseex/capitalflow/pkg/money"
)

const version = "0.3.0-dev"

func main() {
	if err := config.Init(); err != nil {
		slog.Error("config init failed", "error", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		showHelp()
		return
	}

	ctx := context.Background()
	switch os.Args[1] {
	case "doctor":
		if err := runDoctor(ctx, os.Args[2:]); err != nil {
			slog.Error("doctor failed", "error", err)
			os.Exit(1)
		}
	case "accounts":
		if err := runAccounts(ctx, os.Args[2:]); err != nil {
			slog.Error("accounts failed", "error", err)
			os.Exit(1)
		}
	case "transactions":
		if err := runTransactions(ctx, os.Args[2:]); err != nil {
			slog.Error("transactions failed", "error", err)
			os.Exit(1)
		}
	case "balance":
		if err := runBalance(ctx, os.Args[2:]); err != nil {
			slog.Error("balance failed", "error", err)
			os.Exit(1)
		}
	case "migrate-json":
		if err := runMigrateJSON(ctx, os.Args[2:]); err != nil {
			slog.Error("migrate-json failed", "error", err)
			os.Exit(1)
		}
	case "version":
		fmt.Println(version)
	case "help", "-h", "--help":
		showHelp()
	default:
		slog.Error("unknown command", "command", os.Args[1])
		showHelp()
		os.Exit(1)
	}
}

func openStore(ctx context.Context, databaseURL string) (*postgres.Store, func(), error) {
	pool, err := postgres.OpenPool(ctx, databaseURL)
	if err != nil {
		return nil, nil, err
	}
	return postgres.NewStore(pool), pool.Close, nil
}

func databaseFlags(name string, args []string) (*flag.FlagSet, *string, *time.Duration, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	databaseURL := flags.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	timeout := flags.Duration("timeout", 30*time.Second, "operation timeout")
	if err := flags.Parse(args); err != nil {
		return nil, nil, nil, err
	}
	return flags, databaseURL, timeout, nil
}

func runDoctor(ctx context.Context, args []string) error {
	_, databaseURL, timeout, err := databaseFlags("doctor", args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	store, closeStore, err := openStore(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer closeStore()

	if err := store.Ping(ctx); err != nil {
		return err
	}
	fmt.Println("postgres: ok")
	return nil
}

func runAccounts(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("accounts subcommand is required: list or create")
	}

	switch args[0] {
	case "list":
		return runAccountsList(ctx, args[1:])
	case "create":
		return runAccountsCreate(ctx, args[1:])
	default:
		return fmt.Errorf("unknown accounts subcommand: %s", args[0])
	}
}

func runAccountsList(ctx context.Context, args []string) error {
	_, databaseURL, timeout, err := databaseFlags("accounts list", args)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	store, closeStore, err := openStore(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer closeStore()

	accounts, err := store.Accounts().List(ctx)
	if err != nil {
		return err
	}
	for i := range accounts {
		account := &accounts[i]
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%t\n", account.ID, account.Name, account.Type, account.Currency, account.Bank, account.IsActive)
	}
	return nil
}

func runAccountsCreate(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("accounts create", flag.ContinueOnError)
	name := flags.String("name", "", "account name")
	bank := flags.String("bank", "", "bank name")
	accountType := flags.String("type", string(models.AccountTypeOther), "account type")
	currency := flags.String("currency", "RUB", "currency code")
	opened := flags.String("opened", "", "opened date YYYY-MM-DD")
	ownerUserID := flags.String("owner-user-id", "", "owner user id")
	databaseURL := flags.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	timeout := flags.Duration("timeout", 30*time.Second, "operation timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}

	openedAt, err := parseOptionalDate(*opened)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	store, closeStore, err := openStore(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer closeStore()

	resolvedOwnerUserID, err := resolveOwnerUserID(ctx, store.Users(), *ownerUserID)
	if err != nil {
		return err
	}

	service := services.NewAccountService(store.Accounts())
	account, err := service.Create(ctx, &services.CreateAccountRequest{
		OwnerUserID: resolvedOwnerUserID,
		Name:        *name,
		Bank:        *bank,
		Type:        models.AccountType(strings.TrimSpace(*accountType)),
		Currency:    *currency,
		OpenedAt:    openedAt,
	})
	if err != nil {
		return err
	}

	fmt.Printf("%s\t%s\t%s\t%s\n", account.ID, account.Name, account.Type, account.Currency)
	return nil
}

func runTransactions(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("transactions subcommand is required: list or create")
	}

	switch args[0] {
	case "list":
		return runTransactionsList(ctx, args[1:])
	case "create":
		return runTransactionsCreate(ctx, args[1:])
	default:
		return fmt.Errorf("unknown transactions subcommand: %s", args[0])
	}
}

func runTransactionsList(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("transactions list", flag.ContinueOnError)
	accountID := flags.String("account", "", "account id")
	databaseURL := flags.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	timeout := flags.Duration("timeout", 30*time.Second, "operation timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	store, closeStore, err := openStore(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer closeStore()

	var transactions []models.Transaction
	if strings.TrimSpace(*accountID) == "" {
		transactions, err = store.Transactions().List(ctx)
	} else {
		transactions, err = store.Transactions().ListByAccount(ctx, strings.TrimSpace(*accountID))
	}
	if err != nil {
		return err
	}

	for i := range transactions {
		tx := &transactions[i]
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", tx.ID, tx.AccountID, tx.Type, money.FormatLegacyKopecks(tx.AmountMinor), tx.Description)
	}
	return nil
}

func runTransactionsCreate(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("transactions create", flag.ContinueOnError)
	accountID := flags.String("account", "", "account id")
	transactionType := flags.String("type", string(models.TransactionTypeIncome), "transaction type")
	amount := flags.String("amount", "", "amount in RUB")
	description := flags.String("description", "", "description")
	occurred := flags.String("occurred", "", "occurred date YYYY-MM-DD")
	databaseURL := flags.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	timeout := flags.Duration("timeout", 30*time.Second, "operation timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}

	amountMinor, err := parseAmountMinor(*amount)
	if err != nil {
		return err
	}
	occurredAt, err := parseOptionalDate(*occurred)
	if err != nil {
		return err
	}

	parsedType := models.TransactionType(strings.TrimSpace(*transactionType))
	if isTransferTransactionType(parsedType) {
		return fmt.Errorf("transfer transactions must be created through transfer command")
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	store, closeStore, err := openStore(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer closeStore()

	service := services.NewTransactionService(store.Transactions())
	transaction, err := service.Create(ctx, &services.CreateTransactionRequest{
		AccountID:   *accountID,
		Type:        parsedType,
		AmountMinor: amountMinor,
		Description: *description,
		OccurredAt:  occurredAt,
	})
	if err != nil {
		return err
	}

	fmt.Printf("%s\t%s\t%s\t%s\n", transaction.ID, transaction.AccountID, transaction.Type, money.FormatLegacyKopecks(transaction.AmountMinor))
	return nil
}

func runBalance(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("balance", flag.ContinueOnError)
	accountID := flags.String("account", "", "account id")
	databaseURL := flags.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	timeout := flags.Duration("timeout", 30*time.Second, "operation timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*accountID) == "" {
		return fmt.Errorf("account id is required")
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	store, closeStore, err := openStore(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer closeStore()

	transactions, err := store.Transactions().ListByAccount(ctx, strings.TrimSpace(*accountID))
	if err != nil {
		return err
	}
	balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
		AccountID:    strings.TrimSpace(*accountID),
		Transactions: transactions,
	})
	if err != nil {
		return err
	}

	fmt.Printf("%s\t%s\t%d transactions\n", balance.AccountID, money.FormatLegacyKopecks(balance.BalanceMinor), balance.Count)
	return nil
}

func runMigrateJSON(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("migrate-json", flag.ContinueOnError)
	depositsPath := flags.String("deposits", config.AppConfig.DepositsDataPath, "legacy deposits JSON path")
	ownerUserID := flags.String("owner-user-id", "", "owner user id for migrated accounts")
	databaseURL := flags.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	timeout := flags.Duration("timeout", 30*time.Second, "migration timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	deposits, err := storage.LoadDeposits(*depositsPath)
	if err != nil {
		return fmt.Errorf("load deposits json: %w", err)
	}

	store, closeStore, err := openStore(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer closeStore()

	resolvedOwnerUserID, err := resolveOwnerUserID(ctx, store.Users(), *ownerUserID)
	if err != nil {
		return err
	}

	migrator := migration.NewJSONMigrator(
		store.Accounts(),
		store.Transactions(),
		store.InterestRules(),
		migration.WithDepositMigrationRepository(store),
		migration.WithOwnerUserID(resolvedOwnerUserID),
	)
	report, err := migrator.MigrateDeposits(ctx, deposits.Deposits)
	if err != nil {
		return err
	}

	printMigrationReport(report)
	if len(report.Errors) > 0 || !report.BalanceMatchesSource {
		return fmt.Errorf("migration completed with errors or balance mismatch")
	}
	return nil
}

func printMigrationReport(report *migration.JSONMigrationReport) {
	fmt.Println("JSON migration report")
	fmt.Printf("  deposits: %d\n", report.TotalDeposits)
	fmt.Printf("  accounts created: %d\n", report.CreatedAccounts)
	if report.OwnerUserID == "" {
		fmt.Println("  owner_user_id: none (setup will claim unowned accounts)")
	} else {
		fmt.Printf("  owner_user_id: %s\n", report.OwnerUserID)
	}
	fmt.Printf("  interest rules created: %d\n", report.CreatedInterestRules)
	fmt.Printf("  transactions created: %d\n", report.CreatedTransactions)
	fmt.Printf("  skipped existing: %d\n", report.SkippedExisting)
	fmt.Printf("  source balance minor: %d\n", report.SourceBalanceMinor)
	fmt.Printf("  migrated balance minor: %d\n", report.MigratedBalanceMinor)
	fmt.Printf("  balance matches: %t\n", report.BalanceMatchesSource)
	if len(report.Errors) > 0 {
		fmt.Println("  errors:")
		for _, err := range report.Errors {
			fmt.Printf("    - %s\n", err)
		}
	}
}

func parseOptionalDate(input string) (time.Time, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return time.Time{}, nil
	}
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", input)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC), nil
}

func parseAmountMinor(input string) (int64, error) {
	amount, err := money.ParseRUB(input)
	if err != nil {
		return 0, err
	}
	return money.DecimalToLegacyKopecks(amount)
}

func resolveOwnerUserID(ctx context.Context, users repository.UserRepository, ownerUserID string) (string, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)

	count, err := users.Count(ctx)
	if err != nil {
		return "", fmt.Errorf("count users: %w", err)
	}

	if count == 0 {
		if ownerUserID != "" {
			return "", fmt.Errorf("owner-user-id was provided, but setup has not created a user yet")
		}
		return "", nil
	}

	if ownerUserID != "" {
		if _, err := users.GetByID(ctx, ownerUserID); err != nil {
			return "", fmt.Errorf("get owner user: %w", err)
		}
		return ownerUserID, nil
	}

	if count == 1 {
		return "", nil
	}

	return "", fmt.Errorf("owner-user-id is required when multiple users exist")
}

func showHelp() {
	fmt.Println(`capitalflow

Commands:
  capitalflow doctor [--database-url url]
  capitalflow accounts list [--database-url url]
  capitalflow accounts create --name name --owner-user-id user-id [--type type] [--bank bank] [--currency RUB] [--opened YYYY-MM-DD]
  capitalflow transactions list [--account id] [--database-url url]
  capitalflow transactions create --account id --type income --amount 1000.00 [--description text] [--occurred YYYY-MM-DD]
  capitalflow balance --account id [--database-url url]
  capitalflow migrate-json [--deposits path] [--owner-user-id user-id] [--database-url url]
  capitalflow version
  capitalflow help`)
}

func isTransferTransactionType(transactionType models.TransactionType) bool {
	return transactionType == models.TransactionTypeTransferIn ||
		transactionType == models.TransactionTypeTransferOut
}
