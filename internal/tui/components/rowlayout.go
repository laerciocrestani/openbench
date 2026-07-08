package components

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/tui/theme"
	"github.com/laerciocrestani/gitai/internal/ui"
	"github.com/laerciocrestani/gitai/internal/uiprefs"
)

const (
	pathPad  = 3
	statsPad = 3
	minDots  = 4
)

func buildStatsBlock(insertions, deletions int) (string, int) {
	plus := theme.S.Success.Render(fmt.Sprintf("+%d", insertions))
	minus := theme.S.Error.Render(fmt.Sprintf("-%d", deletions))
	sep := theme.S.Hint.Render("·")
	right := plus + " " + sep + " " + minus
	return right, ui.DisplayWidth(right)
}

func renderGradientDots(count int, colorsEnabled bool) string {
	if count <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < count; i++ {
		progress := float64(i) / float64(maxInt(count-1, 1))
		b.WriteString(ui.GradientDot(progress, colorsEnabled))
	}
	return b.String()
}

// buildAlignedStatsRow renders left text, dot leaders, and +N · -M stats.
func buildAlignedStatsRow(left, leftStyled string, insertions, deletions, inner int) string {
	if leftStyled == "" {
		leftStyled = left
	}
	right, rightW := buildStatsBlock(insertions, deletions)
	gapBeforeStats := strings.Repeat(" ", statsPad)
	gapAfterLeft := strings.Repeat(" ", pathPad)

	leftW := ui.DisplayWidth(leftStyled) + pathPad
	dots := inner - leftW - statsPad - rightW
	if dots < minDots {
		dots = minDots
	}

	dotsStyled := renderGradientDots(dots, uiprefs.ColorsEnabled())
	row := leftStyled + gapAfterLeft + dotsStyled + gapBeforeStats + right
	return ui.PadDisplayWidth(row, inner)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
