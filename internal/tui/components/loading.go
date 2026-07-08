package components

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

// RenderLoading renders a loading state with step-based progress.
func RenderLoading(message string, alerts []string, percent, width int) string {
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
	for _, alert := range alerts {
		lines = append(lines, styleAlertLine(alert))
	}

	body := strings.Join(lines, "\n")
	return RenderPanel("Loading", body, width)
}

func styleAlertLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if strings.HasPrefix(line, "✖") || strings.HasPrefix(line, "✗") {
		return theme.S.Error.Render("  " + line)
	}
	if strings.HasPrefix(line, "✓") {
		return theme.S.Success.Render("  " + line)
	}
	return theme.S.Warn.Render("  " + line)
}

func AlertLogs(logs []string) []string {
	var alerts []string
	for _, line := range logs {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "✖") || strings.HasPrefix(trimmed, "✗") {
			alerts = append(alerts, trimmed)
		}
	}
	return alerts
}
