package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/mattn/go-runewidth"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
	"github.com/laerciocrestani/gitai/internal/uiprefs"
)

const (
	panelDecorSolidEnd = 10
	panelDecorFadeEnd  = 30
)

// RenderPanel renders a titled panel with optional body content.
func RenderPanel(title, body string, width int) string {
	if width < 20 {
		width = 78
	}
	inner := width - 4

	titleLine := buildPanelTitleLine(title, width)
	var lines []string
	lines = append(lines, theme.S.PanelTitle.Render(titleLine))

	if body == "" {
		lines = append(lines, theme.S.Panel.Render(boxEmpty(inner)))
	} else {
		for _, line := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
			lines = append(lines, theme.S.Panel.Render(boxContent(line, inner)))
		}
	}
	lines = append(lines, theme.S.Panel.Render(boxBottom(width)))
	return strings.Join(lines, "\n") + "\n"
}

func buildPanelTitleLine(title string, width int) string {
	var b strings.Builder
	b.WriteString("╭ ")
	b.WriteString(title)
	b.WriteString(" ")

	pos := runewidth.StringWidth(b.String())
	for pos < width-1 {
		if pos >= panelDecorFadeEnd {
			break
		}
		if pos >= panelDecorSolidEnd {
			progress := float64(pos-panelDecorSolidEnd) / float64(panelDecorFadeEnd-panelDecorSolidEnd)
			b.WriteString(gradientDash(progress))
		} else {
			b.WriteString(solidDash())
		}
		pos++
	}

	line := b.String()
	for runewidth.StringWidth(line) < width-1 {
		line += " "
	}
	return line + "╮"
}

func solidDash() string {
	if uiprefs.ColorsEnabled() {
		return theme.S.PanelTitle.Render("─")
	}
	return "─"
}

func gradientDash(progress float64) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	if !uiprefs.ColorsEnabled() {
		return "─"
	}
	start := colorful.Color{R: 0.34, G: 0.84, B: 0.88}
	end := colorful.Color{R: 0.18, G: 0.18, B: 0.20}
	c := start.BlendLuv(end, progress)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c.Hex())).Render("─")
}

func boxEmpty(inner int) string {
	padding := inner
	if padding < 0 {
		padding = 0
	}
	return "│ " + strings.Repeat(" ", padding) + " │"
}

func boxContent(content string, inner int) string {
	w := runewidth.StringWidth(content)
	if w > inner {
		content = truncate(content, inner)
		w = runewidth.StringWidth(content)
	}
	pad := inner - w
	if pad < 0 {
		pad = 0
	}
	return "│ " + content + strings.Repeat(" ", pad) + " │"
}

func boxBottom(width int) string {
	return "╰" + strings.Repeat("─", width-2) + "╯"
}

func truncate(s string, max int) string {
	return runewidth.Truncate(s, max, "…")
}

// RenderDivider renders a horizontal divider spanning the given width.
func RenderDivider(width int) string {
	if width < 4 {
		width = 78
	}
	return theme.S.Hint.Render("├"+strings.Repeat("─", width-2)+"┤") + "\n"
}

// PadLine pads content to the given display width.
func PadLine(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
