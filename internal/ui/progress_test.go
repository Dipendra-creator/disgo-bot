package ui_test

import (
	"strings"
	"testing"

	"github.com/dipu-sharma/disgo-bot/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestProgressBar(t *testing.T) {
	assert.Contains(t, ui.ProgressBar(5, 10, 10), "50%")
	assert.Contains(t, ui.ProgressBar(0, 10, 10), "0%")
	assert.Contains(t, ui.ProgressBar(10, 10, 10), "100%")

	// Over/under-flow is clamped.
	assert.Contains(t, ui.ProgressBar(20, 10, 10), "100%")
	assert.Contains(t, ui.ProgressBar(-5, 10, 10), "0%")

	// Width is respected (filled + empty runes == width).
	bar := ui.ProgressBar(3, 10, 10)
	glyphs := strings.Count(bar, "▰") + strings.Count(bar, "▱")
	assert.Equal(t, 10, glyphs)
}
