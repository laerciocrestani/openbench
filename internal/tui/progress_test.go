package tui_test

import (
	"testing"

	"github.com/laerciocrestani/gitai/internal/tui"
)

func TestActionProgressWarn(t *testing.T) {
	p := tui.NewActionProgress()
	p.Warn("Modelo sobrecarregado — tentando novamente em 3s (1/3)...")
	_, logs := p.Snapshot()
	if len(logs) != 1 || logs[0] != "✖ Modelo sobrecarregado — tentando novamente em 3s (1/3)..." {
		t.Fatalf("logs = %v", logs)
	}
}

func TestActionProgressAdvancesByStep(t *testing.T) {
	p := tui.NewActionProgress()
	p.Reset()

	_ = p.Step("Staging changes", func() error { return nil })
	if got := p.Percent(); got < 5 {
		t.Fatalf("after staging percent = %d, want > 0", got)
	}

	_ = p.Step("Reading git diff", func() error { return nil })
	if got := p.Percent(); got < 20 {
		t.Fatalf("after diff percent = %d, want >= 20", got)
	}

	p.Success("Done")
	if got := p.Percent(); got != 100 {
		t.Fatalf("after success percent = %d, want 100", got)
	}
}
