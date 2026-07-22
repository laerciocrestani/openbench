package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCommitActivity_groupsByDay(t *testing.T) {
	root := t.TempDir()
	mustGit(t, root, "init")
	mustGit(t, root, "config", "user.email", "dev@example.com")
	mustGit(t, root, "config", "user.name", "Dev")

	mustWriteFile(t, filepath.Join(root, "a.txt"), "a")
	mustGit(t, root, "add", ".")
	mustGit(t, root, "commit", "-m", "first")

	mustWriteFile(t, filepath.Join(root, "b.txt"), "b")
	mustGit(t, root, "add", ".")
	mustGit(t, root, "commit", "-m", "second")

	repo, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	act, err := repo.LoadCommitActivity(30, true)
	if err != nil {
		t.Fatal(err)
	}
	if act.Total < 2 {
		t.Fatalf("total=%d, want >= 2", act.Total)
	}
	today := time.Now().Format("2006-01-02")
	found := false
	for _, d := range act.Days {
		if d.Date == today && d.Count >= 2 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected commits on %s in %+v", today, act.Days)
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
