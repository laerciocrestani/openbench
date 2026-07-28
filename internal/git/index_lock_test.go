package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClearStaleIndexLock(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repo := &Repo{dir: dir}

	if err := repo.ClearStaleIndexLock(); err != nil {
		t.Fatalf("no lock: %v", err)
	}

	lock := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// Make it look stale.
	past := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(lock, past, past); err != nil {
		t.Fatal(err)
	}
	if err := repo.ClearStaleIndexLock(); err != nil {
		t.Fatalf("stale lock: %v", err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatal("expected lock removed")
	}

	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo.ClearStaleIndexLock(); err == nil {
		t.Fatal("expected error for fresh lock")
	}
}
