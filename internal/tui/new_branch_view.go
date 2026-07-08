package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

type branchCreateMsg struct {
	name string
	err  error
}

func (m *branchesModel) startNewBranch() tea.Cmd {
	m.mode = branchesModeNew
	m.newStep = components.NewBranchStepFrom
	m.fromCursor = m.cursor
	m.templateCursor = 0
	m.templateItems = components.BranchTemplateItems()
	m.fromBranch = ""
	m.selectedTemplate = components.NewBranchTemplate{}
	m.newErr = nil
	m.nameInput = textinput.New()
	m.nameInput.Placeholder = "nome-da-branch"
	m.nameReady = false
	m.refreshNewBranchContent()
	return textinput.Blink
}

func (m *branchesModel) cancelNewBranch() {
	m.mode = branchesModeList
	m.newErr = nil
	m.nameInput.Blur()
}

func (m *branchesModel) newBranchBack() tea.Cmd {
	switch m.newStep {
	case components.NewBranchStepFrom:
		m.cancelNewBranch()
		return nil
	case components.NewBranchStepTemplate:
		m.newStep = components.NewBranchStepFrom
		m.newErr = nil
		return nil
	case components.NewBranchStepName:
		m.newStep = components.NewBranchStepTemplate
		m.nameInput.Blur()
		m.newErr = nil
		return nil
	}
	return nil
}

func (m *branchesModel) newBranchAdvance() tea.Cmd {
	switch m.newStep {
	case components.NewBranchStepFrom:
		m.fromBranch = m.selectedFromBranch()
		if m.fromBranch == "" {
			m.newErr = errEmptyBranchName
			return nil
		}
		m.newStep = components.NewBranchStepTemplate
		m.templateCursor = 0
		m.newErr = nil
		m.refreshTemplateContent()
		return nil

	case components.NewBranchStepTemplate:
		m.selectedTemplate = components.TemplateAtCursor(m.templateItems, m.templateCursor)
		m.newStep = components.NewBranchStepName
		m.nameInput.SetValue(m.selectedTemplate.NameSeed())
		m.nameInput.Width = maxInt(m.listViewport.Width-6, 20)
		m.nameInput.Focus()
		m.nameReady = true
		m.newErr = nil
		return textinput.Blink

	case components.NewBranchStepName:
		name := strings.TrimSpace(m.nameInput.Value())
		if !validBranchName(name) {
			m.newErr = errInvalidBranchName
			return nil
		}
		m.newErr = nil
		return createBranchCmd(name, m.fromBranch)
	}
	return nil
}

func (m *branchesModel) selectedFromBranch() string {
	if m.fromCursor < 0 || m.fromCursor >= len(m.branches) {
		return ""
	}
	return m.branches[m.fromCursor].Name
}

func (m *branchesModel) moveFromCursor(delta int) {
	if len(m.branches) == 0 {
		return
	}
	m.fromCursor += delta
	if m.fromCursor < 0 {
		m.fromCursor = 0
	}
	if m.fromCursor >= len(m.branches) {
		m.fromCursor = len(m.branches) - 1
	}
	m.refreshFromContent()
}

func (m *branchesModel) moveTemplateCursor(delta int) {
	selectable := components.SelectableTemplateCount(m.templateItems)
	if selectable == 0 {
		return
	}
	m.templateCursor += delta
	if m.templateCursor < 0 {
		m.templateCursor = 0
	}
	if m.templateCursor >= selectable {
		m.templateCursor = selectable - 1
	}
	m.refreshTemplateContent()
}

func (m *branchesModel) refreshNewBranchContent() {
	switch m.newStep {
	case components.NewBranchStepFrom:
		m.refreshFromContent()
	case components.NewBranchStepTemplate:
		m.refreshTemplateContent()
	}
}

func (m *branchesModel) refreshFromContent() {
	if !m.ready {
		return
	}
	lines := make([]string, len(m.branches))
	for i, b := range m.branches {
		lines[i] = components.RenderBranchListLineNumbered(i, b, i == m.fromCursor)
	}
	m.fromViewport.SetContent(strings.Join(lines, "\n"))
	m.syncFromScroll()
}

func (m *branchesModel) syncFromScroll() {
	if !m.ready || len(m.branches) == 0 {
		return
	}
	if m.fromCursor >= m.fromViewport.YOffset+m.fromViewport.Height {
		m.fromViewport.SetYOffset(m.fromCursor - m.fromViewport.Height + 1)
	} else if m.fromCursor < m.fromViewport.YOffset {
		m.fromViewport.SetYOffset(m.fromCursor)
	}
}

