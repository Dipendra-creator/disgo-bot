package web

import "testing"

func TestAtoiOr(t *testing.T) {
	cases := []struct {
		in  string
		def int
		out int
	}{
		{"", 25, 25},     // empty → default
		{"50", 25, 50},   // valid
		{"0", 25, 0},     // explicit zero is honoured
		{"-3", 0, -3},    // negative passes through (clamped downstream)
		{"abc", 10, 10},  // garbage → default
		{"  5", 7, 7},    // leading space is not trimmed by Atoi → default
		{"100", 25, 100}, // valid large
	}
	for _, c := range cases {
		if got := atoiOr(c.in, c.def); got != c.out {
			t.Errorf("atoiOr(%q, %d) = %d, want %d", c.in, c.def, got, c.out)
		}
	}
}
