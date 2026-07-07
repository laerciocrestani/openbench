package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
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

	var lines []string
	top := boxTop(width)
	lines = append(lines, top)

	titleLeft := paint("GITAI", bold+cyan)
	tagline := paint("AI Git Workflow", dim)
	version := Version()
	if dryRun {
		version += " · dry-run"
	}
	version = paint(version, dim)
	titleLine := fitLine([]string{titleLeft, tagline, version}, width-4)
	lines = append(lines, boxRow(titleLine, width))

	lines = append(lines, boxDivider(width))

	inner := width - 4
	if ctx != nil {
		lines = append(lines, boxRow(headerMetaRow("Repository", ctx.Repo, ctx.Status, inner, paint), width))
		lines = append(lines, boxRow(headerMetaRow("Branch", ctx.Branch, ctx.Sync, inner, paint), width))

		aiLabel := formatProviderModel(ctx.Provider, ctx.Model)
		aiStatus := aiStatusLabel(ctx.AIReady)
		if colorsEnabled {
			if ctx.AIReady {
				aiStatus = paint(aiStatus, green)
			} else {
				aiStatus = paint(aiStatus, yellow)
			}
		}
		lines = append(lines, boxRow(headerMetaRow("AI", aiLabel, aiStatus, inner, paint), width))

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
		lines = append(lines, boxRow(headerMetaRow("Commit", commitValue, commitNote, inner, paint), width))
	} else {
		fallback := "AI Git Workflow · " + Version()
		if dryRun {
			fallback += " · dry-run"
		}
		lines = append(lines, boxRow(paint(fallback, dim), width))
	}

	lines = append(lines, boxBottom(width))
	return strings.Join(lines, "\n") + "\n"
}

// FormatBanner renders the dashboard header (replaces the legacy ASCII banner).
func FormatBanner(dryRun bool, ctx *BannerContext, colorsEnabled bool) string {
	return FormatDashboardHeader(ctx, defaultHeaderWidth, dryRun, colorsEnabled)
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

func headerMetaRow(label, value, right string, innerWidth int, paint func(string, string) string) string {
	labelPart := paint(fmt.Sprintf("%-10s", label+":"), dim)
	val := value
	if val == "" {
		val = "—"
	}
	if right != "" {
		rightW := runewidth.StringWidth(right)
		labelW := runewidth.StringWidth(labelPart) + 1
		maxVal := innerWidth - labelW - rightW - 1
		if maxVal < 1 {
			maxVal = 1
		}
		if runewidth.StringWidth(val) > maxVal {
			val = truncateRunewidth(val, maxVal)
		}
	}
	left := labelPart + " " + val
	if right == "" {
		return padToWidth(left, innerWidth)
	}
	gap := innerWidth - runewidth.StringWidth(left) - runewidth.StringWidth(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func fitLine(parts []string, width int) string {
	if len(parts) == 0 {
		return strings.Repeat(" ", width)
	}
	if len(parts) == 1 {
		return padToWidth(parts[0], width)
	}

	left := parts[0]
	right := parts[len(parts)-1]
	mid := strings.Join(parts[1:len(parts)-1], "  ")

	rightW := runewidth.StringWidth(right)
	if rightW >= width {
		return truncateRunewidth(right, width)
	}

	leftBlock := left
	if mid != "" {
		leftBlock = left + "  " + mid
	}

	gap := width - runewidth.StringWidth(leftBlock) - rightW
	if gap < 1 {
		leftBlock = truncateRunewidth(leftBlock, width-rightW-1)
		gap = 1
	}
	return leftBlock + strings.Repeat(" ", gap) + right
}

func padToWidth(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func boxTop(width int) string {
	return "╭" + strings.Repeat("─", width-2) + "╮"
}

func boxBottom(width int) string {
	return "╰" + strings.Repeat("─", width-2) + "╯"
}

func boxDivider(width int) string {
	return "├" + strings.Repeat("─", width-2) + "┤"
}

func boxRow(content string, width int) string {
	inner := width - 4
	if runewidth.StringWidth(content) > inner {
		content = truncateRunewidth(content, inner)
	}
	padding := inner - runewidth.StringWidth(content)
	if padding < 0 {
		padding = 0
	}
	return "│ " + content + strings.Repeat(" ", padding) + " │"
}

func truncateRunewidth(s string, max int) string {
	return runewidth.Truncate(s, max, "…")
}
