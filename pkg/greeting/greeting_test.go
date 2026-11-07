package greeting

import "testing"

func TestHello(t *testing.T) {
	if got, want := Hello("bazel"), "Hello, Bazel!"; got != want {
		t.Fatalf("Hello() = %q, want %q", got, want)
	}
}
