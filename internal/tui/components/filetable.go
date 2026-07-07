package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
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

	inner := width - 4
	if inner < 40 {
		inner = 40
	}

	pathWidth := inner - 24
	if pathWidth < 20 {
		pathWidth = 20
	}

	header := fmt.Sprintf("%-4s %-*s %6s %6s", "TYPE", pathWidth, "FILE", "+", "-")
	var rows []string
	rows = append(rows, theme.S.Hint.Render(header))

	for _, f := range sorted[:limit] {
		tag := statusTag(f.Status)
		path := truncate(f.Path, pathWidth)
		plus := padNumber(theme.S.Success.Render(fmt.Sprintf("%d", f.Insertions)), 6)
		minus := padNumber(theme.S.Error.Render(fmt.Sprintf("%d", f.Deletions)), 6)
		row := fmt.Sprintf("%-4s %-*s %s %s", tag, pathWidth, path, plus, minus)
		rows = append(rows, fileRowStyle(f.Status).Render(row))
	}

	footer := fmt.Sprintf("Total: %d files", len(sorted))
	if len(sorted) > limit {
		footer += fmt.Sprintf(" (showing %d)", limit)
	}
	rows = append(rows, theme.S.Hint.Render(footer))

	body := strings.Join(rows, "\n")
	return RenderPanel("Changed Files", body, width)
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

func padNumber(colored string, width int) string {
	w := lipgloss.Width(colored)
	if w >= width {
		return colored
	}
	return strings.Repeat(" ", width-w) + colored
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
