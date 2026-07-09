package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBranchAbsorbedIntoBase_squash(t *testing.T) {
	root := filepath.Join("..", "..", ".tmp", "squash-test")
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("integration fixture missing")
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(prev)

	repo, err := New()
	if err != nil {
		t.Fatal(err)
	}

	absorbed, err := repo.BranchAbsorbedIntoBase("feature/foo", "main")
	if err != nil {
		t.Fatal(err)
	}
	if !absorbed {
		t.Fatal("expected feature/foo to be absorbed into main after squash merge")
	}

	absorbed, err = repo.BranchAbsorbedIntoBase("feature/wip", "main")
	if err != nil {
		t.Fatal(err)
	}
	if absorbed {
		t.Fatal("expected feature/wip to have unique commits")
	}

	candidates, err := repo.LocalPruneCandidates("main")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, name := range candidates {
		if name == "feature/foo" {
			found = true
		}
		if name == "feature/wip" {
			t.Fatalf("feature/wip should not be pruned, candidates=%#v", candidates)
		}
	}
	if !found {
		t.Fatalf("expected feature/foo in candidates, got %#v", candidates)
	}
}

func TestLocalBranchesWithGoneUpstream_forEachRef(t *testing.T) {
	root := filepath.Join("..", "..", ".tmp", "prune-test")
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("integration fixture missing")
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(prev)

	repo, err := New()
	if err != nil {
		t.Fatal(err)
	}

	branches, err := repo.LocalBranchesWithGoneUpstream("main")
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 1 || branches[0] != "feature/foo" {
		t.Fatalf("branches = %#v, want [feature/foo]", branches)
	}
}
