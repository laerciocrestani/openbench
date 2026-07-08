package components

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/app"
	prpkg "github.com/laerciocrestani/gitai/internal/pr"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

// FooterItem represents a keyboard shortcut in the footer bar.
type FooterItem struct {
	Key     string
	Label   string
	Enabled bool
}

// RenderFooter renders the modern shortcut bar.
func RenderFooter(items []FooterItem, width int) string {
	var parts []string
	for _, item := range items {
		key := "[" + item.Key + "]"
		label := item.Label
		if item.Enabled {
			parts = append(parts, theme.S.Key.Render(key)+theme.S.Hint.Render(" "+label))
		} else {
			parts = append(parts, theme.S.Disabled.Render(key+" "+label))
		}
	}
	sep := theme.S.Hint.Render(" │ ")
	line := strings.Join(parts, sep)
	if width > 0 {
		return theme.S.StatusBar.Width(width).Render(line)
	}
	return theme.S.StatusBar.Render(line)
}

// DefaultFooterItems builds the standard dashboard footer shortcuts.
func DefaultFooterItems(snap *app.WorkspaceSnapshot) []FooterItem {
	commitEnabled := false
	pushEnabled := false
	prEnabled := false
	syncEnabled := false

	if snap != nil && snap.Overview != nil && snap.ConfigErr == nil {
		o := snap.Overview
		commitEnabled = o.IsDirty()
		pushEnabled = app.CanPush(snap)
		prEnabled = app.CanPR(snap)
		syncEnabled = o.Behind > 0
	}

	items := []FooterItem{
		{Key: "c", Label: "Commit", Enabled: commitEnabled},
		{Key: "p", Label: "Push", Enabled: pushEnabled},
		{Key: "P", Label: "PR", Enabled: prEnabled},
		{Key: "d", Label: "Diff", Enabled: true},
		{Key: "y", Label: "Copy hash", Enabled: snap != nil && snap.Overview != nil && snap.Overview.HeadHash != ""},
		{Key: "l", Label: "Logs", Enabled: len(snapSafeCommits(snap)) > 0},
		{Key: "?", Label: "Help", Enabled: true},
		{Key: "q", Label: "Quit", Enabled: true},
	}

	if syncEnabled {
		syncItem := FooterItem{Key: "s", Label: "Sync", Enabled: true}
		items = append(items[:3], append([]FooterItem{syncItem}, items[3:]...)...)
	}

	return items
}

func snapSafeCommits(snap *app.WorkspaceSnapshot) []string {
	if snap == nil || snap.Overview == nil {
		return nil
	}
	return snap.Overview.RecentCommits
}

// RenderOpenPR renders an open PR info line when present.
func RenderOpenPR(pr *prpkg.PRView, width int) string {
	if pr == nil {
		return ""
	}
	state := strings.ToLower(pr.State)
	if pr.IsDraft {
		state = "draft"
	}
	line := fmt.Sprintf("PR #%d %s (%s)", pr.Number, pr.Title, state)
	if len(line) > width-4 {
		line = line[:width-5] + "…"
	}
	return theme.S.Info.Render(line) + "\n"
}
