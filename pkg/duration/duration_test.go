package duration

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"30s", 30 * time.Second},
		{"10m", 10 * time.Minute},
		{"2h", 2 * time.Hour},
		{"2h30m", 2*time.Hour + 30*time.Minute},
		{"7d", 7 * Day},
		{"1w", Week},
		{"1w2d3h", Week + 2*Day + 3*time.Hour},
		{" 1H 30M ", time.Hour + 30*time.Minute}, // case-insensitive + spaces
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Parse(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "10", "m", "10x", "abc", "10m5"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", in)
		}
	}
}

func TestHuman(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{500 * time.Millisecond, "0s"},
		{90 * time.Minute, "1h30m"},
		{6 * Day, "6d"},
		{7 * Day, "1w"}, // 7 days normalizes to one week
		{Week + 2*Day, "1w2d"},
		{45 * time.Second, "45s"},
	}
	for _, c := range cases {
		if got := Human(c.in); got != c.want {
			t.Errorf("Human(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
