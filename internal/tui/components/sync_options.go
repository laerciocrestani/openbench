package components

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/tui/theme"
	"github.com/laerciocrestani/gitai/internal/ui"
)

// SyncMode identifies a sync execution preset.
type SyncMode int

const (
	SyncModeStandard SyncMode = iota
	SyncModePruneRemote
	SyncModePruneFull
)

// SyncModeOption describes one sync preset with CLI flag and explanation.
type SyncModeOption struct {
	Mode        SyncMode
	Label       string
	Flag        string
	Summary     string
	Description string
	Prune       bool
	PruneRemote bool
}

// SyncModeCatalog returns all sync presets in display order.
func SyncModeCatalog() []SyncModeOption {
	return []SyncModeOption{
		{
			Mode:        SyncModeStandard,
			Label:       "Standard sync",
			Flag:        "(none)",
			Summary:     "Fetch + pull base branch",
			Description: "Updates remote refs (fetch --prune) and fast-forwards the base branch with origin. Does not remove merged branches.",
			Prune:       false,
			PruneRemote: false,
		},
		{
			Mode:        SyncModePruneRemote,
			Label:       "Sync + remote prune",
			Flag:        "--prune-remote",
			Summary:     "Sync + clean branches on GitHub",
			Description: "After sync, removes remote branches already merged into base (git push origin --delete). Keeps local branches.",
			Prune:       false,
			PruneRemote: true,
		},
		{
			Mode:        SyncModePruneFull,
			Label:       "Sync + full prune",
			Flag:        "--prune",
			Summary:     "Sync + clean local and remote",
			Description: "After sync, removes local and remote branches merged into base, with upstream gone, or already absorbed via squash/rebase. Divergent branches prompt before -D.",
			Prune:       true,
			PruneRemote: false,
		},
	}
}

// ToAppOptions maps the selected preset to app.SyncOptions fields.
func (o SyncModeOption) ToAppOptions(base string) (prune, pruneRemote bool, resolvedBase string) {
	return o.Prune, o.PruneRemote || o.Prune, base
}

// RenderSyncOptionsPanel renders the sync mode picker with a detail table.
func RenderSyncOptionsPanel(cursor int, modes []SyncModeOption, base string, dirty bool, width int) string {
	inner := ui.ContentInner(width)
	var lines []string

	if dirty {
		lines = append(lines, theme.S.Warn.Render("  ⚠ Dirty working tree — commit or stash before syncing"))
		lines = append(lines, "")
	}

	for i, mode := range modes {
		marker := "  "
		if i == cursor {
			marker = "> "
		}
		flag := mode.Flag
		if flag == "(none)" {
			flag = theme.S.Hint.Render("(none)")
		} else {
			flag = theme.S.Key.Render(flag)
		}
		label := mode.Label + "  " + flag
		if i == cursor {
			lines = append(lines, theme.S.Current.Render(marker+label))
		} else {
			lines = append(lines, theme.S.Hint.Render(marker+label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, theme.S.Hint.Render("  Base: "+base))
	lines = append(lines, "")

	selected := modes[cursor]
	lines = append(lines, renderSyncDetailTable(selected, base, inner))

	body := strings.Join(lines, "\n")
	return RenderPanel("Sync · Options", body, width)
}

func renderSyncDetailTable(mode SyncModeOption, base string, inner int) string {
	const colW = 14

	lines := []string{
		theme.S.Hint.Render(fmt.Sprintf("  %-*s %s", colW, "Option", mode.Label)),
		theme.S.Hint.Render(fmt.Sprintf("  %-*s %s", colW, "Flag", mode.Flag)),
		theme.S.Hint.Render(fmt.Sprintf("  %-*s %s", colW, "Summary", truncatePlain(mode.Summary, inner-colW-2))),
		"",
		theme.S.Hint.Render("  What it does"),
		"  " + wrapPlain(mode.Description, inner-2),
		"",
		theme.S.Hint.Render("  Commands"),
	}

	for _, cmd := range syncCommandPreview(mode, base) {
		lines = append(lines, theme.S.Hint.Render("  · "+cmd))
	}

	return strings.Join(lines, "\n")
}

func syncCommandPreview(mode SyncModeOption, base string) []string {
	if base == "" {
		base = "main"
	}
	cmds := []string{
		"git fetch origin --prune",
		"git checkout " + base,
		"git pull --ff-only origin " + base,
	}
	if mode.Prune || mode.PruneRemote {
		cmds = append(cmds, "git branch --merged "+base+" …")
		cmds = append(cmds, "git cherry "+base+" <branch> …")
	}
	if mode.Prune {
		cmds = append(cmds, "git branch -d/-D <merged-local> …")
		cmds = append(cmds, "git branch -D <gone-upstream> …")
		cmds = append(cmds, "git branch -D <squash-absorbed> …")
	}
	if mode.Prune || mode.PruneRemote {
		cmds = append(cmds, "git push origin --delete <merged-remote> …")
	}
	return cmds
}

// RenderSyncBaseEditor renders the base branch edit step.
func RenderSyncBaseEditor(baseField string, width int) string {
	body := theme.S.Hint.Render("  Base branch for pull and prune:\n\n  ") + baseField
	return RenderPanel("Sync · Base branch", body, width)
}

func wrapPlain(text string, width int) string {
	if width < 20 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var lines []string
	var current strings.Builder
	for _, word := range words {
		add := word
		if current.Len() > 0 {
			add = " " + word
		}
		if current.Len()+len(add) > width && current.Len() > 0 {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
			continue
		}
		current.WriteString(add)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "\n")
}
