package validate

import "testing"

func TestEmail(t *testing.T) {
	for _, ok := range []string{"a@b.c", "shop@example.com"} {
		if err := Email(ok); err != nil {
			t.Errorf("Email(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"", "  ", "nobody", "@example.com", "me@"} {
		if err := Email(bad); err == nil {
			t.Errorf("Email(%q) = nil, want an error", bad)
		}
	}
}
