package components_test

import (
	"strings"
	"testing"

	"github.com/laerciocrestani/gitai/internal/tui/components"
)

func TestAlertLogsFiltersWarnings(t *testing.T) {
	logs := []string{
		"✓ Staging changes",
		"✖ Modelo sobrecarregado — tentando novamente em 3s (1/3)...",
		"✖ Modelo sobrecarregado — tentando novamente em 3s (2/3)...",
	}
	alerts := components.AlertLogs(logs)
	if len(alerts) != 2 {
		t.Fatalf("alerts = %v", alerts)
	}
}

func TestSpinnerFrameAnimates(t *testing.T) {
	t.Parallel()
	a := components.SpinnerFrame(0)
	b := components.SpinnerFrame(1)
	if a == b {
		t.Fatalf("spinner should change between ticks: %q %q", a, b)
	}
}

func TestRenderLoadingShowsSpinner(t *testing.T) {
	alerts := []string{
		"✖ Modelo sobrecarregado — tentando novamente em 3s (1/3)...",
	}
	out := components.RenderSpinnerLine("Thinking", 3)
	if !strings.Contains(out, "Thinking") {
		t.Fatalf("missing status: %q", out)
	}
	if !strings.Contains(out, components.SpinnerFrame(3)) {
		t.Fatalf("missing spinner frame: %q", out)
	}

	out = components.RenderLoading("Pulling main", alerts, 5, 100)
	if !strings.Contains(out, "Pulling main") {
		t.Fatalf("missing status: %q", out)
	}
	if strings.Contains(out, "%") {
		t.Fatalf("should not show percent: %q", out)
	}
	if strings.Contains(out, "█") {
		t.Fatalf("should not show progress bar: %q", out)
	}
	for _, want := range []string{components.SpinnerFrame(5), "Working"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderActionDone(t *testing.T) {
	logs := []string{
		"✓ Fetching origin",
		"✓ Pulling main",
		"  * branch main -> FETCH_HEAD",
		"  Already up to date.",
		"✓ Synced with origin/main",
	}
	out := components.RenderActionDone("Sync", "Synced with origin/main", logs, 80)
	for _, want := range []string{"Concluído", "Synced with origin/main", "Fetching origin", "Pulling main"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "FETCH_HEAD") {
		t.Fatalf("git noise should be filtered: %q", out)
	}
}
