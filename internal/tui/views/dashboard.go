package views

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

// DashboardOptions configures dashboard rendering.
type DashboardOptions struct {
	Width  int
	Height int
	Tick   int
}

// RenderDashboard builds the full dashboard view from a workspace snapshot.
func RenderDashboard(snap *app.WorkspaceSnapshot, opts DashboardOptions) string {
	if snap == nil || snap.Overview == nil {
		return ""
	}

	width := opts.Width
	if width < 40 {
		width = 78
	}

	o := snap.Overview
	var b strings.Builder

	b.WriteString(components.RenderOpenPR(snap.OpenPR, width))

	summary := app.BuildChangeSummary(o)
	b.WriteString(components.RenderGitGraph(o, width))

	if summary.FileCount > 0 {
		b.WriteString(components.RenderSummary(summary, width))
	}

	maxFiles := fileLimit(opts.Height)
	if len(o.FileChanges) > 0 {
		b.WriteString(components.RenderFileTable(o.FileChanges, width, maxFiles))
	}

	if len(o.RecentCommits) > 0 {
		b.WriteString(components.RenderCommits(o.RecentCommits, width))
	}

	if len(o.Stashes) > 0 {
		b.WriteString(components.RenderStash(o.Stashes, width))
	}

	b.WriteString(components.RenderAIPanel(snap, width))

	action := app.BuildTUINextAction(snap)
	b.WriteString(components.RenderNextAction(action, width))

	if !snap.HasGH {
		b.WriteString(components.RenderPanel("Note", "install gh for PR info — https://cli.github.com/", width))
	}

	return b.String()
}

// RenderLoadingDashboard shows a loading panel while fetching snapshot data.
func RenderLoadingDashboard(message string, alerts []string, percent, width int) string {
	return components.RenderLoading(message, alerts, percent, width)
}

func fileLimit(height int) int {
	if height <= 0 {
		return 12
	}
	limit := height/3 - 2
	if limit < 6 {
		return 6
	}
	if limit > 20 {
		return 20
	}
	return limit
}

// RenderFooterBar renders the bottom shortcut bar for the dashboard.
func RenderFooterBar(snap *app.WorkspaceSnapshot, width int) string {
	items := components.DefaultFooterItems(snap)
	return "\n" + components.RenderFooter(items, width)
}

// FormatStatusLeft returns the left side of the status bar.
func FormatStatusLeft(status string) string {
	return status
}

// FormatStatusRight returns contextual help for non-dashboard screens.
func FormatStatusRight(screen string, snap *app.WorkspaceSnapshot) string {
	switch screen {
	case "dashboard":
		return ""
	default:
		return ""
	}
}

// DebugSummary returns a one-line summary for tests.
func DebugSummary(snap *app.WorkspaceSnapshot) string {
	if snap == nil || snap.Overview == nil {
		return ""
	}
	o := snap.Overview
	return fmt.Sprintf("%s@%s dirty=%v", o.Branch, o.HeadHash, o.IsDirty())
}
