package services

import "errors"

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}

func validationError(message string) error {
	return ValidationError(message)
}

func IsValidationError(err error) bool {
	var validationErr ValidationError
	return errors.As(err, &validationErr)
}
