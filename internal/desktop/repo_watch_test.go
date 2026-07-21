package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitWatchPaths_standardRepo(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "index"), []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	paths := gitWatchPaths(dir)
	if len(paths) == 0 {
		t.Fatal("expected at least HEAD")
	}
	foundHEAD := false
	for _, p := range paths {
		if filepath.Base(p) == "HEAD" {
			foundHEAD = true
		}
		rel, err := filepath.Rel(gitDir, p)
		if err != nil || strings.HasPrefix(rel, "..") {
			t.Fatalf("watch path outside .git: %s", p)
		}
	}
	if !foundHEAD {
		t.Fatalf("HEAD missing from %v", paths)
	}
}

func TestResolveGitDir_worktreeFile(t *testing.T) {
	root := t.TempDir()
	realGit := filepath.Join(root, "real.git")
	if err := os.MkdirAll(filepath.Join(realGit, "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realGit, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	gitFile := filepath.Join(work, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: "+realGit+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveGitDir(work)
	if err != nil {
		t.Fatal(err)
	}
	if got != realGit {
		t.Fatalf("got %q want %q", got, realGit)
	}
	paths := gitWatchPaths(work)
	if len(paths) == 0 {
		t.Fatal("expected watch paths for worktree")
	}
}

func TestStartRepoWatcher_close(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := StartRepoWatcher(dir, func() {})
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	w.Close() // idempotent
}
