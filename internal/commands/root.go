package commands

import (
	"fmt"
	"os"
)

func Execute() error {
	if len(os.Args) == 1 {
		return ListPayments()
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
		return ShowUnknownCommand(os.Args[1])
	}
}

func ShowUnknownCommand(command string) error {
	return fmt.Errorf("неизвестная команда: %s", command)
}
