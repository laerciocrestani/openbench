package tui

import (
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/uiprefs"
)

const watchDebounce = 400 * time.Millisecond

type pollRefreshMsg struct{}
type watchRefreshMsg struct{}

type refreshConfig struct {
	pollInterval time.Duration
	watchFiles   bool
}

func loadRefreshConfig() refreshConfig {
	return refreshConfig{
		pollInterval: uiprefs.AutoRefreshInterval(),
		watchFiles:   uiprefs.WatchFilesEnabled(),
	}
}

func initRefreshCmds(cfg refreshConfig) tea.Cmd {
	var cmds []tea.Cmd
	if cfg.pollInterval > 0 {
		cmds = append(cmds, schedulePollRefresh(cfg.pollInterval))
	}
	return tea.Batch(cmds...)
}

func schedulePollRefresh(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return pollRefreshMsg{}
	})
}

func loadSnapshotSilent() tea.Cmd {
	return func() tea.Msg {
		snap, err := app.LoadWorkspaceSnapshot()
		return snapshotMsg{snap: snap, err: err, silent: true}
	}
}

func (m appModel) canAutoRefresh() bool {
	if m.loading || m.action != nil {
		return false
	}
	switch m.screen {
	case ScreenDashboard, ScreenDiff:
		return true
	default:
		return false
	}
}

func (m appModel) requestAutoRefresh() (appModel, tea.Cmd) {
	if !m.canAutoRefresh() {
		return m, m.reschedulePollIfNeeded()
	}
	if m.refreshPending {
		return m, nil
	}
	m.refreshPending = true
	return m, tea.Tick(watchDebounce, func(time.Time) tea.Msg {
		return debouncedRefreshMsg{}
	})
}

func (m appModel) reschedulePollIfNeeded() tea.Cmd {
	if m.refresh.pollInterval <= 0 {
		return nil
	}
	return schedulePollRefresh(m.refresh.pollInterval)
}

type debouncedRefreshMsg struct{}

func snapshotChanged(a, b *app.WorkspaceSnapshot) bool {
	if a == nil || b == nil {
		return a != b
	}
	if a.ConfigErr != b.ConfigErr {
		return true
	}
	if (a.OpenPR == nil) != (b.OpenPR == nil) {
		return true
	}
	if a.OpenPR != nil && b.OpenPR != nil {
		if a.OpenPR.Number != b.OpenPR.Number || a.OpenPR.State != b.OpenPR.State || a.OpenPR.IsDraft != b.OpenPR.IsDraft {
			return true
		}
	}
	if a.Overview == nil || b.Overview == nil {
		return a.Overview != b.Overview
	}
	return overviewChanged(a.Overview, b.Overview)
}

func overviewChanged(a, b *gitpkg.Overview) bool {
	if a.Branch != b.Branch || a.Detached != b.Detached {
		return true
	}
	if a.Ahead != b.Ahead || a.Behind != b.Behind || a.CommitsAheadOfBase != b.CommitsAheadOfBase {
		return true
	}
	if a.Staged != b.Staged || a.Modified != b.Modified || a.Untracked != b.Untracked {
		return true
	}
	if len(a.FileChanges) != len(b.FileChanges) {
		return true
	}
	for i := range a.FileChanges {
		fa, fb := a.FileChanges[i], b.FileChanges[i]
		if fa.Path != fb.Path || fa.Status != fb.Status ||
			fa.Insertions != fb.Insertions || fa.Deletions != fb.Deletions {
			return true
		}
	}
	if len(a.RecentCommits) != len(b.RecentCommits) {
		return true
	}
	for i := range a.RecentCommits {
		if a.RecentCommits[i] != b.RecentCommits[i] {
			return true
		}
	}
	return false
}
