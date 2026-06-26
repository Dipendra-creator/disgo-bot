package ui

import (
	"fmt"
	"strings"
)

// ProgressBar renders a textual progress bar of the given width, e.g.
// "▰▰▰▰▰▱▱▱▱▱ 50%". current is clamped to [0, total]; width defaults to 10.
func ProgressBar(current, total, width int) string {
	if width <= 0 {
		width = 10
	}
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}

	ratio := float64(current) / float64(total)
	filled := int(ratio*float64(width) + 0.5)

	bar := strings.Repeat("▰", filled) + strings.Repeat("▱", width-filled)
	return fmt.Sprintf("%s %d%%", bar, int(ratio*100+0.5))
}
