// Package validate checks user input.
package validate

import (
	"fmt"
	"strings"
)

// NonEmpty reports an error if value is blank.
func NonEmpty(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// Email reports an error if value does not look like an email address.
func Email(value string) error {
	if err := NonEmpty("email", value); err != nil {
		return err
	}
	at := strings.Index(value, "@")
	if at <= 0 || at == len(value)-1 {
		return fmt.Errorf("invalid email %q", value)
	}
	return nil
}
