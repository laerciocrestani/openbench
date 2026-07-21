package desktop

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveUnderProject(t *testing.T) {
	root := "/tmp/openbench-proj"
	if runtime.GOOS == "windows" {
		root = `C:\tmp\openbench-proj`
	}
	root, _ = filepath.Abs(root)

	ok, err := resolveUnderProject(root, ".gitignore")
	if err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	want := filepath.Join(root, ".gitignore")
	if ok != want {
		t.Fatalf("got %q want %q", ok, want)
	}

	ok, err = resolveUnderProject(root, "src/../README.md")
	if err != nil {
		t.Fatalf("expected ok after clean: %v", err)
	}
	if ok != filepath.Join(root, "README.md") {
		t.Fatalf("got %q", ok)
	}

	_, err = resolveUnderProject(root, "../outside")
	if err == nil {
		t.Fatal("expected escape to fail")
	}

	_, err = resolveUnderProject(root, "")
	if err == nil {
		t.Fatal("expected empty path to fail")
	}
}
