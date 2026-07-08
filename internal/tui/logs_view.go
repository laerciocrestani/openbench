package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type logsModel struct {
	content  string
	viewport viewport.Model
	ready    bool
	err      error
}

func newLogsModel() logsModel {
	return logsModel{}
}

func (m *logsModel) SetSize(width, height int) {
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

func (m *logsModel) Load(content string, err error) {
	m.content = content
	m.err = err
	m.ready = false
}

func (m logsModel) Update(msg tea.Msg) (logsModel, tea.Cmd) {
	if m.err != nil {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m logsModel) View(width int) string {
	if m.err != nil {
		return styleError.Render("  ✗ " + m.err.Error())
	}

	var b strings.Builder
	b.WriteString(styleSection.Render("Logs"))
	b.WriteString("\n\n")
	if !m.ready {
		b.WriteString(styleHint.Render("  (empty)"))
		return b.String()
	}
	b.WriteString(m.viewport.View())
	return b.String()
}

func logsHelpLine() string {
	return styleKey.Render("↑↓") + " scroll  " +
		styleKey.Render("esc") + " back"
}
