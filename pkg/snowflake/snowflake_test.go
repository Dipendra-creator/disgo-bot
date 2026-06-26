package snowflake_test

import (
	"testing"

	"github.com/dipu-sharma/disgo-bot/pkg/snowflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimestamp(t *testing.T) {
	// Reference snowflake from Discord's documentation.
	ts, err := snowflake.Timestamp("175928847299117063")
	require.NoError(t, err)
	assert.Equal(t, int64(1462015105796), ts.UnixMilli())
	assert.Equal(t, 2016, ts.Year())
}

func TestTimestampInvalid(t *testing.T) {
	_, err := snowflake.Timestamp("not-a-number")
	assert.Error(t, err)
}
