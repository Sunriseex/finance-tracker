package commands

import (
	"fmt"
	"os"

	"github.com/sunriseex/finance-manager/internal/config"
)

func Execute() error {
	if err := config.Init(); err != nil {
		return fmt.Errorf("ошибка инициализации конфига: %v", err)
	}
	if len(os.Args) == 1 {
		DisplayWidget()
		return nil
	}
	switch os.Args[1] {
	case "paid":
		return MarkPaid()
	case "list":
		return ListPayments()
	case "add":
		return AddPayment()
	case "cleanup":
		return CleanupPayments()
	case "help", "--help", "-h":
		ShowHelp()
		return nil

	default:
		DisplayWidget()
		return nil
	}
}
