package services

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsValidationError(t *testing.T) {
	err := validationError("invalid input")

	if !IsValidationError(err) {
		t.Fatal("expected validation error")
	}
}

func TestIsValidationErrorWrapped(t *testing.T) {
	err := fmt.Errorf("create transaction: %w", validationError("invalid input"))

	if !IsValidationError(err) {
		t.Fatal("expected wrapped validation error")
	}
}

func TestIsValidationErrorReturnsFalseForRegularError(t *testing.T) {
	err := errors.New("database failed")

	if IsValidationError(err) {
		t.Fatal("expected regular error")
	}
}
