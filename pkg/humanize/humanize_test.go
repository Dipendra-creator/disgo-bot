package humanize_test

import (
	"testing"
	"time"

	"github.com/dipu-sharma/disgo-bot/pkg/humanize"
	"github.com/stretchr/testify/assert"
)

func TestComma(t *testing.T) {
	cases := map[int]string{
		0:        "0",
		42:       "42",
		1000:     "1,000",
		1234567:  "1,234,567",
		-1000:    "-1,000",
		-1234567: "-1,234,567",
	}
	for in, want := range cases {
		assert.Equal(t, want, humanize.Comma(in), "Comma(%d)", in)
	}
}

func TestTimeTags(t *testing.T) {
	ts := time.Unix(1600000000, 0)
	assert.Equal(t, "<t:1600000000:F>", humanize.TimeTag(ts))
	assert.Equal(t, "<t:1600000000:R>", humanize.RelativeTag(ts))
}
