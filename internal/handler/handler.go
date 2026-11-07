// Package handler wires HTTP requests to greetings.
package handler

import (
	"fmt"
	"net/http"

	"example.com/bazelviewer/pkg/greeting"
	"example.com/bazelviewer/pkg/store"
)

// New returns an http.Handler that greets ?name= and counts visits.
func New(c *store.Counter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "world"
		}
		n := c.Inc(name)
		fmt.Fprintf(w, "%s (visit #%d)\n", greeting.Hello(name), n)
	})
}
