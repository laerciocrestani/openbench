package ui

import (
	"sync"
	"testing"
)

func TestVersionFromBuild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	t.Chdir(dir)

	buildVersion = "v0.1.5"
	buildCommit = "3e691df"
	t.Cleanup(func() {
		buildVersion = ""
		buildCommit = ""
		resetRuntimeForTest()
	})
	resetRuntimeForTest()

	if got := Version(); got != "v0.1.5 · 3e691df" {
		t.Errorf("Version() = %q", got)
	}
}

func TestVersionExactBuild(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	t.Chdir(dir)

	buildVersion = "v0.1.0"
	buildCommit = ""
	t.Cleanup(func() {
		buildVersion = ""
		resetRuntimeForTest()
	})
	resetRuntimeForTest()

	if got := Version(); got != "v0.1.0" {
		t.Errorf("Version() = %q", got)
	}
}

func resetRuntimeForTest() {
	runtimeOnce = sync.Once{}
	runtimeOK = false
}
