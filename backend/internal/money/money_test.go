package money

import "testing"

func TestFormat(t *testing.T) {
	cases := map[Kobo]string{0: "₦0", 870000: "₦8,700", 123456789: "₦1,234,567.89", 30000: "₦300", 100000000: "₦1,000,000", -870000: "-₦8,700", 5: "₦0.05"}
	for k, want := range cases {
		if got := k.Format(); got != want {
			t.Errorf("%d: got %s want %s", k, got, want)
		}
	}
}

func TestMul(t *testing.T) {
	if FromNaira(3500).MulQty(2) != 700000 {
		t.Fatal("mul")
	}
}
