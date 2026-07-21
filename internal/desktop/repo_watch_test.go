package desktop

import "testing"

func TestShouldSkipWatchDir(t *testing.T) {
	if !shouldSkipWatchDir("node_modules") {
		t.Fatal("expected skip node_modules")
	}
	if !shouldSkipWatchDir(".git") {
		t.Fatal("expected skip .git")
	}
	if shouldSkipWatchDir("src") {
		t.Fatal("src should be watched")
	}
}
