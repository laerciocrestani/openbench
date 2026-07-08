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

func TestRenderLoadingStacksAlerts(t *testing.T) {
	alerts := []string{
		"✖ Modelo sobrecarregado — tentando novamente em 3s (1/3)...",
		"✖ Modelo sobrecarregado — tentando novamente em 3s (2/3)...",
	}
	out := components.RenderLoading("Thinking…", alerts, 52, 100)
	if !strings.Contains(out, "Thinking") {
		t.Fatalf("missing status: %q", out)
	}
	if !strings.Contains(out, "52%") {
		t.Fatalf("missing percent: %q", out)
	}
	for _, want := range []string{"Modelo sobrecarregado", "(1/3)", "(2/3)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	idx1 := strings.Index(out, "(1/3)")
	idx2 := strings.Index(out, "(2/3)")
	if idx1 < 0 || idx2 < 0 || idx2 <= idx1 {
		t.Fatalf("alerts not stacked in order: %d %d", idx1, idx2)
	}
}
