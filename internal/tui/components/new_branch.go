package components

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/tui/theme"
	"github.com/laerciocrestani/gitai/internal/ui"
)

// NewBranchStep identifies the wizard step for creating a branch.
type NewBranchStep int

const (
	NewBranchStepFrom NewBranchStep = iota
	NewBranchStepTemplate
	NewBranchStepName
)

// RenderNewBranchFromPanel renders the source branch picker.
func RenderNewBranchFromPanel(cursor, total int, body string, width int) string {
	title := "New Branch · From"
	if total > 0 {
		title += fmt.Sprintf("  %d/%d", cursor+1, total)
	}
	if strings.TrimSpace(body) == "" {
		body = theme.S.Hint.Render("  (no local branches)")
	}
	return RenderPanel(title, body, width)
}

// RenderNewBranchTemplateListBody renders scrollable picker rows with icon + example labels.
func RenderNewBranchTemplateListBody(cursor int, items []NewBranchTemplateItem) string {
	var lines []string
	selectableIdx := 0
	for _, item := range items {
		if item.Separator {
			lines = append(lines, theme.S.Hint.Render("  ─── more ───"))
			continue
		}
		line := "  " + item.Template.ListLabel()
		if selectableIdx == cursor {
			line = theme.S.Current.Render("> " + strings.TrimPrefix(line, "  "))
		} else {
			line = theme.S.Hint.Render(line)
		}
		lines = append(lines, line)
		selectableIdx++
	}
	return strings.Join(lines, "\n")
}

// RenderNewBranchTemplateBody renders picker list plus the full reference table.
func RenderNewBranchTemplateBody(cursor int, items []NewBranchTemplateItem, selected NewBranchTemplate, inner int) string {
	var lines []string
	lines = append(lines, RenderNewBranchTemplateListBody(cursor, items))
	lines = append(lines, "")
	lines = append(lines, renderFullTemplateTable(items, selected, inner)...)
	return strings.Join(lines, "\n")
}

// RenderNewBranchTemplatePanel wraps the template step content.
func RenderNewBranchTemplatePanel(cursor, selectable int, body string, width int) string {
	title := "New Branch · Template"
	if selectable > 0 {
		title += fmt.Sprintf("  %d/%d", cursor+1, selectable)
	}
	return RenderPanel(title, body, width)
}

func renderFullTemplateTable(items []NewBranchTemplateItem, selected NewBranchTemplate, inner int) []string {
	_ = inner
	header := theme.S.Hint.Render("  Example · Usage · Example")
	lines := []string{header}

	for _, item := range items {
		if item.Separator {
			continue
		}
		t := item.Template
		row := truncatePlain(t.DetailLabel(), inner-2)
		if templatesMatch(t, selected) {
			lines = append(lines, theme.S.Current.Render("> "+row))
		} else {
			lines = append(lines, theme.S.Hint.Render("  "+row))
		}
	}
	return lines
}

// RenderNewBranchNamePanel renders the branch name input step.
func RenderNewBranchNamePanel(from string, template NewBranchTemplate, nameField string, width int) string {
	var lines []string

	lines = append(lines, theme.S.Hint.Render("  From: "+from))
	if template.Other {
		lines = append(lines, theme.S.Hint.Render("  Template: ✏️ Other (free-form name)"))
	} else if template.Prefix != "" {
		lines = append(lines, theme.S.Hint.Render("  Template: "+template.ListLabel()))
	}
	lines = append(lines, "")
	lines = append(lines, theme.S.Hint.Render("  Branch name:"))
	lines = append(lines, "  "+nameField)

	preview := strings.TrimSpace(stripANSI(nameField))
	if preview == "" {
		preview = "(type a name)"
	}
	lines = append(lines, "")
	lines = append(lines, theme.S.Hint.Render("  Preview: "+preview))

	body := strings.Join(lines, "\n")
	return RenderPanel("New Branch · Name", body, width)
}

func truncatePlain(s string, max int) string {
	if ui.DisplayWidth(s) <= max {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && ui.DisplayWidth(string(runes)) > max-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		if r == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
