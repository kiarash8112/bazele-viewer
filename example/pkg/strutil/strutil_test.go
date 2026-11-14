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

func TestSlug(t *testing.T) {
	cases := map[string]string{"": "", "Blue Mug": "blue-mug", "  Tea   Pot ": "tea-pot"}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
