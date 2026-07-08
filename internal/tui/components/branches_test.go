package components_test

import (
	"strings"
	"testing"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

func TestRenderBranchesPanelShowsPosition(t *testing.T) {
	body := components.RenderBranchListLineNumbered(1, gitpkg.BranchInfo{Name: "feature/x"}, true)
	out := components.RenderBranchesPanel(1, 5, "main", body, 70)
	if !strings.Contains(out, "2/5") {
		t.Fatalf("missing position in title: %q", out)
	}
	if !strings.Contains(out, "feature/x") {
		t.Fatalf("missing branch name: %q", out)
	}
}

func TestRenderBranchDetailContextTitle(t *testing.T) {
	out := components.RenderBranchDetail(nil, "feature/x", "main", 60, 2)
	if !strings.Contains(out, "Context · feature/x") {
		t.Fatalf("missing context title: %q", out)
	}
}
