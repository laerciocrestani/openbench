package components_test

import (
	"strings"
	"testing"

	"github.com/laerciocrestani/gitai/internal/tui/components"
)

func TestSyncModeCatalog(t *testing.T) {
	t.Parallel()
	modes := components.SyncModeCatalog()
	if len(modes) != 3 {
		t.Fatalf("expected 3 modes, got %d", len(modes))
	}
	if modes[0].Mode != components.SyncModeStandard {
		t.Fatalf("first mode should be standard")
	}
	if modes[1].Flag != "--prune-remote" {
		t.Fatalf("second flag = %q", modes[1].Flag)
	}
	if modes[2].Flag != "--prune" {
		t.Fatalf("third flag = %q", modes[2].Flag)
	}
}

func TestSyncModeToAppOptions(t *testing.T) {
	t.Parallel()
	full := components.SyncModeCatalog()[2]
	prune, pruneRemote, base := full.ToAppOptions("main")
	if !prune || !pruneRemote || base != "main" {
		t.Fatalf("prune=%v pruneRemote=%v base=%q", prune, pruneRemote, base)
	}

	remote := components.SyncModeCatalog()[1]
	prune, pruneRemote, _ = remote.ToAppOptions("develop")
	if prune || !pruneRemote {
		t.Fatalf("prune-remote: prune=%v pruneRemote=%v", prune, pruneRemote)
	}
}

func TestRenderSyncOptionsPanel(t *testing.T) {
	modes := components.SyncModeCatalog()
	out := components.RenderSyncOptionsPanel(0, modes, "main", false, 90)
	for _, want := range []string{"Sync · Options", "Standard sync", "Base: main", "git fetch origin --prune", "--prune-remote", "--prune"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
