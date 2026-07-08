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

const addRowTodos = 0

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

func (m *addModel) rowCount() int {
	if len(m.files) == 0 {
		return 0
	}
	return len(m.files) + 1 // +1 para "Todos"
}

func (m *addModel) Load(snap *app.WorkspaceSnapshot) {
	m.files = app.AddableFiles(snap)
	m.selected = map[int]bool{}
	m.cursor = addRowTodos
	m.err = nil
	if m.ready {
		m.viewport.SetContent(m.listContent())
	}
}

func (m *addModel) SetSize(width, height int) {
	rows := height - 8
	if rows < 6 {
		rows = 6
	}
	total := m.rowCount()
	if total > 0 && rows > total {
		rows = total
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
		return styleHint.Render("  (no files to stage)")
	}
	lines := make([]string, m.rowCount())
	lines[0] = components.RenderAddTodosLine(m.allSelected(), m.cursor == addRowTodos)
	for i, f := range m.files {
		lines[i+1] = components.RenderAddFileLine(m.selected[i], m.cursor == i+1, f)
	}
	return strings.Join(lines, "\n")
}

func (m *addModel) moveCursor(delta int) tea.Cmd {
	if m.rowCount() == 0 {
		return nil
	}
	m.cursor += delta
	if m.cursor < addRowTodos {
		m.cursor = addRowTodos
	}
	max := m.rowCount() - 1
	if m.cursor > max {
		m.cursor = max
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
	if m.cursor == addRowTodos {
		m.toggleAll()
		return
	}
	idx := m.cursor - 1
	if idx < 0 || idx >= len(m.files) {
		return
	}
	m.selected[idx] = !m.selected[idx]
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
	if m.cursor == addRowTodos {
		return nil
	}
	paths := m.selectedPaths()
	if len(paths) > 0 {
		return paths
	}
	idx := m.cursor - 1
	if idx >= 0 && idx < len(m.files) {
		return []string{m.files[idx].Path}
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
	if m.cursor == addRowTodos || m.allSelected() {
		return stageAllCmd()
	}
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
	b.WriteString(styleSection.Render("Stage files"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(styleError.Render("  ✖ " + m.err.Error()))
		b.WriteString("\n\n")
	}

	if m.ready {
		b.WriteString(m.viewport.View())
	} else {
		b.WriteString(styleHint.Render("  (empty)"))
	}

	count := len(m.selectedPaths())
	footer := fmt.Sprintf("%d selected · %d available", count, len(m.files))
	b.WriteString("\n\n")
	b.WriteString(styleHint.Render("  " + footer))

	_ = width
	return b.String()
}

func addHelpLine() string {
	return styleKey.Render("↑↓") + " navigate  " +
		styleKey.Render("space") + " toggle  " +
		styleKey.Render("Enter") + " stage  " +
		styleKey.Render(".") + " git add .  " +
		styleKey.Render("esc") + " back"
}
