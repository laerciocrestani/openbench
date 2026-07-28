package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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

func TestLoadTimelineCommitsOnDay(t *testing.T) {
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
	runEnv := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(append(os.Environ(), "GIT_TERMINAL_PROMPT=0"), env...)
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
	run("commit", "-m", "today-commit")

	repo, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Now().In(time.Local).Format("2006-01-02")
	commits, err := repo.LoadTimelineCommitsOnDay(day)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) < 1 {
		t.Fatal("expected today's commit")
	}
	empty, err := repo.LoadTimelineCommitsOnDay("2000-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no commits on empty day, got %d", len(empty))
	}

	// Fixed local midnights: yesterday must not bleed into today (and vice-versa).
	// Bare YYYY-MM-DD --since=<today> is "now" in Git and would hide today's commits.
	yesterday := time.Now().In(time.Local).AddDate(0, 0, -1)
	yStamp := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 15, 30, 0, 0, time.Local).
		Format(time.RFC3339)
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	runEnv([]string{
		"GIT_AUTHOR_DATE=" + yStamp,
		"GIT_COMMITTER_DATE=" + yStamp,
	}, "commit", "-m", "yesterday-commit")

	todayOnly, err := repo.LoadTimelineCommitsOnDay(day)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range todayOnly {
		if c.Subject == "yesterday-commit" {
			t.Fatal("yesterday commit leaked into today")
		}
	}
	if len(todayOnly) < 1 {
		t.Fatal("expected today's commit after adding yesterday")
	}

	yDay := yesterday.Format("2006-01-02")
	yCommits, err := repo.LoadTimelineCommitsOnDay(yDay)
	if err != nil {
		t.Fatal(err)
	}
	foundY := false
	for _, c := range yCommits {
		if c.Subject == "today-commit" {
			t.Fatal("today commit leaked into yesterday")
		}
		if c.Subject == "yesterday-commit" {
			foundY = true
		}
	}
	if !foundY {
		t.Fatal("expected yesterday-commit on yesterday")
	}
}

func TestParseDecorations(t *testing.T) {
	got := parseDecorations("HEAD -> main, origin/main, tag: v1.0.0")
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
}
