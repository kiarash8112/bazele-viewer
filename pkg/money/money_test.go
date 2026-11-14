package money

import "testing"

func TestString(t *testing.T) {
	cases := map[Amount]string{0: "$0.00", 1250: "$12.50", -5: "-$0.05"}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("Amount(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestSum(t *testing.T) {
	if got := Sum(100, 250, -50); got != 300 {
		t.Errorf("Sum = %d, want 300", got)
	}
}
