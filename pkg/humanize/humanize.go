// Package humanize formats values for friendly display in Discord messages.
package humanize

import (
	"strconv"
	"time"
)

// Comma formats an integer with thousands separators, e.g. 1234567 -> "1,234,567".
func Comma(n int) string {
	s := strconv.Itoa(n)
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}

	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// Discord renders timestamps from a unix time plus a style suffix:
//
//	t  short time      f  short date/time
//	T  long time       F  long date/time
//	d  short date      R  relative ("3 days ago")
//	D  long date
//
// See https://discord.com/developers/docs/reference#message-formatting.

// TimeTag renders an absolute timestamp (long date/time) Discord localises per
// viewer.
func TimeTag(t time.Time) string {
	return "<t:" + strconv.FormatInt(t.Unix(), 10) + ":F>"
}

// RelativeTag renders a self-updating relative timestamp ("2 months ago").
func RelativeTag(t time.Time) string {
	return "<t:" + strconv.FormatInt(t.Unix(), 10) + ":R>"
}
