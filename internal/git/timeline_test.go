package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadTimelineCommits(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "dev@example.com")
	run("config", "user.name", "Dev")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "first")

	repo, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	commits, err := repo.LoadTimelineCommits(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) < 1 {
		t.Fatal("expected at least one commit")
	}
	if commits[0].Subject != "first" {
		t.Fatalf("subject=%q", commits[0].Subject)
	}
}

func TestParseDecorations(t *testing.T) {
	got := parseDecorations("HEAD -> main, origin/main, tag: v1.0.0")
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
}
