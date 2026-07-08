package views_test

import (
	"strings"
	"testing"

	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/config"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/views"
)

func TestRenderDashboardPanels(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{
			Branch:     "feature/ui",
			BaseBranch: "main",
			HeadHash:   "abc1234",
			Modified:   1,
			FileChanges: []gitpkg.FileChange{
				{Path: "internal/tui/app.go", Status: "modified", Insertions: 5, Deletions: 1},
			},
			RecentCommits: []string{"abc1234 feat: ui"},
			CommitsAheadOfBase: 2,
		},
		Config:    &config.Config{APIKey: "k", Provider: config.ProviderGemini, Model: "gemini-2.5-flash-lite"},
		NextSteps: []app.NextStep{{Command: "gitai commit"}},
		HasGH:     true,
	}

	out := views.RenderDashboard(snap, views.DashboardOptions{Width: 80, Height: 40})
	for _, want := range []string{"Git Graph", "Repository Summary", "Changed Files", "AI Engine", "Recent Commits", "Suggested Action"} {
		if !strings.Contains(out, want) {
			t.Fatalf("dashboard missing %q:\n%s", want, out)
		}
	}
}

func TestRenderLoadingDashboard(t *testing.T) {
	out := views.RenderLoadingDashboard("Carregando…", 45, 60)
	if !strings.Contains(out, "Carregando") {
		t.Fatalf("loading missing message: %q", out)
	}
	if !strings.Contains(out, "45%") {
		t.Fatalf("loading missing percent: %q", out)
	}
	if !strings.Contains(out, "█") {
		t.Fatalf("loading missing progress bar: %q", out)
	}
}
