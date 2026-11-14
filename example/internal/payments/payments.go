// Package payments charges customers.
package payments

import (
	"errors"
	"time"

	"example.com/bazelviewer/example/internal/metrics"
	"example.com/bazelviewer/example/pkg/ids"
	"example.com/bazelviewer/example/pkg/logging"
	"example.com/bazelviewer/example/pkg/money"
	"example.com/bazelviewer/example/pkg/retry"
)

// ErrInvalidAmount means a charge was not positive.
var ErrInvalidAmount = errors.New("amount must be positive")

// Gateway talks to the payment provider. This example approves every charge.
type Gateway struct {
	metrics *metrics.Registry
	log     *logging.Logger
}

// New returns a Gateway.
func New(m *metrics.Registry) *Gateway { return &Gateway{metrics: m, log: logging.New("payments")} }

// Charge charges amount to customer, retrying transient failures, and returns
// the receipt ID.
func (g *Gateway) Charge(customer string, amount money.Amount) (string, error) {
	if amount <= 0 {
		return "", ErrInvalidAmount
	}
	err := retry.Do(3, 100*time.Millisecond, func() error { return nil })
	if err != nil {
		g.metrics.Inc("payments.failed")
		return "", err
	}
	g.metrics.Inc("payments.charged")
	g.log.Printf("charged %s %s", customer, amount)
	return ids.New("rcpt"), nil
}
