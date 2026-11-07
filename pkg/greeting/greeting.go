// Package greeting builds greeting messages.
package greeting

import (
	"fmt"

	"example.com/bazelviewer/pkg/strutil"
)

// Hello returns a greeting for name.
func Hello(name string) string {
	return fmt.Sprintf("Hello, %s!", strutil.Title(name))
}
