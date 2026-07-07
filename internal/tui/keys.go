package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
)

type dashKey int

const (
	dashKeyNone dashKey = iota
	dashKeyCommit
	dashKeyPush
	dashKeyPR
	dashKeyDiff
	dashKeySync
	dashKeyOpenPR
	dashKeyCopyHash
	dashKeyReport
	dashKeyHelp
)

func parseGlobalKey(msg tea.KeyMsg) (keyMsg, bool) {
	switch msg.String() {
	case "q", "ctrl+c":
		return keyQuit, true
	case "r":
		return keyRefresh, true
	}
	return 0, false
}

func parseDashboardKey(msg tea.KeyMsg, snap *app.WorkspaceSnapshot) (dashKey, bool) {
	switch msg.String() {
	case "?":
		return dashKeyHelp, true
	case "u":
		return dashKeyReport, true
	case "c":
		if snap != nil && snap.Overview != nil && snap.Overview.IsDirty() && snap.ConfigErr == nil {
			return dashKeyCommit, true
		}
	case "p":
		if app.CanPush(snap) {
			return dashKeyPush, true
		}
	case "P", "shift+p":
		if app.CanPR(snap) {
			return dashKeyPR, true
		}
	case "d":
		return dashKeyDiff, true
	case "s":
		if snap != nil && snap.Overview != nil && snap.Overview.Behind > 0 {
			return dashKeySync, true
		}
	case "o":
		if snap != nil && snap.OpenPR != nil {
			return dashKeyOpenPR, true
		}
	case "y":
		if snap != nil && snap.Overview != nil && snap.Overview.HeadHash != "" {
			return dashKeyCopyHash, true
		}
	}
	return dashKeyNone, false
}

func dashboardHelpLine() string {
	return ""
}

type keyMsg int

const (
	keyRefresh keyMsg = iota
	keyQuit
)