func (m *branchesModel) refreshTemplateContent() {
	if !m.ready {
		return
	}
	inner := m.templateViewport.Width - 4
	if inner < 40 {
		inner = 40
	}
	selected := components.TemplateAtCursor(m.templateItems, m.templateCursor)
	body := components.RenderNewBranchTemplateBody(m.templateCursor, m.templateItems, selected, inner)
	m.templateViewport.SetContent(body)
	line := m.templateLineOffset(m.templateCursor)
	if line >= m.templateViewport.YOffset+m.templateViewport.Height {
		m.templateViewport.SetYOffset(line - m.templateViewport.Height + 1)
	} else if line < m.templateViewport.YOffset {
		m.templateViewport.SetYOffset(line)
	}
}

func (m *branchesModel) templateLineOffset(cursor int) int {
	idx := 0
	line := 0
	for _, item := range m.templateItems {
		if item.Separator {
			line++
			continue
		}
		if idx == cursor {
			return line
		}
		idx++
		line++
	}
	return line
}

func (m *branchesModel) fromBody() string {
	if len(m.branches) == 0 {
		return ""
	}
	if m.ready {
		return m.fromViewport.View()
	}
	lines := make([]string, len(m.branches))
	for i, b := range m.branches {
		lines[i] = components.RenderBranchListLineNumbered(i, b, i == m.fromCursor)
	}
	return strings.Join(lines, "\n")
}

func (m *branchesModel) templateBody() string {
	if m.ready {
		return m.templateViewport.View()
	}
	inner := 76
	if m.listViewport.Width > 4 {
		inner = m.listViewport.Width - 4
	}
	selected := components.TemplateAtCursor(m.templateItems, m.templateCursor)
	return components.RenderNewBranchTemplateBody(m.templateCursor, m.templateItems, selected, inner)
}

func (m *branchesModel) updateNewBranch(msg tea.Msg) (tea.Cmd, bool) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return m.newBranchBack(), true
		case "enter":
			return m.newBranchAdvance(), true
		case "up", "k":
			switch m.newStep {
			case components.NewBranchStepFrom:
				m.moveFromCursor(-1)
			case components.NewBranchStepTemplate:
				m.moveTemplateCursor(-1)
			}
			return nil, true
		case "down", "j":
			switch m.newStep {
			case components.NewBranchStepFrom:
				m.moveFromCursor(1)
			case components.NewBranchStepTemplate:
				m.moveTemplateCursor(1)
			}
			return nil, true
		}
	}

	if m.newStep == components.NewBranchStepName && m.nameReady {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		return cmd, true
	}

	if m.newStep == components.NewBranchStepFrom {
		var cmd tea.Cmd
		m.fromViewport, cmd = m.fromViewport.Update(msg)
		return cmd, true
	}
	if m.newStep == components.NewBranchStepTemplate {
		var cmd tea.Cmd
		m.templateViewport, cmd = m.templateViewport.Update(msg)
		return cmd, true
	}
	return nil, false
}

func (m branchesModel) viewNewBranch(width int) string {
	var b strings.Builder
	if m.newErr != nil {
		b.WriteString(styleError.Render("  ✖ " + m.newErr.Error()))
		b.WriteString("\n\n")
	}

	switch m.newStep {
	case components.NewBranchStepFrom:
		b.WriteString(components.RenderNewBranchFromPanel(m.fromCursor, len(m.branches), m.fromBody(), width))
	case components.NewBranchStepTemplate:
		selectable := components.SelectableTemplateCount(m.templateItems)
		b.WriteString(components.RenderNewBranchTemplatePanel(m.templateCursor, selectable, m.templateBody(), width))
	case components.NewBranchStepName:
		b.WriteString(components.RenderNewBranchNamePanel(m.fromBranch, m.selectedTemplate, m.nameInput.View(), width))
	}
	return b.String()
}

func createBranchCmd(name, from string) tea.Cmd {
	return func() tea.Msg {
		err := app.CreateBranch(name, from)
		return branchCreateMsg{name: name, err: err}
	}
}

var (
	errEmptyBranchName   = teaErr("select a source branch")
	errInvalidBranchName = teaErr("invalid branch name")
)

type teaErr string

func (e teaErr) Error() string { return string(e) }

func validBranchName(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "/") || strings.Contains(name, "..") {
		return false
	}
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}
	for _, r := range name {
		if unicode.IsSpace(r) {
			return false
		}
		switch r {
		case '~', '^', ':', '?', '*', '[', '\\', '@', '{', '}':
			return false
		}
	}
	return true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
