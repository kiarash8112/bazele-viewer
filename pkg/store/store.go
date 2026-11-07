// Package store is a tiny in-memory visit counter.
package store

import "sync"

// Counter counts visits per name.
type Counter struct {
	mu     sync.Mutex
	counts map[string]int
}

// New returns an empty Counter.
func New() *Counter { return &Counter{counts: map[string]int{}} }

// Inc increments and returns the count for name.
func (c *Counter) Inc(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[name]++
	return c.counts[name]
}
