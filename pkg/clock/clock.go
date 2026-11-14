// Package clock lets code read the time through an interface, so tests can
// control it.
package clock

import "time"

// Clock tells the time.
type Clock interface {
	Now() time.Time
}

type system struct{}

func (system) Now() time.Time { return time.Now() }

// System is the real wall clock.
var System Clock = system{}

// Fixed is a clock that always returns the same time.
type Fixed time.Time

// Now returns the fixed time.
func (f Fixed) Now() time.Time { return time.Time(f) }
