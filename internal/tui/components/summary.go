package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

// RenderSummary renders the repository change summary panel.
func RenderSummary(summary app.ChangeSummary, width int) string {
	if summary.FileCount == 0 {
		return ""
	}

	plus := theme.S.Success.Render(fmt.Sprintf("+%d", summary.Insertions))
	minus := theme.S.Error.Render(fmt.Sprintf("-%d", summary.Deletions))
	stats := plus + "      " + minus
	line1 := PadLine(
		fmt.Sprintf("Files Changed: %d", summary.FileCount),
		stats,
		width-4,
	)

	var langParts []string
	keys := make([]string, 0, len(summary.Languages))
	for k := range summary.Languages {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		langParts = append(langParts, fmt.Sprintf("%s:%d", k, summary.Languages[k]))
	}
	line2 := strings.Join(langParts, "     ")

	body := line1 + "\n" + line2
	return RenderPanel("Repository Summary", body, width)
}
