package app

import (
	"testing"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
)

func TestCanAdd(t *testing.T) {
	snap := &WorkspaceSnapshot{
		Overview: &gitpkg.Overview{
			FileChanges: []gitpkg.FileChange{
				{Path: "new.go", Status: "untracked"},
				{Path: "old.go", Status: "staged"},
			},
		},
	}
	if !CanAdd(snap) {
		t.Fatal("expected CanAdd")
	}
	files := AddableFiles(snap)
	if len(files) != 1 || files[0].Path != "new.go" {
		t.Fatalf("AddableFiles = %+v", files)
	}
}

func TestCanAdd_none(t *testing.T) {
	snap := &WorkspaceSnapshot{
		Overview: &gitpkg.Overview{
			FileChanges: []gitpkg.FileChange{
				{Path: "old.go", Status: "staged"},
			},
		},
	}
	if CanAdd(snap) {
		t.Fatal("expected !CanAdd")
	}
}
