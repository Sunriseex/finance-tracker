package commands

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sunriseex/finance-manager/internal/config"
)

func DepositDoctor() error {
	startedAt := time.Now()
	slog.Info("doctor started")
	checks := []struct {
		name string
		err  error
	}{
		{name: "deposits JSON", err: validateJSONFile(config.AppConfig.DepositsDataPath)},
		{name: "payments JSON", err: validateJSONFile(config.AppConfig.DataPath)},
	}

	failed := 0
	for _, check := range checks {
		if check.err != nil {
			failed++
			slog.Warn("doctor check failed", "check", check.name, "error", check.err)
			fmt.Printf("FAIL %s: %v\n", check.name, check.err)
			continue
		}
		slog.Info("doctor check passed", "check", check.name)
		fmt.Printf("OK   %s\n", check.name)
	}

	if failed > 0 {
		slog.Warn("doctor completed with failures", "failed", failed, "duration", time.Since(startedAt))
		return fmt.Errorf("doctor found %d failed checks", failed)
	}

	slog.Info("doctor completed", "duration", time.Since(startedAt))
	fmt.Println("Doctor completed successfully")
	return nil
}

func validateJSONFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var payload any
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&payload); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func validateWritableDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}
