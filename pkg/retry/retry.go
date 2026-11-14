// Package retry runs an operation again when it fails.
package retry

import "time"

// Do calls fn up to attempts times, waiting backoff after the first failure and
// doubling the wait after each one. It returns the last error.
func Do(attempts int, backoff time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return err
}
