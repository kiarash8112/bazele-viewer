// Package ids makes random, prefixed identifiers.
package ids

import (
	"crypto/rand"
	"encoding/hex"

	"example.com/bazelviewer/pkg/strutil"
)

// New returns a random ID with the given prefix, such as "ord_3f9a1c2b7e4d5f60".
func New(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand does not fail on supported platforms.
	}
	return strutil.Slug(prefix) + "_" + hex.EncodeToString(b)
}
