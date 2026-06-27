package ai

import "strconv"

// pid parses a Discord snowflake string into the int64 used by the schema.
func pid(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// sid formats a stored int64 snowflake back into its string form.
func sid(n int64) string { return strconv.FormatInt(n, 10) }
