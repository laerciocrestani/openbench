package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
	"github.com/laerciocrestani/gitai/internal/ui"
	"github.com/laerciocrestani/gitai/internal/uiprefs"
)

const (
	pathPad    = 3
	statsPad   = 3
	minDots    = 4
	tagWidth   = 4
)

// RenderFileTable renders changed files as an aligned table.
func RenderFileTable(changes []gitpkg.FileChange, width, maxRows int) string {
	if len(changes) == 0 {
		return ""
	}

	sorted := sortFileChanges(changes)

	if maxRows <= 0 {
		maxRows = 12
	}
	limit := len(sorted)
	if limit > maxRows {
		limit = maxRows
	}

	inner := ui.ContentInner(width)
	rows := []string{theme.S.Hint.Render(fmt.Sprintf("%-*s %s", tagWidth, "TYPE", "FILE"))}

	for _, f := range sorted[:limit] {
		rows = append(rows, buildFileRow(
			statusTag(f.Status),
			f.Path,
			f.Insertions,
			f.Deletions,
			inner,
			f.Status,
		))
	}

	footer := fmt.Sprintf("Total: %d files", len(sorted))
	if len(sorted) > limit {
		footer += fmt.Sprintf(" (showing %d)", limit)
	}
	rows = append(rows, theme.S.Hint.Render(footer))

	body := strings.Join(rows, "\n")
	return RenderPanel("Changed Files", body, width)
}

func buildFileRow(tag, path string, insertions, deletions, inner int, status string) string {
	right, rightW := buildStatsBlock(insertions, deletions)
	gapBeforeStats := strings.Repeat(" ", statsPad)
	gapAfterPath := strings.Repeat(" ", pathPad)

	fixedW := pathPad + statsPad + rightW
	maxPathW := inner - tagWidth - 1 - minDots - fixedW
	if maxPathW < 8 {
		maxPathW = 8
	}

	displayPath := path
	for {
		if ui.DisplayWidth(displayPath) > maxPathW {
			displayPath = truncate(displayPath, maxPathW)
		}
		left := fmt.Sprintf("%-*s %s", tagWidth, tag, displayPath)
		leftW := ui.DisplayWidth(left)
		if leftW+fixedW+minDots <= inner {
			break
		}
		if ui.DisplayWidth(displayPath) <= 1 {
			break
		}
		displayPath = truncate(displayPath, ui.DisplayWidth(displayPath)-1)
		maxPathW = ui.DisplayWidth(displayPath)
	}

	left := fmt.Sprintf("%-*s %s", tagWidth, tag, displayPath)
	leftStyled := fileRowStyle(status).Render(left)
	leftW := ui.DisplayWidth(leftStyled) + pathPad

	dots := inner - leftW - statsPad - rightW
	if dots < minDots {
		dots = minDots
	}

	dotsStyled := renderGradientDots(dots, uiprefs.ColorsEnabled())
	row := leftStyled + gapAfterPath + dotsStyled + gapBeforeStats + right
	return ui.PadDisplayWidth(row, inner)
}

func buildStatsBlock(insertions, deletions int) (string, int) {
	plus := theme.S.Success.Render(fmt.Sprintf("+%d", insertions))
	minus := theme.S.Error.Render(fmt.Sprintf("-%d", deletions))
	sep := theme.S.Hint.Render("·")
	right := plus + " " + sep + " " + minus
	return right, ui.DisplayWidth(right)
}

func renderGradientDots(count int, colorsEnabled bool) string {
	if count <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < count; i++ {
		progress := float64(i) / float64(maxInt(count-1, 1))
		b.WriteString(ui.GradientDot(progress, colorsEnabled))
	}
	return b.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sortFileChanges(changes []gitpkg.FileChange) []gitpkg.FileChange {
	sorted := make([]gitpkg.FileChange, len(changes))
	copy(sorted, changes)
	sort.Slice(sorted, func(i, j int) bool {
		ti := sorted[i].Insertions + sorted[i].Deletions
		tj := sorted[j].Insertions + sorted[j].Deletions
		if ti != tj {
			return ti > tj
		}
		return sorted[i].Path < sorted[j].Path
	})
	return sorted
}

func fileRowStyle(status string) lipgloss.Style {
	switch status {
	case "untracked":
		return theme.S.Untracked
	case "deleted":
		return theme.S.Error
	case "new", "staged":
		return theme.S.New
	case "modified", "staged+modified":
		return theme.S.Modified
	default:
		return theme.S.Hint
	}
}

func statusTag(status string) string {
	switch status {
	case "untracked":
		return "?"
	case "deleted":
		return "D"
	case "new", "staged":
		return "A"
	case "modified", "staged+modified":
		return "M"
	default:
		return "·"
	}
}
