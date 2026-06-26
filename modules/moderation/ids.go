package moderation

import "strconv"

// pid parses a Discord snowflake string into the int64 used by the schema.
// A malformed ID yields 0, which never matches a real snowflake.
func pid(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// sid formats a stored int64 snowflake back into its string form.
func sid(n int64) string { return strconv.FormatInt(n, 10) }

// isSnowflake reports whether s looks like a Discord ID (17-20 digits).
func isSnowflake(s string) bool {
	if len(s) < 17 || len(s) > 20 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
