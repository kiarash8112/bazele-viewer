package retry

import (
	"errors"
	"testing"
)

func TestDoRetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := Do(3, 0, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("Do = %v after %d calls, want nil after 3", err, calls)
	}
}

func TestDoReturnsLastError(t *testing.T) {
	want := errors.New("down")
	if err := Do(2, 0, func() error { return want }); err != want {
		t.Errorf("Do = %v, want %v", err, want)
	}
}
