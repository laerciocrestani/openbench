package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestCanPush_ahead(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{Ahead: 2},
	}
	if !app.CanPush(snap) {
		t.Fatal("expected push available when ahead")
	}
}

func TestCanPush_notConfigured(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview:  &gitpkg.Overview{Ahead: 1},
		ConfigErr: fmt.Errorf("missing"),
	}
	if app.CanPush(snap) {
		t.Fatal("expected push blocked without config")
	}
}

func TestCanPR_requiresGH(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview:  &gitpkg.Overview{CommitsAheadOfBase: 3},
		HasGH:     false,
		ConfigErr: nil,
	}
	if app.CanPR(snap) {
		t.Fatal("expected PR blocked without gh")
	}
}

func TestCanPR_withGH(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview:  &gitpkg.Overview{CommitsAheadOfBase: 1},
		HasGH:     true,
		ConfigErr: nil,
	}
	if !app.CanPR(snap) {
		t.Fatal("expected PR available")
	}
}

func TestParseDashboardKey_help(t *testing.T) {
	k, ok := parseDashboardKey(keyRunes("?"), nil)
	if !ok || k != dashKeyHelp {
		t.Fatalf("key=%v ok=%v", k, ok)
	}
}

func TestParseDashboardKey_report(t *testing.T) {
	k, ok := parseDashboardKey(keyRunes("u"), nil)
	if !ok || k != dashKeyReport {
		t.Fatalf("key=%v ok=%v", k, ok)
	}
}

func TestParseDashboardKey_commit(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{Modified: 1},
	}
	k, ok := parseDashboardKey(keyRunes("c"), snap)
	if !ok || k != dashKeyCommit {
		t.Fatalf("key=%v ok=%v", k, ok)
	}
}

func TestParseDashboardKey_commit_cleanTree(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{},
	}
	_, ok := parseDashboardKey(keyRunes("c"), snap)
	if ok {
		t.Fatal("expected no commit on clean tree")
	}
}

func TestParseDashboardKey_lowercaseActions(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{
			Ahead:         1,
			HeadHash:      "abc",
			RecentCommits: []string{"x"},
			Branches:      []gitpkg.BranchInfo{{Name: "main", Current: true}},
		},
	}
	cases := []struct {
		key  string
		want dashKey
	}{
		{"p", dashKeyPush},
		{"d", dashKeyDiff},
		{"y", dashKeyCopyHash},
		{"l", dashKeyLogs},
		{"b", dashKeyBranches},
	}
	for _, tc := range cases {
		k, ok := parseDashboardKey(keyRunes(tc.key), snap)
		if !ok || k != tc.want {
			t.Fatalf("key %q: got %v ok=%v want %v", tc.key, k, ok, tc.want)
		}
	}
}

func TestShouldLaunch_respectsGITAI_NO_UI(t *testing.T) {
	t.Setenv("GITAI_NO_UI", "1")
	t.Setenv("CI", "")
	if ShouldLaunch() {
		t.Fatal("expected false with GITAI_NO_UI")
	}
}

func TestTerminalTooSmall(t *testing.T) {
	if !terminalTooSmall(40, 10) {
		t.Fatal("expected small terminal")
	}
	if terminalTooSmall(100, 30) {
		t.Fatal("expected large terminal")
	}
}
