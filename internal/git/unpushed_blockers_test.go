package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestScanUnpushedBlockers_largeAndJunk(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run("git", "init", "-b", "main")
	run("git", "commit", "--allow-empty", "-m", "base")

	// Simulate origin/main at empty base commit.
	run("git", "branch", "origin-main")

	big := filepath.Join(dir, "huge.dmg")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(101 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	_ = os.MkdirAll(filepath.Join(dir, "node_modules", "x"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "node_modules", "x", "a.js"), []byte("1"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "app.go"), []byte("package app\n"), 0o644)

	run("git", "add", "-f", "huge.dmg", "node_modules", ".env", "app.go")
	run("git", "commit", "-m", "bad commit")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Use the first commit as "remote" tip.
	base, err := repo.run("rev-parse", "origin-main")
	if err != nil {
		t.Fatal(err)
	}
	blockers, err := repo.ScanUnpushedBlockers(base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !blockers.HasBlockers() {
		t.Fatal("expected blockers")
	}
	if len(blockers.LargeFiles) == 0 {
		t.Fatalf("expected large file, got %#v", blockers)
	}
	foundJunk := false
	for _, p := range blockers.JunkPaths {
		if p == "node_modules" || p == ".env" {
			foundJunk = true
		}
	}
	if !foundJunk {
		t.Fatalf("expected junk paths, got %#v", blockers.JunkPaths)
	}
	roots := blockers.CachedRoots()
	if len(roots) == 0 {
		t.Fatal("expected cached roots")
	}
}
