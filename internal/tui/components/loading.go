package components

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

// RenderLoading renders a loading state with step-based progress.
func RenderLoading(message string, percent, width int) string {
	if width < 20 {
		width = 78
	}
	barWidth := width - 8
	if barWidth > 40 {
		barWidth = 40
	}
	if barWidth < 10 {
		barWidth = 10
	}

	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := percent * barWidth / 100
	if percent > 0 && filled == 0 {
		filled = 1
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	var lines []string
	if message != "" {
		lines = append(lines, theme.S.Hint.Render(message))
	}
	lines = append(lines, theme.S.Info.Render(bar))
	lines = append(lines, theme.S.Hint.Render(fmt.Sprintf("%d%%", percent)))

	body := strings.Join(lines, "\n")
	return RenderPanel("Loading", body, width)
}
