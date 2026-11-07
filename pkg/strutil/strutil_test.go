package strutil

import "testing"

func TestTitle(t *testing.T) {
	cases := map[string]string{"": "", "go": "Go", "Bazel": "Bazel"}
	for in, want := range cases {
		if got := Title(in); got != want {
			t.Errorf("Title(%q) = %q, want %q", in, got, want)
		}
	}
}
