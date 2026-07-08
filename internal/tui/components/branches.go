package components

import (
	"fmt"
	"strings"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

// RenderBranchDetail renders contextual information for the selected branch.
func RenderBranchDetail(detail *gitpkg.BranchDetail, base string, width int) string {
	if detail == nil {
		return theme.S.Hint.Render("  Carregando…")
	}

	var lines []string
	if base == "" {
		base = "main"
	}

	headLine := detail.HeadHash
	if headLine == "" {
		headLine = "—"
	}
	lines = append(lines, theme.S.Hint.Render("  HEAD "+headLine))

	if detail.Info.Upstream != "" {
		sync := detail.Info.Upstream
		if detail.Info.Ahead > 0 || detail.Info.Behind > 0 {
			sync += fmt.Sprintf("  ↑%d ↓%d", detail.Info.Ahead, detail.Info.Behind)
		}
		lines = append(lines, theme.S.Hint.Render("  "+sync))
	}

	if detail.CommitsAheadOfBase > 0 {
		lines = append(lines, theme.S.Hint.Render(fmt.Sprintf(
			"  %d commit(s) ahead of %s",
			detail.CommitsAheadOfBase, base,
		)))
	} else if detail.Info.Name == base || strings.TrimSuffix(detail.Info.Name, "/") == base {
		lines = append(lines, theme.S.Hint.Render("  on base branch"))
	} else {
		lines = append(lines, theme.S.Hint.Render(fmt.Sprintf("  aligned with %s", base)))
	}

	if detail.FilesChanged > 0 {
		lines = append(lines, theme.S.Success.Render(fmt.Sprintf("  +%d", detail.Insertions))+
			theme.S.Hint.Render(" · ")+
			theme.S.Error.Render(fmt.Sprintf("-%d", detail.Deletions))+
			theme.S.Hint.Render(fmt.Sprintf("  em %d arquivo(s) vs %s", detail.FilesChanged, base)))
	}

	if len(detail.RecentCommits) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.S.Hint.Render("  Recent commits:"))
		for _, c := range detail.RecentCommits {
			lines = append(lines, theme.S.Hint.Render("  ● "+c))
		}
	}

	body := strings.Join(lines, "\n")
	return RenderPanel("Branch Details", body, width)
}

// RenderBranchListLine renders a single branch entry for the picker list.
func RenderBranchListLine(info gitpkg.BranchInfo, selected bool) string {
	prefix := "  "
	if selected {
		prefix = theme.S.Current.Render("> ")
	}

	name := info.Name
	if info.Current {
		name = theme.S.Current.Render("* " + info.Name)
	} else if selected {
		name = theme.S.Current.Render(info.Name)
	} else {
		name = theme.S.Hint.Render(info.Name)
	}

	line := prefix + name
	if info.Upstream != "" {
		line += theme.S.Hint.Render("  → " + info.Upstream)
	}
	if info.Ahead > 0 || info.Behind > 0 {
		line += theme.S.Warn.Render(fmt.Sprintf("  ↑%d ↓%d", info.Ahead, info.Behind))
	}
	return line
}
