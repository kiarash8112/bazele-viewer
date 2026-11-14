// Package strutil holds small string helpers.
package strutil

import "strings"

// Title upper-cases the first letter of s.
func Title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Slug lower-cases s and joins its words with "-": "Blue Mug" is "blue-mug".
func Slug(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), "-")
}
