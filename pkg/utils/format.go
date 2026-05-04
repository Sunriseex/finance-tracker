package utils

import (
	"fmt"

	"github.com/sunriseex/finance-manager/pkg/money"
)

func FormatRubles(kopecks int64) string {
	return money.FormatLegacyKopecks(kopecks)
}

func RublesToKopecks(rublesStr string) (int64, error) {
	amount, err := money.ParseRUB(rublesStr)
	if err != nil {
		return 0, fmt.Errorf("неверный формат суммы: %w", err)
	}

	kopecks, err := money.DecimalToLegacyKopecks(amount)
	if err != nil {
		return 0, fmt.Errorf("convert amount to kopecks: %w", err)
	}
	return kopecks, nil
}

func TruncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	return str[:length] + "..."
}
