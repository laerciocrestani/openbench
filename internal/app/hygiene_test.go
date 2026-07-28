package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestEnsureHygieneOnBase_checkoutsBase(t *testing.T) {
	root := t.TempDir()
	gitRun(t, root, "init", "-b", "main")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "a.txt")
	gitRun(t, root, "commit", "-m", "init")
	gitRun(t, root, "checkout", "-b", "feature/old")
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "b.txt")
	gitRun(t, root, "commit", "-m", "feature")
	gitRun(t, root, "checkout", "main")
	gitRun(t, root, "merge", "--no-ff", "feature/old", "-m", "merge feature")
	gitRun(t, root, "checkout", "feature/old")

	repo, err := openRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordedProgress{}
	if err := ensureHygieneOnBase(rec, repo, "main", false); err != nil {
		t.Fatalf("ensureHygieneOnBase: %v", err)
	}
	cur, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if cur != "main" {
		t.Fatalf("current branch: got %q want main", cur)
	}
	joined := strings.Join(rec.steps, "|")
	if !strings.Contains(joined, "Switching to main before prune") {
		t.Fatalf("expected checkout step, got %v", rec.steps)
	}
}

func TestEnsureHygieneOnBase_noopOnBase(t *testing.T) {
	root := t.TempDir()
	gitRun(t, root, "init", "-b", "main")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "a.txt")
	gitRun(t, root, "commit", "-m", "init")

	repo, err := openRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recordedProgress{}
	if err := ensureHygieneOnBase(rec, repo, "main", false); err != nil {
		t.Fatalf("ensureHygieneOnBase: %v", err)
	}
	if len(rec.steps) != 0 {
		t.Fatalf("expected no checkout step on base, got %v", rec.steps)
	}
}

func TestRunHygiene_fromFeatureDeletesMergedBranch(t *testing.T) {
	root := t.TempDir()
	gitRun(t, root, "init", "-b", "main")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "a.txt")
	gitRun(t, root, "commit", "-m", "init")
	gitRun(t, root, "checkout", "-b", "feature/done")
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "b.txt")
	gitRun(t, root, "commit", "-m", "feature")
	gitRun(t, root, "checkout", "main")
	gitRun(t, root, "merge", "--no-ff", "feature/done", "-m", "merge feature")

	// Bare origin so fetch --prune succeeds.
	bare := filepath.Join(t.TempDir(), "origin.git")
	gitRun(t, root, "clone", "--bare", root, bare)
	gitRun(t, root, "remote", "add", "origin", bare)
	gitRun(t, root, "push", "-u", "origin", "main")

	// Stay on a different feature so hygiene must switch to main first.
	gitRun(t, root, "checkout", "-b", "feature/wip")

	rec := &recordedProgress{}
	if err := RunHygiene(HygieneOptions{
		Mode:     HygieneModeLocal,
		Base:     "main",
		WorkDir:  root,
		Progress: rec,
	}); err != nil {
		t.Fatalf("RunHygiene: %v\nsteps=%v\ndetails=%v", err, rec.steps, rec.details)
	}

	repo, err := openRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	cur, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if cur != "main" {
		t.Fatalf("after hygiene current=%q want main", cur)
	}
	out, err := exec.Command("git", "-C", root, "branch", "--list", "feature/done").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("expected feature/done deleted, still have %q; steps=%v", strings.TrimSpace(string(out)), rec.steps)
	}
	joined := strings.Join(append(rec.steps, rec.details...), "|")
	if !strings.Contains(joined, "Switching to main before prune") {
		t.Fatalf("expected checkout to main, got %v / %v", rec.steps, rec.details)
	}
}
