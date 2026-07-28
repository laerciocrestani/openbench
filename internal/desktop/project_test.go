package desktop

import (
	"os"
	"path/filepath"
	"testing"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

func TestTryOpenProject_needsInit(t *testing.T) {
	dir := t.TempDir()
	res, err := TryOpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.NeedsGitInit || res.Dashboard != nil {
		t.Fatalf("expected needsGitInit without dashboard, got %+v", res)
	}
}

func TestInitGitAndOpen(t *testing.T) {
	dir := t.TempDir()
	dash, err := InitGitAndOpen(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if dash == nil || dash.Path == "" {
		t.Fatalf("incomplete dashboard: %+v", dash)
	}
	repo, err := gitpkg.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.IsRepo(); err != nil {
		t.Fatal(err)
	}
}

func TestInitGitAndOpen_addAll(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitGitAndOpen(dir, true); err != nil {
		t.Fatal(err)
	}
	repo, err := gitpkg.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := repo.HasStagedChanges()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected staged files after git add .")
	}
}

func TestStageFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitGitAndOpen(dir, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := StageFiles(dir, []string{"a.txt"}); err != nil {
		t.Fatal(err)
	}
	repo, err := gitpkg.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, _, unt, err := repo.WorktreeCounts()
	if err != nil {
		t.Fatal(err)
	}
	if st < 1 {
		t.Fatalf("expected at least 1 staged, got staged=%d untracked=%d", st, unt)
	}
	if unt < 1 {
		t.Fatalf("expected b.txt still untracked, got untracked=%d", unt)
	}
}

func TestCreateProject(t *testing.T) {
	parent := t.TempDir()
	dash, err := CreateProject(parent, "meu-app")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(parent, "meu-app")
	if filepath.Base(dash.Path) != "meu-app" {
		t.Fatalf("repo name: got %q", dash.Path)
	}
	if _, err := os.Stat(filepath.Join(want, ".git")); err != nil {
		t.Fatalf("expected .git: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dash.Path, ".git")); err != nil {
		t.Fatalf("dashboard path missing .git: %v", err)
	}
}

func TestCreateProject_invalidName(t *testing.T) {
	parent := t.TempDir()
	if _, err := CreateProject(parent, "../evil"); err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestValidateProjectName(t *testing.T) {
	if err := validateProjectName(""); err == nil {
		t.Fatal("empty")
	}
	if err := validateProjectName("ok-name"); err != nil {
		t.Fatal(err)
	}
}
