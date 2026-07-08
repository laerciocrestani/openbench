package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

type branchesModel struct {
	snap           *app.WorkspaceSnapshot
	branches       []gitpkg.BranchInfo
	base           string
	cursor         int
	listViewport   viewport.Model
	detail         *gitpkg.BranchDetail
	detailLoading  bool
	detailFor      string
	err            error
	confirmDirty   bool
	checkoutTarget string
	dirty          bool
	ready          bool
	checkoutOK     bool
}

type branchDetailMsg struct {
	name   string
	detail *gitpkg.BranchDetail
	err    error
}

type branchCheckoutMsg struct {
	target string
	err    error
}

func newBranchesModel() branchesModel {
	return branchesModel{}
}

func (m *branchesModel) Load(snap *app.WorkspaceSnapshot) tea.Cmd {
	m.snap = snap
	m.branches = nil
	m.base = "main"
	m.dirty = false
	m.cursor = 0
	m.detail = nil
	m.detailFor = ""
	m.detailLoading = false
	m.confirmDirty = false
	m.checkoutTarget = ""
	m.checkoutOK = false
	m.err = nil
	m.ready = false
	m.detailLoading = true

	if snap != nil && snap.Overview != nil {
		m.branches = append([]gitpkg.BranchInfo(nil), snap.Overview.Branches...)
		m.base = snap.Overview.BaseBranch
		m.dirty = snap.Overview.IsDirty()
		for i, b := range m.branches {
			if b.Current {
				m.cursor = i
				break
			}
		}
	}
	if m.base == "" {
		m.base = "main"
	}
	return loadBranchDetailCmd(m.snap, m.selectedBranch())
}

func (m *branchesModel) SetSize(width, height int) {
	listRows := height/3 - 2
	if listRows < 6 {
		listRows = 6
	}
	if len(m.branches) > 0 && listRows > len(m.branches)+1 {
		listRows = len(m.branches) + 1
	}
	if listRows < 4 {
		listRows = 4
	}

	if !m.ready {
		m.listViewport = viewport.New(width, listRows)
		m.ready = true
	} else {
		m.listViewport.Width = width
		m.listViewport.Height = listRows
	}
	m.listViewport.SetContent(m.listContent())
}

func (m *branchesModel) listContent() string {
	var lines []string
	for i, b := range m.branches {
		lines = append(lines, components.RenderBranchListLine(b, i == m.cursor))
	}
	if len(lines) == 0 {
		return styleHint.Render("  (nenhuma branch local)")
	}
	return strings.Join(lines, "\n")
}

func (m *branchesModel) selectedBranch() string {
	if m.cursor < 0 || m.cursor >= len(m.branches) {
		return ""
	}
	return m.branches[m.cursor].Name
}

func (m *branchesModel) isCurrentSelected() bool {
	if m.cursor < 0 || m.cursor >= len(m.branches) {
		return false
	}
	return m.branches[m.cursor].Current
}

func loadBranchDetailCmd(snap *app.WorkspaceSnapshot, name string) tea.Cmd {
	return func() tea.Msg {
		if name == "" {
			return branchDetailMsg{name: name, err: nil}
		}
		detail, err := app.LoadBranchDetail(name, snap)
		return branchDetailMsg{name: name, detail: detail, err: err}
	}
}

func checkoutBranchCmd(name string) tea.Cmd {
	return func() tea.Msg {
		err := app.CheckoutBranch(name)
		return branchCheckoutMsg{target: name, err: err}
	}
}

func (m *branchesModel) moveCursor(delta int) tea.Cmd {
	if len(m.branches) == 0 {
		return nil
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.branches) {
		m.cursor = len(m.branches) - 1
	}
	m.listViewport.SetContent(m.listContent())
	if m.cursor >= m.listViewport.Height {
		m.listViewport.SetYOffset(m.cursor - m.listViewport.Height + 1)
	} else if m.cursor < m.listViewport.YOffset {
		m.listViewport.SetYOffset(m.cursor)
	}
	m.confirmDirty = false
	m.checkoutTarget = ""
	m.detailLoading = true
	return loadBranchDetailCmd(m.snap, m.selectedBranch())
}

func (m *branchesModel) requestCheckout() tea.Cmd {
	target := m.selectedBranch()
	if target == "" || m.isCurrentSelected() {
		return nil
	}
	if m.dirty && !m.confirmDirty {
		m.confirmDirty = true
		m.checkoutTarget = target
		return nil
	}
	m.confirmDirty = false
	return checkoutBranchCmd(target)
}

func (m branchesModel) Update(msg tea.Msg) (branchesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case branchDetailMsg:
		if msg.name != m.selectedBranch() {
			return m, nil
		}
		m.detailLoading = false
		m.detailFor = msg.name
		m.detail = msg.detail
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
		}
		return m, nil

	case branchCheckoutMsg:
		if msg.err != nil {
			m.err = msg.err
			m.confirmDirty = false
			return m, nil
		}
		m.checkoutOK = true
		m.err = nil
		return m, nil
	}

	var cmd tea.Cmd
	m.listViewport, cmd = m.listViewport.Update(msg)
	return m, cmd
}

func (m branchesModel) View(width int) string {
	var b strings.Builder
	b.WriteString(styleSection.Render("Branches"))
	if m.base != "" {
		b.WriteString(styleHint.Render("  base: " + m.base))
	}
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(styleError.Render("  ✗ " + m.err.Error()))
		b.WriteString("\n\n")
	}

	if m.confirmDirty {
		b.WriteString(styleWarn.Render("  Working tree com alterações não commitadas."))
		b.WriteString("\n")
		b.WriteString(styleWarn.Render("  Enter confirma checkout para " + m.checkoutTarget))
		b.WriteString("\n")
		b.WriteString(styleHint.Render("  esc cancela"))
		b.WriteString("\n\n")
	}

	if m.ready {
		b.WriteString(m.listViewport.View())
	} else {
		b.WriteString(styleHint.Render("  (vazio)"))
	}
	b.WriteString("\n")

	if m.detailLoading || m.detailFor != m.selectedBranch() {
		b.WriteString(components.RenderBranchDetail(nil, m.base, width))
	} else {
		b.WriteString(components.RenderBranchDetail(m.detail, m.base, width))
	}

	return b.String()
}

func branchesHelpLine() string {
	return styleKey.Render("↑↓") + " navegar  " +
		styleKey.Render("Enter") + " checkout  " +
		styleKey.Render("esc") + " voltar"
}
