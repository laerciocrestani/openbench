package desktop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPull_emptyPath(t *testing.T) {
	if _, err := RunPull("", "main"); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestRunPull_featureUpdatesBase(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	local := filepath.Join(root, "local")

	mustRun(t, root, "git", "init", "--bare", remote)
	mustRun(t, root, "git", "clone", remote, local)
	mustRun(t, local, "git", "config", "user.email", "test@example.com")
	mustRun(t, local, "git", "config", "user.name", "Test")
	mustRun(t, local, "git", "checkout", "-b", "main")
	mustWrite(t, filepath.Join(local, "a.txt"), "a\n")
	mustRun(t, local, "git", "add", ".")
	mustRun(t, local, "git", "commit", "-m", "init")
	mustRun(t, local, "git", "push", "-u", "origin", "main")

	mustRun(t, local, "git", "checkout", "-b", "feat")
	mustWrite(t, filepath.Join(local, "feat.txt"), "f\n")
	mustRun(t, local, "git", "add", ".")
	mustRun(t, local, "git", "commit", "-m", "feat")
	mustRun(t, local, "git", "push", "-u", "origin", "feat")

	// Advance main on a second clone.
	other := filepath.Join(root, "other")
	mustRun(t, root, "git", "clone", remote, other)
	mustRun(t, other, "git", "config", "user.email", "test@example.com")
	mustRun(t, other, "git", "config", "user.name", "Test")
	mustRun(t, other, "git", "checkout", "main")
	mustWrite(t, filepath.Join(other, "b.txt"), "b\n")
	mustRun(t, other, "git", "add", ".")
	mustRun(t, other, "git", "commit", "-m", "main update")
	mustRun(t, other, "git", "push", "origin", "main")

	res, err := RunPull(local, "main")
	if err != nil {
		t.Fatalf("RunPull: %v", err)
	}
	if res == nil || res.Dashboard == nil {
		t.Fatal("missing result dashboard")
	}
	branch, err := runOut(t, local, "git", "branch", "--show-current")
	if err != nil || branch != "feat" {
		t.Fatalf("expected stay on feat, got %q (%v)", branch, err)
	}
	if _, err := os.Stat(filepath.Join(local, "b.txt")); err == nil {
		t.Fatal("b.txt should not appear on feat working tree without merge")
	}
	// Local main ref should include b.txt content via show.
	out, err := runOut(t, local, "git", "show", "main:b.txt")
	if err != nil || out != "b" {
		t.Fatalf("local main not updated: out=%q err=%v", out, err)
	}
	if res.Dashboard.BaseBehind != 0 {
		t.Fatalf("baseBehind=%d, want 0", res.Dashboard.BaseBehind)
	}
}

func mustRun(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runOut(t *testing.T, dir string, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
