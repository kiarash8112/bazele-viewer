// Package orders places customer orders: it prices them from the catalog,
// reserves stock and charges the customer.
package orders

import (
	"fmt"

	"example.com/bazelviewer/example/internal/catalog"
	"example.com/bazelviewer/example/internal/db"
	"example.com/bazelviewer/example/internal/inventory"
	"example.com/bazelviewer/example/internal/metrics"
	"example.com/bazelviewer/example/internal/payments"
	"example.com/bazelviewer/example/pkg/logging"
	"example.com/bazelviewer/example/pkg/money"
	"example.com/bazelviewer/example/pkg/validate"
)

// Order is a placed order.
type Order struct {
	ID       string
	Customer string
	Total    money.Amount
	Receipt  string
}

// Service places orders.
type Service struct {
	db       *db.DB
	catalog  *catalog.Catalog
	stock    *inventory.Stock
	payments *payments.Gateway
	metrics  *metrics.Registry
	log      *logging.Logger
}

// New returns a Service.
func New(d *db.DB, c *catalog.Catalog, s *inventory.Stock, p *payments.Gateway, m *metrics.Registry) *Service {
	return &Service{db: d, catalog: c, stock: s, payments: p, metrics: m, log: logging.New("orders")}
}

// Place orders items (units by product ID) for customer.
func (s *Service) Place(customer string, items map[string]int) (Order, error) {
	if err := validate.Email(customer); err != nil {
		return Order{}, err
	}
	var total money.Amount
	for id, n := range items {
		p, ok := s.catalog.Get(id)
		if !ok {
			return Order{}, fmt.Errorf("no product %s", id)
		}
		if err := s.stock.Reserve(id, n); err != nil {
			return Order{}, err
		}
		total += p.Price * money.Amount(n)
	}
	receipt, err := s.payments.Charge(customer, total)
	if err != nil {
		return Order{}, err
	}
	r := s.db.Insert("orders", map[string]string{"customer": customer, "total": total.String(), "receipt": receipt})
	s.metrics.Inc("orders.placed")
	s.log.Printf("order %s for %s: %s", r.ID, customer, total)
	return Order{ID: r.ID, Customer: customer, Total: total, Receipt: receipt}, nil
}
