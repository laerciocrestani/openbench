package components_test

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

func TestRenderFooter(t *testing.T) {
	snap := &app.WorkspaceSnapshot{
		Overview: &gitpkg.Overview{Modified: 1},
	}
	items := components.DefaultFooterItems(snap)
	out := components.RenderFooter(items, 80)
	if !strings.Contains(out, "Commit") || !strings.Contains(out, "Quit") {
		t.Fatalf("footer missing shortcuts: %q", out)
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
	for i, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		got := runewidth.StringWidth(line)
		if got != width {
			t.Fatalf("line %d width = %d, want %d: %q", i, got, width, line)
		}
	}
}

func TestRenderFileTableSortsByChanges(t *testing.T) {
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
}
