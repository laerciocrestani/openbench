package tui

import (
	"testing"

	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
)

func TestSnapshotChanged_nil(t *testing.T) {
	if snapshotChanged(nil, nil) {
		t.Fatal("expected no change for both nil")
	}
	snap := &app.WorkspaceSnapshot{}
	if !snapshotChanged(nil, snap) {
		t.Fatal("expected change when one is nil")
	}
}

func TestSnapshotChanged_fileChanges(t *testing.T) {
	a := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{
			FileChanges: []gitpkg.FileChange{{Path: "a.go", Status: "modified"}},
		},
	}
	b := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{
			FileChanges: []gitpkg.FileChange{{Path: "a.go", Status: "modified", Insertions: 1}},
		},
	}
	if !snapshotChanged(a, b) {
		t.Fatal("expected change when insertions differ")
	}
}

func TestSnapshotChanged_branch(t *testing.T) {
	a := &app.WorkspaceSnapshot{Overview: &gitpkg.Overview{Branch: "main"}}
	b := &app.WorkspaceSnapshot{Overview: &gitpkg.Overview{Branch: "feat/x"}}
	if !snapshotChanged(a, b) {
		t.Fatal("expected change when branch differs")
	}
}

func TestSnapshotChanged_same(t *testing.T) {
	overview := &gitpkg.Overview{
		Branch:      "main",
		FileChanges: []gitpkg.FileChange{{Path: "a.go", Status: "modified", Insertions: 2}},
		RecentCommits: []string{"abc feat: x"},
	}
	a := &app.WorkspaceSnapshot{Overview: overview}
	b := &app.WorkspaceSnapshot{Overview: overview}
	if snapshotChanged(a, b) {
		t.Fatal("expected no change for identical snapshots")
	}
}

func TestOverviewChanged_counts(t *testing.T) {
	a := &gitpkg.Overview{Staged: 1}
	b := &gitpkg.Overview{Staged: 2}
	if !overviewChanged(a, b) {
		t.Fatal("expected change when staged count differs")
	}
}
