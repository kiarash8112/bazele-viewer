// Package cache keeps values for a limited time.
package cache

import (
	"sync"
	"time"

	"example.com/bazelviewer/pkg/clock"
)

type entry[V any] struct {
	value   V
	expires time.Time
}

// Cache holds values by key until they are older than its TTL.
type Cache[V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	clock   clock.Clock
	entries map[string]entry[V]
}

// New returns an empty Cache.
func New[V any](ttl time.Duration, c clock.Clock) *Cache[V] {
	return &Cache[V]{ttl: ttl, clock: c, entries: map[string]entry[V]{}}
}

// Get returns the value for key, if it is there and fresh.
func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || !c.clock.Now().Before(e.expires) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Put stores value for key.
func (c *Cache[V]) Put(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry[V]{value: value, expires: c.clock.Now().Add(c.ttl)}
}
