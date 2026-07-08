package components_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

func TestRenderFooterLowercaseKeys(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{
			Modified:      1,
			HeadHash:      "abc1234",
			RecentCommits: []string{"abc feat"},
		},
	}
	items := components.DefaultFooterItems(snap)
	out := components.RenderFooter(items, 120)
	plain := ansi.Strip(out)
	for _, key := range []string{"[p]", "[d]", "[y]", "[l]"} {
		if !strings.Contains(plain, key) {
			t.Fatalf("footer missing lowercase key %s: %q", key, plain)
		}
	}
	if strings.Contains(plain, "[P] Push") {
		t.Fatalf("push should not display uppercase P: %q", plain)
	}
}

func TestRenderGitGraph(t *testing.T) {
	o := &gitpkg.Overview{
		Branch:             "feature/x",
		BaseBranch:         "main",
		CommitsAheadOfBase: 3,
	}
	out := components.RenderGitGraph(o, 60)
	if !strings.Contains(out, "Git Graph") || !strings.Contains(out, "HEAD") {
		t.Fatalf("git graph incomplete: %q", out)
	}
}

func TestRenderPanelWidthAlignment(t *testing.T) {
	width := 80
	out := components.RenderPanel("Repository Summary", "line one\nline two", width)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	title := lines[0]
	if strings.HasSuffix(title, "╮") {
		t.Fatalf("title should not end with ╮: %q", title)
	}
	bottom := lines[len(lines)-1]
	if strings.HasSuffix(bottom, "╯") {
		t.Fatalf("bottom should not end with ╯: %q", bottom)
	}
	for i, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasSuffix(strings.TrimSpace(line), "│") {
			t.Fatalf("line %d should not end with right border: %q", i, line)
		}
		got := runewidth.StringWidth(line)
		if got != width {
			t.Fatalf("line %d width = %d, want %d: %q", i, got, width, line)
		}
	}
}

func TestRenderSummaryShowsStats(t *testing.T) {
	summary := app.ChangeSummary{
		FileCount:  20,
		Insertions: 119,
		Deletions:  214,
		Languages:  map[string]int{"Go": 5},
	}
	out := components.RenderSummary(summary, 80)
	plain := ansi.Strip(out)
	for _, want := range []string{"+119 · -214", "Files Changed: 20", "...."} {
		if !strings.Contains(plain, want) {
			t.Fatalf("summary missing %q in:\n%s", want, plain)
		}
	}
}


func TestRenderFileTableUsesDotLeaders(t *testing.T) {
	changes := []gitpkg.FileChange{
		{Path: "small.go", Status: "modified", Insertions: 2, Deletions: 1},
		{Path: "big.go", Status: "modified", Insertions: 100, Deletions: 50},
		{Path: "mid.go", Status: "modified", Insertions: 20, Deletions: 10},
	}
	out := components.RenderFileTable(changes, 80, 10)
	bigIdx := strings.Index(out, "big.go")
	midIdx := strings.Index(out, "mid.go")
	smallIdx := strings.Index(out, "small.go")
	if bigIdx == -1 || midIdx == -1 || smallIdx == -1 {
		t.Fatalf("missing files in output: %q", out)
	}
	if !(bigIdx < midIdx && midIdx < smallIdx) {
		t.Fatalf("files not sorted by change count: big@%d mid@%d small@%d", bigIdx, midIdx, smallIdx)
	}
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "TYPE FILE") {
		t.Fatalf("header should be TYPE FILE only: %q", plain)
	}
	if strings.Contains(plain, "TYPE FILE +") {
		t.Fatalf("header should not include stats columns: %q", plain)
	}
	if !strings.Contains(plain, "+100 · -50") {
		t.Fatalf("stats should use middle dot separator: %q", plain)
	}
	if !strings.Contains(plain, "....") {
		t.Fatalf("missing dot leaders: %q", plain)
	}
	if strings.Contains(plain, "+100 …") || strings.Contains(plain, "-50 …") {
		t.Fatalf("stats should not end with ellipsis: %q", plain)
	}
}
