package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidRepoRoot(t *testing.T) {
	root, err := FindRepoRoot()
	if err != nil {
		t.Skip("not in gitia repo")
	}
	if !isValidRepoRoot(root) {
		t.Fatalf("expected valid root: %s", root)
	}
	if isValidRepoRoot("/tmp") {
		t.Fatal("expected /tmp invalid")
	}
}

func TestSaveAndReadSourceRoot(t *testing.T) {
	root, err := FindRepoRoot()
	if err != nil {
		t.Skip("not in gitia repo")
	}

	path, err := sourceRootFile()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	if err := saveSourceRoot(root); err != nil {
		t.Skip(err)
	}
	if got := readSavedSourceRoot(); got != filepath.Clean(root) {
		t.Fatalf("got %q want %q", got, root)
	}
}
