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
	statsColPlus  = 6
	statsColMinus = 6
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
	statsWidth := statsColPlus + 1 + statsColMinus
	pathWidth := inner - 4 - 1 - statsWidth
	if pathWidth < 20 {
		pathWidth = 20
	}

	shade := rightShadeStyle(limit > 1)
	rows := []string{fileTableRow("TYPE", "FILE", "+", "-", pathWidth, statsWidth, inner, shade, true)}

	for _, f := range sorted[:limit] {
		rows = append(rows, fileTableRow(
			statusTag(f.Status),
			truncate(f.Path, pathWidth),
			fmt.Sprintf("+%d", f.Insertions),
			fmt.Sprintf("-%d", f.Deletions),
			pathWidth,
			statsWidth,
			inner,
			shade,
			false,
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

func fileTableRow(tag, path, plus, minus string, pathWidth, statsWidth, inner int, shade func(string) string, header bool, status ...string) string {
	var plusStyled, minusStyled string
	if header {
		plusStyled = theme.S.Hint.Render(padPlain("+", statsColPlus))
		minusStyled = theme.S.Hint.Render(padPlain("-", statsColMinus))
	} else {
		plusStyled = theme.S.Success.Render(padPlain(plus, statsColPlus))
		minusStyled = theme.S.Error.Render(padPlain(minus, statsColMinus))
	}
	right := formatStatsBlock(plusStyled, minusStyled)
	left := fmt.Sprintf("%-4s %s", tag, padPlain(path, pathWidth))

	var row string
	if shade != nil {
		row = PadLineShaded(left, right, inner, statsWidth, shade)
	} else {
		row = PadLine(left, right, inner)
	}
	if header {
		return row
	}
	st := "modified"
	if len(status) > 0 {
		st = status[0]
	}
	return fileRowStyle(st).Render(row)
}

func rightShadeStyle(enabled bool) func(string) string {
	if !enabled || !uiprefs.ColorsEnabled() {
		return nil
	}
	return func(s string) string { return theme.S.RightShade.Render(s) }
}

func formatStatsBlock(plus, minus string) string {
	return PadLine(plus, minus, statsColPlus+1+statsColMinus)
}

func padPlain(s string, width int) string {
	w := ui.DisplayWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
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
