// Package snowflake decodes Discord snowflake IDs into their embedded creation
// timestamp. See https://discord.com/developers/docs/reference#snowflakes.
package snowflake

import (
	"fmt"
	"strconv"
	"time"
)

// discordEpoch is the first second of 2015 in milliseconds (Discord's epoch).
const discordEpoch int64 = 1420070400000

// Timestamp extracts the creation time encoded in a Discord snowflake ID.
func Timestamp(id string) (time.Time, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid snowflake %q: %w", id, err)
	}
	ms := (n >> 22) + discordEpoch
	return time.UnixMilli(ms).UTC(), nil
}
