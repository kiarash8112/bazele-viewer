// Package notify emails customers.
package notify

import (
	"time"

	"example.com/bazelviewer/example/pkg/greeting"
	"example.com/bazelviewer/example/pkg/logging"
	"example.com/bazelviewer/example/pkg/retry"
	"example.com/bazelviewer/example/pkg/validate"
)

// Mailer sends email. This example logs each message instead.
type Mailer struct {
	from string
	log  *logging.Logger
}

// New returns a Mailer that sends from the given address.
func New(from string) *Mailer { return &Mailer{from: from, log: logging.New("notify")} }

// Send emails to with subject, greeting them by name.
func (m *Mailer) Send(to, subject string) error {
	if err := validate.Email(to); err != nil {
		return err
	}
	return retry.Do(3, 50*time.Millisecond, func() error {
		m.log.Printf("%s -> %s: %s. %s", m.from, to, subject, greeting.Hello(to))
		return nil
	})
}
