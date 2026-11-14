// Package inventory tracks how many units of each product are in stock.
package inventory

import (
	"fmt"
	"sync"

	"example.com/bazelviewer/internal/metrics"
	"example.com/bazelviewer/pkg/logging"
)

// Stock counts units per product ID.
type Stock struct {
	mu      sync.Mutex
	units   map[string]int
	metrics *metrics.Registry
	log     *logging.Logger
}

// New returns an empty Stock.
func New(m *metrics.Registry) *Stock {
	return &Stock{units: map[string]int{}, metrics: m, log: logging.New("inventory")}
}

// Restock adds n units of product.
func (s *Stock) Restock(product string, n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.units[product] += n
	s.log.Printf("restocked %s: %d units", product, s.units[product])
}

// Reserve takes n units of product out of stock, if there are enough.
func (s *Stock) Reserve(product string, n int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.units[product] < n {
		s.metrics.Inc("inventory.out_of_stock")
		return fmt.Errorf("only %d units of %s in stock", s.units[product], product)
	}
	s.units[product] -= n
	return nil
}

// Units returns how many units of product are in stock.
func (s *Stock) Units(product string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.units[product]
}
