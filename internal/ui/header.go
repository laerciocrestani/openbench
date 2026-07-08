package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// HeaderContext holds repository and AI status for the dashboard header.
type HeaderContext struct {
	Repo     string
	Branch   string
	HeadHash string
	Status   string
	Sync     string
	Provider string
	Model    string
	AIReady  bool
	OnBase   bool
}

// BannerContext is an alias kept for backward compatibility during migration.
type BannerContext = HeaderContext

const defaultHeaderWidth = 78

// FormatDashboardHeader renders a bordered dashboard-style header.
func FormatDashboardHeader(ctx *HeaderContext, width int, dryRun bool, colorsEnabled bool) string {
	if width < 40 {
		width = defaultHeaderWidth
	}

	paint := func(text, code string) string {
		if !colorsEnabled || code == "" {
			return text
		}
		return code + text + reset
	}

	style := headerBoxStyle(colorsEnabled)
	inner := ContentInner(width)
	var lines []string
	lines = append(lines, RenderBoxTop("GITAI", width, style))

	tagline := paint("AI Git Workflow", dim)
	version := Version()
	if dryRun {
		version += " · dry-run"
	}
	version = paint(version, dim)

	var rightParts []string
	if version != "" {
		rightParts = append(rightParts, version)
	}

	type metaLine struct {
		left  string
		right string
	}
	var metaLines []metaLine

	if ctx != nil {
		aiLabel := formatProviderModel(ctx.Provider, ctx.Model)
		aiStatus := aiStatusLabel(ctx.AIReady)
		if colorsEnabled {
			if ctx.AIReady {
				aiStatus = paint(aiStatus, green)
			} else {
				aiStatus = paint(aiStatus, yellow)
			}
		}

		commitNote := ""
		if ctx.OnBase {
			commitNote = "Main"
			if colorsEnabled {
				commitNote = paint("● "+commitNote, green)
			}
		}
		hashVal := ctx.HeadHash
		if hashVal == "" {
			hashVal = "—"
		}
		commitValue := hashVal
		if ctx.HeadHash != "" {
			commitValue += "  " + paint("⧉", dim)
		}

		metaLines = []metaLine{
			{headerMetaLeft("Repository", ctx.Repo, ctx.Status, inner, paint), ctx.Status},
			{headerMetaLeft("Branch", ctx.Branch, ctx.Sync, inner, paint), ctx.Sync},
			{headerMetaLeft("AI", aiLabel, aiStatus, inner, paint), aiStatus},
			{headerMetaLeft("Commit", commitValue, commitNote, inner, paint), commitNote},
		}
		for _, m := range metaLines {
			if m.right != "" {
				rightParts = append(rightParts, m.right)
			}
		}
	}

	shade := RightShadeStyle(colorsEnabled && len(rightParts) > 1)
	rightColW := MaxDisplayWidth(rightParts...)

	lines = append(lines, RenderBoxLine(formatHeaderRow(tagline, version, inner, rightColW, shade), width))

	for _, m := range metaLines {
		lines = append(lines, RenderBoxLine(formatHeaderRow(m.left, m.right, inner, rightColW, shade), width))
	}

	if ctx == nil {
		fallback := "AI Git Workflow · " + Version()
		if dryRun {
			fallback += " · dry-run"
		}
		lines = append(lines, RenderBoxLine(paint(fallback, dim), width))
	}

	lines = append(lines, RenderBoxBottom(width, style))
	return linesJoin(lines)
}

// FormatBanner renders the dashboard header (replaces the legacy ASCII banner).
func FormatBanner(dryRun bool, ctx *BannerContext, colorsEnabled bool) string {
	return FormatDashboardHeader(ctx, defaultHeaderWidth, dryRun, colorsEnabled)
}

func headerBoxStyle(colorsEnabled bool) BoxStyle {
	title := func(s string) string {
		if !colorsEnabled {
			return s
		}
		return bold + cyan + s + reset
	}
	return BoxStyle{
		Title: title,
		TopDash: func(p float64) string {
			return TopGradientDash(p, colorsEnabled)
		},
		BottomDash: func(p float64) string {
			return BottomGradientDash(p, colorsEnabled)
		},
	}
}

func formatHeaderRow(left, right string, inner, rightColW int, shade func(string) string) string {
	if right == "" || shade == nil {
		return PadLine(left, right, inner)
	}
	return PadLineShaded(left, right, inner, rightColW, shade)
}

func headerMetaLeft(label, value, right string, innerWidth int, paint func(string, string) string) string {
	labelPart := paint(fmt.Sprintf("%-10s", label+":"), dim)
	val := value
	if val == "" {
		val = "—"
	}
	if right == "" {
		return labelPart + " " + val
	}

	rightW := DisplayWidth(right)
	maxVal := innerWidth - DisplayWidth(labelPart) - 1 - rightW
	if maxVal < 1 {
		maxVal = 1
	}
	if DisplayWidth(val) > maxVal {
		val = truncateRunewidth(val, maxVal)
	}
	return labelPart + " " + val
}

func formatProviderModel(provider, model string) string {
	if provider == "" && model == "" {
		return "not configured"
	}
	if model == "" {
		return provider
	}
	if provider == "" {
		return model
	}
	display := provider
	if len(provider) > 0 {
		display = strings.ToUpper(provider[:1]) + provider[1:]
	}
	return display + " · " + model
}

func aiStatusLabel(ready bool) string {
	if ready {
		return "⚡ Ready"
	}
	return "⚠ Setup"
}

func truncateRunewidth(s string, max int) string {
	return ansi.Truncate(s, max, "…")
}

func linesJoin(lines []string) string {
	out := ""
	for i, line := range lines {
		out += line
		if i < len(lines)-1 {
			out += "\n"
		}
	}
	return out + "\n"
}
