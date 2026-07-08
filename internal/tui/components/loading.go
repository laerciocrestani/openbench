package components

import (
	"strings"

	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

var loadingFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerFrame returns the braille frame for a tick counter.
func SpinnerFrame(tick int) string {
	if len(loadingFrames) == 0 {
		return "⠋"
	}
	if tick < 0 {
		tick = 0
	}
	return loadingFrames[tick%len(loadingFrames)]
}

// RenderSpinnerLine renders an animated spinner with a status message.
func RenderSpinnerLine(message string, tick int) string {
	if strings.TrimSpace(message) == "" {
		message = "Working…"
	}
	if !strings.HasSuffix(message, "…") && !strings.HasSuffix(message, "...") {
		message += "…"
	}
	return theme.S.Info.Render("  "+SpinnerFrame(tick)) + " " + theme.S.Hint.Render(message)
}

// RenderLoading renders an animated thinking spinner with the current status message.
func RenderLoading(message string, alerts []string, tick, width int) string {
	if width < 20 {
		width = 78
	}

	var lines []string
	lines = append(lines, RenderSpinnerLine(message, tick))
	for _, alert := range alerts {
		lines = append(lines, styleAlertLine(alert))
	}

	body := strings.Join(lines, "\n")
	return RenderPanel("Working", body, width)
}

// RenderActionDone renders a structured completion panel for long-running actions.
func RenderActionDone(title, summary string, logs []string, width int) string {
	var lines []string
	lines = append(lines, theme.S.Success.Render("  ✓ Concluído"))
	if summary != "" {
		lines = append(lines, theme.S.Current.Render("  "+summary))
	}

	stepLines := formatActionLogs(logs)
	if len(stepLines) > 0 {
		lines = append(lines, "")
		lines = append(lines, stepLines...)
	}

	lines = append(lines, "")
	lines = append(lines, theme.S.Hint.Render("  Enter para voltar"))

	body := strings.Join(lines, "\n")
	return RenderPanel(title, body, width)
}

func formatActionLogs(logs []string) []string {
	var out []string
	for _, line := range logs {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "✓"):
			out = append(out, theme.S.Success.Render("  "+line))
		case strings.HasPrefix(line, "✗"):
			out = append(out, theme.S.Error.Render("  "+line))
		case strings.HasPrefix(line, "✖"):
			out = append(out, theme.S.Warn.Render("  "+line))
		default:
			// Skip noisy git output lines that leaked before capture fix.
			if looksLikeGitNoise(line) {
				continue
			}
			out = append(out, theme.S.Hint.Render("  "+strings.TrimSpace(line)))
		}
	}
	return out
}

func looksLikeGitNoise(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	if strings.Contains(trimmed, "-> FETCH_HEAD") {
		return true
	}
	if strings.EqualFold(trimmed, "Already up to date.") {
		return true
	}
	if strings.HasPrefix(trimmed, "From ") && strings.Contains(trimmed, "origin") {
		return true
	}
	return false
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
