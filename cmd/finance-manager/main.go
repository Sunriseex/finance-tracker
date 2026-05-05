package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/migration"
	"github.com/sunriseex/finance-manager/internal/postgres"
	"github.com/sunriseex/finance-manager/internal/storage"
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

func runMigrateJSON(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("migrate-json", flag.ContinueOnError)
	depositsPath := flags.String("deposits", config.AppConfig.DepositsDataPath, "legacy deposits JSON path")
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

	pool, err := postgres.OpenPool(ctx, *databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	migrator := migration.NewJSONMigrator(store.Accounts(), store.Transactions(), store.InterestRules())
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

func showHelp() {
	fmt.Println(`finance-manager

Commands:
  finance-manager migrate-json [--deposits path] [--database-url url]
  finance-manager version
  finance-manager help`)
}
