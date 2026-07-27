package git

import (
	"path/filepath"
	"testing"
)

func TestOverviewWithOpts_skipsHeavyCollectors(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	repo, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.IsRepo(); err != nil {
		t.Fatal(err)
	}

	full, err := repo.Overview("main")
	if err != nil {
		t.Fatal(err)
	}
	lite, err := repo.OverviewWithOpts("main", OverviewOpts{
		SkipBranches:      true,
		SkipStashes:       true,
		SkipRecentCommits: true,
		SkipFileNumstat:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if lite.Branch == "" || lite.HeadHash == "" {
		t.Fatalf("lite overview missing core fields: %+v", lite)
	}
	if lite.Branch != full.Branch {
		t.Fatalf("branch mismatch: lite=%q full=%q", lite.Branch, full.Branch)
	}
	if len(lite.Branches) != 0 {
		t.Fatalf("expected skipped branches, got %d", len(lite.Branches))
	}
	if len(lite.RecentCommits) != 0 {
		t.Fatalf("expected skipped recent commits, got %d", len(lite.RecentCommits))
	}
	for _, f := range lite.FileChanges {
		if f.Insertions != 0 || f.Deletions != 0 {
			t.Fatalf("expected no numstat on lite path, got %+v", f)
		}
	}
}
