package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
	"github.com/laerciocrestani/gitai/internal/ui"
)

// RenderSummary renders the repository change summary panel.
func RenderSummary(summary app.ChangeSummary, width int) string {
	if summary.FileCount == 0 {
		return ""
	}

	inner := ui.ContentInner(width)
	left := fmt.Sprintf("Files Changed: %d", summary.FileCount)
	line1 := buildAlignedStatsRow(left, theme.S.Hint.Render(left), summary.Insertions, summary.Deletions, inner)

	var langParts []string
	keys := make([]string, 0, len(summary.Languages))
	for k := range summary.Languages {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		langParts = append(langParts, fmt.Sprintf("%s:%d", k, summary.Languages[k]))
	}
	line2 := theme.S.Hint.Render(strings.Join(langParts, "     "))

	body := line1 + "\n" + line2
	return RenderPanel("Repository Summary", body, width)
}
