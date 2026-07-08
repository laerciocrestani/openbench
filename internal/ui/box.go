package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/mattn/go-runewidth"
)

const (
	BoxTopGradientRatio    = 1.0
	BoxBottomGradientRatio = 1.0
	boxLeftPrefix          = "│ "
)

// BoxStyle customizes title and gradient rendering for a box frame.
type BoxStyle struct {
	Title      func(string) string
	TopDash    func(float64) string
	BottomDash func(float64) string
}

// PlainBoxStyle renders boxes without color styling.
func PlainBoxStyle() BoxStyle {
	plain := func(s string) string { return s }
	dash := func(float64) string { return "─" }
	return BoxStyle{Title: plain, TopDash: dash, BottomDash: dash}
}

// ContentInner returns the usable width inside a box content line.
func ContentInner(width int) int {
	inner := width - DisplayWidth(boxLeftPrefix)
	if inner < 1 {
		return 1
	}
	return inner
}

// DisplayWidth returns the visible width of a string, including styled text.
func DisplayWidth(s string) int {
	if w := lipgloss.Width(s); w > 0 || !strings.Contains(s, "\x1b") {
		return w
	}
	return runewidth.StringWidth(s)
}

// PadLine aligns left and right text within an inner width.
func PadLine(left, right string, inner int) string {
	return PadLineShaded(left, right, inner, 0, nil)
}

// PadLineShaded aligns left/right text and optionally shades the right column.
// rightColWidth fixes the shaded column width across multiple lines (0 = auto).
func PadLineShaded(left, right string, inner, rightColWidth int, shade func(string) string) string {
	leftW := DisplayWidth(left)
	rightW := DisplayWidth(right)
	if right == "" {
		return left + strings.Repeat(" ", maxInt(1, inner-leftW))
	}

	colW := rightColWidth
	if colW < rightW {
		colW = rightW
	}
	gap := inner - leftW - colW
	if gap < 1 {
		gap = 1
	}

	rightBlock := right + strings.Repeat(" ", colW-rightW)
	if shade != nil {
		rightBlock = shade(rightBlock)
	}
	return left + strings.Repeat(" ", gap) + rightBlock
}

// RightShadeStyle returns a subtle background style for right-aligned columns.
func RightShadeStyle(colorsEnabled bool) func(string) string {
	if !colorsEnabled {
		return nil
	}
	style := lipgloss.NewStyle().Background(lipgloss.Color("236"))
	return func(s string) string { return style.Render(s) }
}

// MaxDisplayWidth returns the widest visible string length.
func MaxDisplayWidth(parts ...string) int {
	maxW := 0
	for _, p := range parts {
		if w := DisplayWidth(p); w > maxW {
			maxW = w
		}
	}
	return maxW
}

// PadDisplayWidth pads or truncates a line to an exact display width.
func PadDisplayWidth(s string, width int) string {
	w := DisplayWidth(s)
	if w > width {
		return ansi.Truncate(s, width, "")
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

// RenderBoxTop renders the top border line: ╭ title ── gradient.
func RenderBoxTop(title string, width int, style BoxStyle) string {
	if style.Title == nil {
		style = PlainBoxStyle()
	}
	if style.TopDash == nil {
		style.TopDash = PlainBoxStyle().TopDash
	}

	gradientEnd := int(float64(width) * BoxTopGradientRatio)
	if gradientEnd < 1 {
		gradientEnd = 1
	}

	line := style.Title("╭ ") + style.Title(title) + style.Title(" ")
	pos := DisplayWidth(line)
	for pos < gradientEnd && pos < width {
		progress := 0.0
		if gradientEnd > pos {
			progress = float64(pos) / float64(gradientEnd)
		}
		line += style.TopDash(progress)
		pos++
	}
	return PadDisplayWidth(line, width)
}

// RenderBoxBottom renders the bottom border line: ╰ gradient fade.
func RenderBoxBottom(width int, style BoxStyle) string {
	if style.BottomDash == nil {
		style.BottomDash = PlainBoxStyle().BottomDash
	}

	gradientLen := int(float64(width) * BoxBottomGradientRatio)
	if gradientLen < 1 {
		gradientLen = 1
	}

	line := "╰"
	for i := 0; i < gradientLen; i++ {
		progress := float64(i) / float64(maxInt(gradientLen-1, 1))
		line += style.BottomDash(progress)
	}
	return PadDisplayWidth(line, width)
}

// RenderBoxLine renders a content row without a right border.
func RenderBoxLine(content string, width int) string {
	inner := ContentInner(width)
	w := DisplayWidth(content)
	if w > inner {
		content = ansi.Truncate(content, inner, "…")
	}
	line := boxLeftPrefix + content
	return PadDisplayWidth(line, width)
}

// RenderBox renders a full titled box with optional multiline body.
func RenderBox(title, body string, width int, style BoxStyle) string {
	if width < 20 {
		width = 78
	}

	var lines []string
	lines = append(lines, RenderBoxTop(title, width, style))

	if body == "" {
		lines = append(lines, RenderBoxLine("", width))
	} else {
		for _, line := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
			lines = append(lines, RenderBoxLine(line, width))
		}
	}
	lines = append(lines, RenderBoxBottom(width, style))
	return strings.Join(lines, "\n") + "\n"
}

// TopGradientDash renders a top-border dash with cyan fade.
func TopGradientDash(progress float64, colorsEnabled bool) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	if !colorsEnabled {
		return "─"
	}
	start := colorful.Color{R: 0.34, G: 0.84, B: 0.88}
	end := colorful.Color{R: 0.18, G: 0.18, B: 0.20}
	c := start.BlendLuv(end, progress)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c.Hex())).Render("─")
}

// BottomGradientDash renders a bottom-border dash with white-to-dark fade.
func BottomGradientDash(progress float64, colorsEnabled bool) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	if !colorsEnabled {
		return "─"
	}
	start := colorful.Color{R: 0.92, G: 0.92, B: 0.92}
	end := colorful.Color{R: 0.10, G: 0.10, B: 0.10}
	c := start.BlendLuv(end, progress)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c.Hex())).Render("─")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
