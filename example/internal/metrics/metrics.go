// Package metrics counts named events, such as "orders.placed".
package metrics

import "example.com/bazelviewer/example/pkg/store"

// Registry counts events by name.
type Registry struct {
	counts *store.Counter
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry { return &Registry{counts: store.New()} }

// Inc counts one event and returns the new total for it.
func (r *Registry) Inc(event string) int { return r.counts.Inc(event) }
