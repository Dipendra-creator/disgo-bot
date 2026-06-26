// Package duration parses and formats the compact, human-friendly durations
// moderators type into commands, e.g. "10m", "2h30m", "7d", "1w". Go's
// time.ParseDuration tops out at hours, so this adds days and weeks.
package duration

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// ErrEmpty is returned when the input contains no duration tokens.
var ErrEmpty = errors.New("duration: empty")

// Day and Week extend the standard time units.
const (
	Day  = 24 * time.Hour
	Week = 7 * Day
)

var unitMap = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': Day,
	'w': Week,
}

// Parse reads one or more <number><unit> tokens, where unit is s, m, h, d or w,
// and sums them. Whitespace is ignored and input is case-insensitive. Examples:
// "30s", "10m", "2h30m", "7d", "1w".
func Parse(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, ErrEmpty
	}

	var total time.Duration
	var num strings.Builder
	seen := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= '0' && ch <= '9':
			num.WriteByte(ch)
		case ch == ' ':
			continue
		default:
			unit, ok := unitMap[ch]
			if !ok {
				return 0, errors.New("duration: invalid unit '" + string(ch) + "'")
			}
			if num.Len() == 0 {
				return 0, errors.New("duration: unit without a number")
			}
			n, err := strconv.ParseInt(num.String(), 10, 64)
			if err != nil {
				return 0, errors.New("duration: number out of range")
			}
			total += time.Duration(n) * unit
			num.Reset()
			seen = true
		}
	}

	if num.Len() > 0 {
		return 0, errors.New("duration: trailing number without a unit")
	}
	if !seen {
		return 0, ErrEmpty
	}
	return total, nil
}

// Human renders a duration compactly, e.g. 90*time.Minute -> "1h30m". Sub-second
// durations render as "0s".
func Human(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	units := []struct {
		suffix string
		unit   time.Duration
	}{
		{"w", Week}, {"d", Day}, {"h", time.Hour}, {"m", time.Minute}, {"s", time.Second},
	}
	var b strings.Builder
	for _, u := range units {
		if d >= u.unit {
			n := d / u.unit
			d -= n * u.unit
			b.WriteString(strconv.FormatInt(int64(n), 10))
			b.WriteString(u.suffix)
		}
	}
	return b.String()
}
