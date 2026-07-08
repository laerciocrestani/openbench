package ui_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/laerciocrestani/gitai/internal/ui"
)

func TestRenderBoxNoRightBorder(t *testing.T) {
	width := 80
	out := ui.RenderBox("Git Graph", "line", width, ui.PlainBoxStyle())
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if strings.Contains(line, "╮") || strings.Contains(line, "╯") || strings.HasSuffix(strings.TrimSpace(line), "│") {
			t.Fatalf("unexpected right border in: %q", line)
		}
		if runewidth.StringWidth(line) != width {
			t.Fatalf("line width = %d, want %d: %q", runewidth.StringWidth(line), width, line)
		}
	}
}

func TestPadLineRightAlign(t *testing.T) {
	inner := ui.ContentInner(80)
	left := "Repository: gitai"
	right := "✓ Clean"
	line := ui.PadLine(left, right, inner)
	if !strings.HasSuffix(strings.TrimSpace(line), right) {
		t.Fatalf("right text not aligned: %q", line)
	}
}

func TestPadLineShaded(t *testing.T) {
	shade := ui.RightShadeStyle(true)
	line := ui.PadLineShaded("left", "right", 40, 10, shade)
	if !strings.HasSuffix(strings.TrimSpace(line), "right") {
		t.Fatalf("right text not aligned: %q", line)
	}
	if shade != nil && !strings.Contains(line, "\x1b") {
		t.Skip("terminal sem suporte ANSI neste ambiente")
	}
}

func TestFormatDashboardHeaderMatchesBoxFormat(t *testing.T) {
	ctx := &ui.HeaderContext{
		Repo:     "gitai",
		Branch:   "main",
		HeadHash: "abc1234",
		Status:   "✓ Clean",
		Sync:     "↑ 1 ahead",
		Provider: "gemini",
		Model:    "gemini-2.5-flash-lite",
		AIReady:  true,
		OnBase:   true,
	}
	out := ui.FormatDashboardHeader(ctx, 100, false, false)
	plain := ansi.Strip(out)
	if !strings.HasPrefix(out, "╭ GITAI") {
		t.Fatalf("header should start with box top: %q", out)
	}
	for _, want := range []string{"AI Git Workflow", "gitai", "✓ Clean", "↑ 1 ahead", "Ready", "abc1234"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("header missing %q", want)
		}
	}
}
