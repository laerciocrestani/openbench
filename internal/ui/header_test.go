package ui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestFormatDashboardHeaderContainsTitleAndVersion(t *testing.T) {
	out := FormatDashboardHeader(nil, 78, false, false)

	if !strings.Contains(out, "GITAI") {
		t.Fatalf("header missing title: %q", out)
	}
	if !strings.Contains(out, "AI Git Workflow") {
		t.Fatalf("header missing tagline: %q", out)
	}
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("header missing box borders: %q", out)
	}
}

func TestFormatDashboardHeaderContext(t *testing.T) {
	ctx := &HeaderContext{
		Repo:     "gitai",
		Branch:   "main",
		HeadHash: "22843f3",
		Status:   "✓ Clean",
		Sync:     "✓ in sync",
		Provider: "gemini",
		Model:    "gemini-2.5-flash-lite",
		AIReady:  true,
		OnBase:   true,
	}
	out := FormatDashboardHeader(ctx, 100, false, false)

	for _, want := range []string{"gitai", "main", "22843f3", "gemini", "gemini-2.5-flash-lite", "Ready", "⧉"} {
		if !strings.Contains(out, want) {
			t.Fatalf("header missing %q: %q", want, out)
		}
	}
}

func TestFormatDashboardHeaderDryRun(t *testing.T) {
	out := FormatDashboardHeader(nil, 100, true, false)
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("header missing dry-run mode: %q", out)
	}
}

func TestFormatDashboardHeaderWidth(t *testing.T) {
	out := FormatDashboardHeader(nil, 60, false, false)
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		if runewidth.StringWidth(line) != 60 {
			t.Fatalf("line width = %d, want 60: %q", runewidth.StringWidth(line), line)
		}
	}
}

func TestFormatBannerAlias(t *testing.T) {
	ctx := HeaderContext{Repo: "gitai", Branch: "main", Status: "clean"}
	out := FormatBanner(false, &ctx, false)
	if !strings.Contains(out, "gitai") {
		t.Fatalf("FormatBanner alias broken: %q", out)
	}
}

func TestTruncateRunewidth(t *testing.T) {
	got := truncateRunewidth("hello world", 8)
	if runewidth.StringWidth(got) > 8 {
		t.Fatalf("unexpected truncation width: %q", got)
	}
}
