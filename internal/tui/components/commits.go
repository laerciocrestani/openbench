package components

import (
	"strings"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

const maxRecentCommits = 3

// RenderCommits renders recent commits in a panel.
func RenderCommits(commits []string, width int) string {
	if len(commits) == 0 {
		return ""
	}

	limit := len(commits)
	if limit > maxRecentCommits {
		limit = maxRecentCommits
	}

	var lines []string
	for _, c := range commits[:limit] {
		lines = append(lines, theme.S.Hint.Render(c))
	}
	return RenderPanel("Recent Commits", strings.Join(lines, "\n"), width)
}

// RenderStash renders stash entries when present.
func RenderStash(stashes []gitpkg.StashInfo, width int) string {
	if len(stashes) == 0 {
		return ""
	}

	limit := len(stashes)
	if limit > 5 {
		limit = 5
	}

	var lines []string
	for _, s := range stashes[:limit] {
		label := s.Ref
		if s.Branch != "" {
			label += " on " + s.Branch
		}
		if s.Message != "" {
			label += ": " + s.Message
		}
		lines = append(lines, theme.S.Hint.Render(label))
	}
	if len(stashes) > limit {
		lines = append(lines, theme.S.Hint.Render(fmtMore(len(stashes)-limit, "stash")))
	}
	return RenderPanel("Stash", strings.Join(lines, "\n"), width)
}

func fmtMore(n int, kind string) string {
	return "… +" + itoa(n) + " more " + kind + "(es)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
