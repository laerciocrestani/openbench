package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type diffModel struct {
	title    string
	content  string
	viewport viewport.Model
	ready    bool
	err      error
}

func newDiffModel() diffModel {
	return diffModel{}
}

func (m diffModel) Init() tea.Cmd {
	return nil
}

func (m *diffModel) SetSize(width, height int) {
	headerRows := 4
	footerRows := 2
	vh := height - headerRows - footerRows
	if vh < 3 {
		vh = 3
	}
	if !m.ready {
		m.viewport = viewport.New(width, vh)
		m.viewport.SetContent(m.content)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = vh
	}
}

func (m *diffModel) Load(title, content string, err error) {
	m.title = title
	m.content = content
	m.err = err
	m.ready = false
}

func (m diffModel) Update(msg tea.Msg) (diffModel, tea.Cmd) {
	if m.err != nil {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m diffModel) View(width int) string {
	if m.err != nil {
		return styleError.Render("  ✗ " + m.err.Error())
	}

	var b strings.Builder
	b.WriteString(styleSection.Render("Diff"))
	b.WriteString("\n")
	if m.title != "" {
		b.WriteString(styleHint.Render("  " + m.title))
		b.WriteString("\n\n")
	}
	if !m.ready {
		b.WriteString(styleHint.Render("  (empty)"))
		return b.String()
	}
	b.WriteString(m.viewport.View())
	return b.String()
}

func diffHelpLine() string {
	return styleKey.Render("↑↓") + " scroll  " +
		styleKey.Render("esc") + " back"
}
