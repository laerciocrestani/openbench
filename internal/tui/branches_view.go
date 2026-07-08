package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

type branchesMode int

const (
	branchesModeList branchesMode = iota
	branchesModeNew
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

	mode             branchesMode
	newStep          components.NewBranchStep
	fromCursor       int
	templateCursor   int
	fromBranch       string
	selectedTemplate components.NewBranchTemplate
	templateItems    []components.NewBranchTemplateItem
	fromViewport     viewport.Model
	templateViewport viewport.Model
	nameInput        textinput.Model
	nameReady        bool
	newErr           error
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
	m.mode = branchesModeList
	m.newErr = nil

	if branches, err := app.ListBranches(); err == nil && len(branches) > 0 {
		m.branches = branches
	} else if snap != nil && snap.Overview != nil {
		m.branches = append([]gitpkg.BranchInfo(nil), snap.Overview.Branches...)
	}

	if snap != nil && snap.Overview != nil {
		if m.base == "main" && snap.Overview.BaseBranch != "" {
			m.base = snap.Overview.BaseBranch
		}
		m.dirty = snap.Overview.IsDirty()
	}
	for i, b := range m.branches {
		if b.Current {
			m.cursor = i
			break
		}
	}
	if m.base == "" {
		m.base = "main"
	}

	m.detailLoading = true
	m.refreshListContent()
	return loadBranchDetailCmd(m.snap, m.selectedBranch())
}

const newBranchPickerRows = 10

func (m *branchesModel) SetSize(width, height int) {
	listRows := height/3 - 2
	if listRows < 6 {
		listRows = 6
	}
	if len(m.branches) > 0 && listRows > len(m.branches) {
		listRows = len(m.branches)
	}
	if listRows < 4 {
		listRows = 4
	}

	pickerRows := newBranchPickerRows
	if height > 0 && pickerRows > height-10 {
		pickerRows = height - 10
	}
	if pickerRows < 4 {
		pickerRows = 4
	}

	if !m.ready {
		m.listViewport = viewport.New(width, listRows)
		m.fromViewport = viewport.New(width, pickerRows)
		m.templateViewport = viewport.New(width, pickerRows)
		m.ready = true
	} else {
		m.listViewport.Width = width
		m.listViewport.Height = listRows
		m.fromViewport.Width = width
		m.fromViewport.Height = pickerRows
		m.templateViewport.Width = width
		m.templateViewport.Height = pickerRows
	}
	m.refreshListContent()
	if m.mode == branchesModeNew {
		m.refreshNewBranchContent()
	}
}

func (m *branchesModel) refreshListContent() {
	if !m.ready {
		return
	}
	m.listViewport.SetContent(m.listContent())
	m.syncScroll()
}

func (m *branchesModel) listContent() string {
	if len(m.branches) == 0 {
		return ""
	}
	lines := make([]string, len(m.branches))
	for i, b := range m.branches {
		lines[i] = components.RenderBranchListLineNumbered(i, b, i == m.cursor)
	}
	return strings.Join(lines, "\n")
}

func (m *branchesModel) listBody() string {
	if len(m.branches) == 0 {
		return ""
	}
	if m.ready {
		return m.listViewport.View()
	}
	return m.listContent()
}

func (m *branchesModel) syncScroll() {
	if !m.ready || len(m.branches) == 0 {
		return
	}
	if m.cursor >= m.listViewport.YOffset+m.listViewport.Height {
		m.listViewport.SetYOffset(m.cursor - m.listViewport.Height + 1)
	} else if m.cursor < m.listViewport.YOffset {
		m.listViewport.SetYOffset(m.cursor)
	}
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
	m.refreshListContent()
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

func (m branchesModel) View(width, tick int) string {
	if m.mode == branchesModeNew {
		return m.viewNewBranch(width)
	}

	var b strings.Builder

	if m.err != nil {
		b.WriteString(styleError.Render("  ✖ " + m.err.Error()))
		b.WriteString("\n\n")
	}

	if m.confirmDirty {
		b.WriteString(styleWarn.Render("  Working tree has uncommitted changes."))
		b.WriteString("\n")
		b.WriteString(styleWarn.Render("  Enter confirms checkout to " + m.checkoutTarget))
		b.WriteString("\n")
		b.WriteString(styleHint.Render("  esc cancels"))
		b.WriteString("\n\n")
	}

	b.WriteString(components.RenderBranchesPanel(m.cursor, len(m.branches), m.base, m.listBody(), width))
	b.WriteString("\n")

	selected := m.selectedBranch()
	if m.detailLoading || m.detailFor != selected {
		b.WriteString(components.RenderBranchDetail(nil, selected, m.base, width, tick))
	} else {
		b.WriteString(components.RenderBranchDetail(m.detail, selected, m.base, width, tick))
	}

	return b.String()
}

func branchesHelpLine() string {
	return styleKey.Render("↑↓") + " navigate  " +
		styleKey.Render("Enter") + " checkout  " +
		styleKey.Render("n") + " new branch  " +
		styleKey.Render("esc") + " back"
}

func newBranchHelpLine(step components.NewBranchStep) string {
	switch step {
	case components.NewBranchStepName:
		return styleKey.Render("Enter") + " create  " +
			styleKey.Render("esc") + " back  " +
			styleKey.Render("tab") + " edit name"
	default:
		return styleKey.Render("↑↓") + " navigate  " +
			styleKey.Render("Enter") + " next  " +
			styleKey.Render("esc") + " back"
	}
}
