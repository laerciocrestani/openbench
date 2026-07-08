package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

type syncScreenMode int

const (
	syncScreenModes syncScreenMode = iota
	syncScreenBase
)

type syncModel struct {
	snap        *app.WorkspaceSnapshot
	cursor      int
	modes       []components.SyncModeOption
	base        string
	screen      syncScreenMode
	baseInput   textinput.Model
	baseReady   bool
	dirty       bool
}

func newSyncModel() syncModel {
	return syncModel{
		modes: components.SyncModeCatalog(),
	}
}

func (m *syncModel) Load(snap *app.WorkspaceSnapshot) {
	m.snap = snap
	m.cursor = 0
	m.screen = syncScreenModes
	m.base = "main"
	m.dirty = false
	if snap != nil && snap.Overview != nil {
		m.dirty = snap.Overview.IsDirty()
		if snap.Overview.BaseBranch != "" {
			m.base = snap.Overview.BaseBranch
		}
	}
	m.baseInput = textinput.New()
	m.baseInput.SetValue(m.base)
	m.baseInput.Placeholder = "main"
	m.baseReady = false
}

func (m *syncModel) selectedMode() components.SyncModeOption {
	if m.cursor < 0 || m.cursor >= len(m.modes) {
		return components.SyncModeCatalog()[0]
	}
	return m.modes[m.cursor]
}

func (m *syncModel) buildSyncOptions() app.SyncOptions {
	mode := m.selectedMode()
	prune, pruneRemote, _ := mode.ToAppOptions(m.base)
	return app.SyncOptions{
		Prune:       prune,
		PruneRemote: pruneRemote,
		Base:        m.base,
	}
}

func (m *syncModel) moveCursor(delta int) {
	if len(m.modes) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.modes) {
		m.cursor = len(m.modes) - 1
	}
}

func (m *syncModel) startBaseEdit(width int) tea.Cmd {
	m.screen = syncScreenBase
	m.baseInput.SetValue(m.base)
	m.baseInput.Width = maxInt(width-8, 16)
	m.baseInput.Focus()
	m.baseReady = true
	return textinput.Blink
}

func (m *syncModel) confirmBaseEdit() {
	m.base = strings.TrimSpace(m.baseInput.Value())
	if m.base == "" {
		m.base = "main"
	}
	m.baseInput.Blur()
	m.screen = syncScreenModes
	m.baseReady = false
}

func (m *syncModel) back() {
	if m.screen == syncScreenBase {
		m.baseInput.Blur()
		m.screen = syncScreenModes
		m.baseReady = false
		return
	}
}

func (m *syncModel) canRun() bool {
	return !m.dirty
}

func (m syncModel) View(width int) string {
	switch m.screen {
	case syncScreenBase:
		return components.RenderSyncBaseEditor(m.baseInput.View(), width)
	default:
		return components.RenderSyncOptionsPanel(m.cursor, m.modes, m.base, m.dirty, width)
	}
}

func (m *syncModel) Update(msg tea.Msg) (tea.Cmd, bool) {
	if m.screen == syncScreenBase && m.baseReady {
		var cmd tea.Cmd
		m.baseInput, cmd = m.baseInput.Update(msg)
		return cmd, true
	}
	return nil, false
}

func syncHelpLine(screen syncScreenMode, dirty bool) string {
	switch screen {
	case syncScreenBase:
		return styleKey.Render("Enter") + " confirm  " +
			styleKey.Render("esc") + " back"
	default:
		run := styleKey.Render("Enter") + " run"
		if dirty {
			run = styleHint.Render("Enter (dirty working tree)")
		}
		return styleKey.Render("↑↓") + " option  " +
			styleKey.Render("b") + " base  " +
			run + "  " +
			styleKey.Render("esc") + " back"
	}
}
