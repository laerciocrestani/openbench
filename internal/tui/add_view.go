package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

type addModel struct {
	files    []gitpkg.FileChange
	selected map[int]bool
	cursor   int
	viewport viewport.Model
	err      error
	ready    bool
}

type stageFilesMsg struct {
	count int
	all   bool
	err   error
}

func newAddModel() addModel {
	return addModel{selected: map[int]bool{}}
}

func (m *addModel) Load(snap *app.WorkspaceSnapshot) {
	m.files = app.AddableFiles(snap)
	m.selected = map[int]bool{}
	m.cursor = 0
	m.err = nil
	if m.cursor >= len(m.files) {
		m.cursor = 0
	}
	if m.ready {
		m.viewport.SetContent(m.listContent())
	}
}

func (m *addModel) SetSize(width, height int) {
	rows := height - 8
	if rows < 6 {
		rows = 6
	}
	if len(m.files) > 0 && rows > len(m.files)+1 {
		rows = len(m.files) + 1
	}
	if !m.ready {
		m.viewport = viewport.New(width, rows)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = rows
	}
	m.viewport.SetContent(m.listContent())
}

func (m *addModel) listContent() string {
	if len(m.files) == 0 {
		return styleHint.Render("  (nenhum arquivo para adicionar)")
	}
	lines := make([]string, len(m.files))
	for i, f := range m.files {
		lines[i] = components.RenderAddFileLine(m.selected[i], i == m.cursor, f)
	}
	return strings.Join(lines, "\n")
}

func (m *addModel) moveCursor(delta int) tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.files) {
		m.cursor = len(m.files) - 1
	}
	m.viewport.SetContent(m.listContent())
	if m.cursor >= m.viewport.Height+m.viewport.YOffset {
		m.viewport.SetYOffset(m.cursor - m.viewport.Height + 1)
	} else if m.cursor < m.viewport.YOffset {
		m.viewport.SetYOffset(m.cursor)
	}
	return nil
}

func (m *addModel) toggleCursor() {
	if m.cursor < 0 || m.cursor >= len(m.files) {
		return
	}
	m.selected[m.cursor] = !m.selected[m.cursor]
	m.viewport.SetContent(m.listContent())
}

func (m *addModel) toggleAll() {
	if len(m.files) == 0 {
		return
	}
	all := m.allSelected()
	for i := range m.files {
		m.selected[i] = !all
	}
	m.viewport.SetContent(m.listContent())
}

func (m *addModel) allSelected() bool {
	if len(m.files) == 0 {
		return false
	}
	for i := range m.files {
		if !m.selected[i] {
			return false
		}
	}
	return true
}

func (m *addModel) selectedPaths() []string {
	var paths []string
	for i, f := range m.files {
		if m.selected[i] {
			paths = append(paths, f.Path)
		}
	}
	return paths
}

func (m *addModel) pathsToStage() []string {
	paths := m.selectedPaths()
	if len(paths) > 0 {
		return paths
	}
	if m.cursor >= 0 && m.cursor < len(m.files) {
		return []string{m.files[m.cursor].Path}
	}
	return nil
}

func stageSelectedCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		err := app.StageFiles(paths)
		return stageFilesMsg{count: len(paths), all: false, err: err}
	}
}

func stageAllCmd() tea.Cmd {
	return func() tea.Msg {
		err := app.StageAll()
		return stageFilesMsg{all: true, err: err}
	}
}

func (m *addModel) requestStageSelected() tea.Cmd {
	paths := m.pathsToStage()
	if len(paths) == 0 {
		return nil
	}
	return stageSelectedCmd(paths)
}

func (m addModel) Update(msg tea.Msg) (addModel, tea.Cmd) {
	switch msg := msg.(type) {
	case stageFilesMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m addModel) View(width int) string {
	var b strings.Builder
	b.WriteString(styleSection.Render("Adicionar ao stage"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(styleError.Render("  ✖ " + m.err.Error()))
		b.WriteString("\n\n")
	}

	if m.ready {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(styleHint.Render("  (vazio)"))
	}

	count := len(m.selectedPaths())
	footer := fmt.Sprintf("%d selecionado(s) · %d disponível(is)", count, len(m.files))
	b.WriteString("\n\n")
	b.WriteString(styleHint.Render("  " + footer))

	_ = width
	return b.String()
}

func addHelpLine() string {
	return styleKey.Render("↑↓") + " navegar  " +
		styleKey.Render("space") + " selecionar  " +
		styleKey.Render("A") + " todos  " +
		styleKey.Render("Enter") + " adicionar  " +
		styleKey.Render(".") + " git add .  " +
		styleKey.Render("esc") + " voltar"
}
