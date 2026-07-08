package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
	"github.com/laerciocrestani/gitai/internal/ui"
	"github.com/laerciocrestani/gitai/internal/uiprefs"
)

// RenderPanel renders a titled panel with optional body content.
func RenderPanel(title, body string, width int) string {
	return ui.RenderBox(title, body, width, panelBoxStyle())
}

func panelBoxStyle() ui.BoxStyle {
	colors := uiprefs.ColorsEnabled()
	title := func(s string) string {
		if !colors {
			return s
		}
		return theme.S.PanelTitle.Render(s)
	}
	return ui.BoxStyle{
		Title: title,
		TopDash: func(p float64) string {
			return ui.TopGradientDash(p, colors)
		},
		BottomDash: func(p float64) string {
			return ui.BottomGradientDash(p, colors)
		},
	}
}

// RenderDivider renders a horizontal divider spanning the given width.
func RenderDivider(width int) string {
	if width < 4 {
		width = 78
	}
	line := ui.PadDisplayWidth("├"+strings.Repeat("─", width-2), width)
	return theme.S.Hint.Render(line) + "\n"
}

// PadLine aligns left and right content within a box inner width.
func PadLine(left, right string, inner int) string {
	return ui.PadLine(left, right, inner)
}

func truncate(s string, max int) string {
	return ansi.Truncate(s, max, "…")
}
