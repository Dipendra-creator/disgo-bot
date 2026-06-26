package utility_test

import (
	"testing"
	"time"

	"github.com/dipu-sharma/disgo-bot/modules/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountAge(t *testing.T) {
	// Snowflake created 2016-04-30T11:18:25.796Z.
	now := time.Date(2017, 4, 30, 11, 18, 25, 796_000_000, time.UTC)
	age, err := utility.AccountAge("175928847299117063", now)
	require.NoError(t, err)

	// Roughly one year (365 days) within a small tolerance.
	assert.InDelta(t, (365 * 24 * time.Hour).Hours(), age.Hours(), 1)
}

func TestAccountAgeInvalid(t *testing.T) {
	_, err := utility.AccountAge("bad", time.Now())
	assert.Error(t, err)
}
