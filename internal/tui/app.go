package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/components"
	"github.com/laerciocrestani/gitai/internal/tui/views"
	"github.com/laerciocrestani/gitai/internal/ui"
)

type snapshotMsg struct {
	snap   *app.WorkspaceSnapshot
	err    error
	silent bool
}

type diffLoadedMsg struct {
	title string
	diff  string
	err   error
}

type tickMsg struct{}

type logsLoadedMsg struct {
	content string
	err     error
}

type appModel struct {
	screen         Screen
	snapshot       *app.WorkspaceSnapshot
	width          int
	height         int
	loading        bool
	loadTick       int
	loadProg       *ActionProgress
	err            error
	status         string
	diff           diffModel
	logs           logsModel
	branches       branchesModel
	sync           syncModel
	add            addModel
	report         reportModel
	action         *actionState
	refresh        refreshConfig
	refreshPending bool
}

func newApp(cfg refreshConfig) appModel {
	return appModel{
		screen:   ScreenDashboard,
		loading:  true,
		status:   "Loading repository…",
		loadProg: NewActionProgress(),
		diff:     newDiffModel(),
		logs:     newLogsModel(),
		branches: newBranchesModel(),
		sync:     newSyncModel(),
		add:      newAddModel(),
		report:   newReportModel(),
		refresh:  cfg,
	}
}

func loadLogsCmdFromSnap(snap *app.WorkspaceSnapshot) tea.Cmd {
	return func() tea.Msg {
		log, err := app.LoadLog(snap)
		return logsLoadedMsg{content: log, err: err}
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(loadSnapshotCmd(m.loadProg), tickCmd(), initRefreshCmds(m.refresh))
}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m appModel) shouldAnimateSpinner() bool {
	if m.loading {
		return true
	}
	if m.action != nil && (m.action.phase == PhaseRunning || m.action.phase == PhaseConfirming) {
		return true
	}
	if m.screen == ScreenReport && !m.report.ready {
		return true
	}
	if m.screen == ScreenBranches && m.branches.detailLoading {
		return true
	}
	return false
}

func loadSnapshotCmd(prog *ActionProgress) tea.Cmd {
	return func() tea.Msg {
		if prog != nil {
			prog.Reset()
		}
		snap, err := app.LoadWorkspaceSnapshotWithProgress(prog)
		return snapshotMsg{snap: snap, err: err}
	}
}

func (m appModel) applySnapshot(msg snapshotMsg) (appModel, tea.Cmd) {
	var cmds []tea.Cmd

	if msg.silent && msg.err == nil && !snapshotChanged(m.snapshot, msg.snap) {
		m.refreshPending = false
		cmds = append(cmds, m.reschedulePollIfNeeded())
		return m, tea.Batch(cmds...)
	}

	if !msg.silent {
		m.loading = false
	}

	m.refreshPending = false
	m.snapshot = msg.snap
	m.err = msg.err

	if msg.err != nil {
		if !msg.silent {
			m.status = msg.err.Error()
		}
	} else if !msg.silent {
		m.status = "Ready"
	} else {
		m.status = "Updated"
	}

	if msg.err == nil && m.screen == ScreenDiff {
		cmds = append(cmds, loadDiffCmd(msg.snap))
	}

	cmds = append(cmds, m.reschedulePollIfNeeded())
	return m, tea.Batch(cmds...)
}

func loadDiffCmd(snap *app.WorkspaceSnapshot) tea.Cmd {
	return func() tea.Msg {
		title, diff, err := app.LoadDiff(snap)
		return diffLoadedMsg{title: title, diff: diff, err: err}
	}
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		cmds := []tea.Cmd{tickCmd()}
		if m.shouldAnimateSpinner() {
			m.loadTick++
		}
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.screen == ScreenDiff {
			m.diff.SetSize(m.width, m.height)
		}
		if m.screen == ScreenLogs {
			m.logs.SetSize(m.width, m.height)
		}
		if m.screen == ScreenBranches {
			m.branches.SetSize(m.width, m.height)
		}
		if m.screen == ScreenAdd {
			m.add.SetSize(m.width, m.height)
		}
		if m.screen == ScreenReport {
			m.report.SetSize(m.width, m.height)
		}
		if m.action != nil {
			m.action.resizeEditors(m.width, m.height)
		}
		return m, nil

	case snapshotMsg:
		return m.applySnapshot(msg)

	case pollRefreshMsg:
		return m.requestAutoRefresh()

	case watchRefreshMsg:
		return m.requestAutoRefresh()

	case debouncedRefreshMsg:
		if !m.canAutoRefresh() {
			m.refreshPending = false
			return m, m.reschedulePollIfNeeded()
		}
		m.refreshPending = false
		return m, loadSnapshotSilent()

	case diffLoadedMsg:
		m.diff.Load(msg.title, msg.diff, msg.err)
		m.diff.SetSize(m.width, m.height)
		if msg.err != nil {
			m.status = msg.err.Error()
		}
		return m, nil

	case logsLoadedMsg:
		m.logs.Load(msg.content, msg.err)
		m.logs.SetSize(m.width, m.height)
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = "Logs"
		}
		return m, nil

	case branchDetailMsg:
		var cmd tea.Cmd
		m.branches, cmd = m.branches.Update(msg)
		return m, cmd

	case branchCheckoutMsg:
		var cmds []tea.Cmd
		m.branches, _ = m.branches.Update(msg)
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.screen = ScreenDashboard
		m.loading = true
		m.status = "Refreshing…"
		cmds = append(cmds, loadSnapshotCmd(m.loadProg))
		return m, tea.Batch(cmds...)

	case branchCreateMsg:
		var cmds []tea.Cmd
		if msg.err != nil {
			m.branches.newErr = msg.err
			m.status = msg.err.Error()
			return m, nil
		}
		m.branches.cancelNewBranch()
		m.screen = ScreenDashboard
		m.loading = true
		m.status = "Branch created: " + msg.name
		cmds = append(cmds, loadSnapshotCmd(m.loadProg))
		return m, tea.Batch(cmds...)

	case stageFilesMsg:
		var cmds []tea.Cmd
		m.add, _ = m.add.Update(msg)
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.screen = ScreenDashboard
		m.loading = true
		if msg.all {
			m.status = "git add ."
		} else if msg.count == 1 {
			m.status = "1 file staged"
		} else {
			m.status = fmt.Sprintf("%d files staged", msg.count)
		}
		cmds = append(cmds, loadSnapshotCmd(m.loadProg))
		return m, tea.Batch(cmds...)

	case reportLoadedMsg:
		m.report.Load(msg)
		m.report.SetSize(m.width, m.height)
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = "AI usage"
		}
		return m, nil

	case actionPreviewMsg:
		if m.action != nil {
			m.action.handlePreview(msg)
		}
		return m, nil

	case actionConfirmMsg:
		if m.action != nil {
			m.action.handleConfirm(msg)
		}
		return m, nil

	case actionSimpleMsg:
		if m.action != nil {
			m.action.handleSimple(msg)
		}
		return m, nil

	case tea.KeyMsg:
		if key, ok := parseGlobalKey(msg); ok && m.screen == ScreenDashboard && m.action == nil {
			switch key {
			case keyQuit:
				return m, tea.Quit
			case keyRefresh:
				m.loading = true
				m.status = "Refreshing…"
				return m, loadSnapshotCmd(m.loadProg)
			}
		}

		switch m.screen {
		case ScreenDashboard:
			return m.updateDashboard(msg)
		case ScreenDiff:
			return m.updateDiff(msg)
		case ScreenLogs:
			return m.updateLogs(msg)
		case ScreenBranches:
			return m.updateBranches(msg)
		case ScreenAdd:
			return m.updateAdd(msg)
		case ScreenSync:
			return m.updateSync(msg)
		case ScreenAction:
			return m.updateActionMsg(msg)
		case ScreenReport:
			return m.updateReport(msg)
		case ScreenHelp:
			return m.updateHelp(msg)
		}
	}

	if m.screen == ScreenDiff {
		var cmd tea.Cmd
		m.diff, cmd = m.diff.Update(msg)
		return m, cmd
	}

	if m.screen == ScreenLogs {
		var cmd tea.Cmd
		m.logs, cmd = m.logs.Update(msg)
		return m, cmd
	}

	if m.screen == ScreenBranches {
		var cmd tea.Cmd
		m.branches, cmd = m.branches.Update(msg)
		return m, cmd
	}

	if m.screen == ScreenSync {
		var cmd tea.Cmd
		cmd, _ = m.sync.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	}

	if m.screen == ScreenAdd {
		var cmd tea.Cmd
		m.add, cmd = m.add.Update(msg)
		return m, cmd
	}

	if m.screen == ScreenReport {
		var cmd tea.Cmd
		m.report, cmd = m.report.Update(msg)
		return m, cmd
	}

	if m.screen == ScreenAction && m.action != nil && m.action.phase == PhaseConfirm && m.action.editing {
		var cmd tea.Cmd
		m.action, cmd = m.action.updateEditors(msg)
		return m, cmd
	}

	return m, nil
}

func (m appModel) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading || m.err != nil {
		if key, ok := parseGlobalKey(msg); ok {
			switch key {
			case keyQuit:
				return m, tea.Quit
			case keyRefresh:
				m.loading = true
				m.status = "Refreshing…"
				return m, loadSnapshotCmd(m.loadProg)
			}
		}
		return m, nil
	}

	if key, ok := parseGlobalKey(msg); ok {
		switch key {
		case keyQuit:
			return m, tea.Quit
		case keyRefresh:
			m.loading = true
			m.status = "Refreshing…"
			return m, loadSnapshotCmd(m.loadProg)
		}
	}

	if dashKey, ok := parseDashboardKey(msg, m.snapshot); ok {
		switch dashKey {
		case dashKeyDiff:
			m.screen = ScreenDiff
			m.status = "Diff"
			return m, loadDiffCmd(m.snapshot)
		case dashKeyLogs:
			m.screen = ScreenLogs
			m.status = "Logs"
			return m, loadLogsCmdFromSnap(m.snapshot)
		case dashKeyBranches:
			m.screen = ScreenBranches
			m.status = "Branches"
			cmd := m.branches.Load(m.snapshot)
			m.branches.SetSize(m.width, m.height)
			return m, cmd
		case dashKeyAdd:
			m.screen = ScreenAdd
			m.status = "Add"
			m.add.SetSize(m.width, m.height)
			m.add.Load(m.snapshot)
			return m, nil
		case dashKeyCommit:
			m.screen = ScreenAction
			m.action = newActionState(ActionCommit)
			m.status = "Commit"
			return m, m.action.previewCmd()
		case dashKeyPush:
			m.screen = ScreenAction
			m.action = newActionState(ActionPush)
			m.status = "Push"
			return m, m.action.previewCmd()
		case dashKeyPR:
			m.screen = ScreenAction
			m.action = newActionState(ActionPR)
			m.status = "PR"
			return m, m.action.previewCmd()
		case dashKeySync:
			m.screen = ScreenAction
			m.action = newSyncActionState(app.SyncOptions{})
			m.status = "Sync"
			return m, m.action.directCmd()
		case dashKeySyncOptions:
			m.screen = ScreenSync
			m.status = "Sync · options"
			m.sync.Load(m.snapshot)
			return m, nil
		case dashKeyOpenPR:
			m.screen = ScreenAction
			m.action = newActionState(ActionOpenPR)
			m.status = "Opening PR"
			return m, m.action.directCmd()
		case dashKeyCopyHash:
			if m.snapshot != nil && m.snapshot.Overview != nil {
				hash := m.snapshot.Overview.HeadFullHash
				if hash == "" {
					hash = m.snapshot.Overview.HeadHash
				}
				if err := ui.CopyToClipboard(hash); err != nil {
					m.status = "Failed to copy hash"
				} else {
					m.status = "Hash copied: " + hash
				}
			}
			return m, nil
		case dashKeyReport:
			m.screen = ScreenReport
			m.status = "AI usage"
			return m, loadReportCmd(m.report.period)
		case dashKeyHelp:
			m.screen = ScreenHelp
			m.status = "Help"
			return m, nil
		}
	}

	return m, nil
}

func (m appModel) updateDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenDashboard
		m.status = "Ready"
		return m, nil
	}
	var cmd tea.Cmd
	m.diff, cmd = m.diff.Update(msg)
	return m, cmd
}

func (m appModel) updateLogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenDashboard
		m.status = "Ready"
		return m, nil
	}
	var cmd tea.Cmd
	m.logs, cmd = m.logs.Update(msg)
	return m, cmd
}

func (m appModel) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenDashboard
		m.status = "Ready"
		return m, nil
	case "up", "k":
		return m, m.add.moveCursor(-1)
	case "down", "j":
		return m, m.add.moveCursor(1)
	case " ":
		m.add.toggleCursor()
		return m, nil
	case "enter":
		return m, m.add.requestStageSelected()
	case ".":
		return m, stageAllCmd()
	}
	var cmd tea.Cmd
	m.add, cmd = m.add.Update(msg)
	return m, cmd
}

func (m appModel) updateSync(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.sync.screen == syncScreenBase {
			m.sync.back()
			return m, nil
		}
		m.screen = ScreenDashboard
		m.status = "Ready"
		return m, nil
	case "up", "k":
		if m.sync.screen == syncScreenModes {
			m.sync.moveCursor(-1)
		}
		return m, nil
	case "down", "j":
		if m.sync.screen == syncScreenModes {
			m.sync.moveCursor(1)
		}
		return m, nil
	case "b":
		if m.sync.screen == syncScreenModes {
			return m, m.sync.startBaseEdit(m.width)
		}
	case "enter":
		if m.sync.screen == syncScreenBase {
			m.sync.confirmBaseEdit()
			return m, nil
		}
		if !m.sync.canRun() {
			return m, nil
		}
		opts := m.sync.buildSyncOptions()
		m.screen = ScreenAction
		m.action = newSyncActionState(opts)
		m.status = "Sync"
		return m, m.action.directCmd()
	}
	var cmd tea.Cmd
	cmd, _ = m.sync.Update(msg)
	return m, cmd
}

func (m appModel) updateBranches(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.branches.mode == branchesModeNew {
		if cmd, handled := m.branches.updateNewBranch(msg); handled {
			return m, cmd
		}
	}

	switch msg.String() {
	case "esc":
		if m.branches.mode == branchesModeNew {
			return m, m.branches.newBranchBack()
		}
		if m.branches.confirmDirty {
			m.branches.confirmDirty = false
			m.branches.checkoutTarget = ""
			return m, nil
		}
		m.screen = ScreenDashboard
		m.status = "Ready"
		return m, nil
	case "n":
		if m.branches.mode == branchesModeList {
			return m, m.branches.startNewBranch()
		}
	case "up", "k":
		if m.branches.mode == branchesModeList {
			return m, m.branches.moveCursor(-1)
		}
	case "down", "j":
		if m.branches.mode == branchesModeList {
			return m, m.branches.moveCursor(1)
		}
	case "enter":
		if m.branches.mode == branchesModeList {
			return m, m.branches.requestCheckout()
		}
	}
	var cmd tea.Cmd
	m.branches, cmd = m.branches.Update(msg)
	return m, cmd
}

func (m appModel) updateReport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenDashboard
		m.status = "Ready"
		return m, nil
	case "r":
		return m, loadReportCmd(m.report.period)
	case "1":
		m.report.period = report24h
		return m, loadReportCmd(report24h)
	case "2":
		m.report.period = report7d
		return m, loadReportCmd(report7d)
	case "3":
		m.report.period = reportMonth
		return m, loadReportCmd(reportMonth)
	case "a":
		m.report.period = reportAll
		return m, loadReportCmd(reportAll)
	}
	var cmd tea.Cmd
	m.report, cmd = m.report.Update(msg)
	return m, cmd
}

func (m appModel) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "q":
		m.screen = ScreenDashboard
		m.status = "Ready"
		return m, nil
	}
	return m, nil
}

func (m appModel) updateActionMsg(msg tea.Msg) (appModel, tea.Cmd) {
	if m.action == nil {
		m.screen = ScreenDashboard
		return m, nil
	}

	if m.action.phase == PhaseConfirm && m.action.editing {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "esc", "e":
				m.action.exitEdit()
				return m, nil
			case "tab":
				if m.action.kind == ActionPR {
					return m, m.action.cyclePRFocus()
				}
			}
		}
		var cmd tea.Cmd
		m.action, cmd = m.action.updateEditors(msg)
		return m, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch m.action.phase {
	case PhaseConfirm:
		switch keyMsg.String() {
		case "esc":
			return m.closeAction(), nil
		case "enter":
			if m.action.editorsReady {
				m.action.syncPreviewFromEditors()
			}
			m.action.editing = false
			m.action.phase = PhaseConfirming
			return m, m.action.confirmCmd()
		case "e":
			if m.action.canEditPreview() {
				return m, m.action.enterEdit(m.width, m.height)
			}
		case "d":
			if m.action.kind == ActionPR {
				m.action.toggleDraft()
			}
		}

	case PhaseConfirming:
		// aguarda mensagem async

	case PhaseDone, PhaseError:
		switch keyMsg.String() {
		case "enter", "esc", "q":
			return m.closeActionAndRefresh(), loadSnapshotCmd(m.loadProg)
		}
	}

	return m, nil
}

func (m appModel) updateAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.updateActionMsg(msg)
}

func (m appModel) closeAction() appModel {
	m.screen = ScreenDashboard
	m.action = nil
	m.status = "Ready"
	return m
}

func (m appModel) closeActionAndRefresh() appModel {
	m.screen = ScreenDashboard
	m.action = nil
	m.loading = true
	m.status = "Refreshing…"
	return m
}

func (m appModel) View() string {
	if m.width == 0 {
		return "Starting…"
	}

	var b strings.Builder

	if terminalTooSmall(m.width, m.height) {
		b.WriteString(styleWarn.Render(fmt.Sprintf(
			"  Terminal too small (%dx%d) — recommended %dx%d+\n",
			m.width, m.height, minWidth, minHeight,
		)))
	}

	var ctx *ui.HeaderContext
	if m.snapshot != nil {
		c := app.BuildHeaderContext(m.snapshot)
		ctx = &c
	}
	b.WriteString(ui.FormatDashboardHeader(ctx, m.width, false, !themePlain()))

	help := dashboardHelpLine()

	switch m.screen {
	case ScreenDiff:
		b.WriteString(m.diff.View(m.width))
		help = diffHelpLine()
	case ScreenLogs:
		b.WriteString(m.logs.View(m.width))
		help = logsHelpLine()
	case ScreenBranches:
		b.WriteString(m.branches.View(m.width, m.loadTick))
		if m.branches.mode == branchesModeNew {
			help = newBranchHelpLine(m.branches.newStep)
		} else {
			help = branchesHelpLine()
		}
	case ScreenAdd:
		b.WriteString(m.add.View(m.width))
		help = addHelpLine()
	case ScreenSync:
		b.WriteString(m.sync.View(m.width))
		help = syncHelpLine(m.sync.screen, m.sync.dirty)
	case ScreenReport:
		b.WriteString(m.report.View(m.loadTick))
		help = reportHelpLine()
	case ScreenHelp:
		b.WriteString("\n")
		b.WriteString(helpContent())
		help = helpHelpLine()
	case ScreenAction:
		if m.action != nil {
			if m.action.phase == PhaseRunning || m.action.phase == PhaseConfirming {
				status, logs := m.action.progress.Snapshot()
				if status == "" {
					status = "Generating…"
				}
				b.WriteString(components.RenderLoading(status, components.AlertLogs(logs), m.loadTick, m.width))
			} else {
				b.WriteString(m.action.View(m.width, m.height))
			}
			help = actionHelpLine()
			if m.action.phase == PhaseConfirm {
				help = actionConfirmHelp(m.action)
			}
			if m.action.phase == PhaseDone || m.action.phase == PhaseError {
				help = styleKey.Render("enter") + " back"
			}
		}
	default:
		if m.loading {
			status, logs := m.loadProg.Snapshot()
			if status == "" {
				status = m.status
			}
			b.WriteString(views.RenderLoadingDashboard(status, components.AlertLogs(logs), m.loadTick, m.width))
		} else if m.err != nil {
			b.WriteString("\n")
			b.WriteString(styleError.Render("  ✖ " + m.err.Error()))
		} else {
			b.WriteString(views.RenderDashboard(m.snapshot, views.DashboardOptions{
				Width:  m.width,
				Height: m.height,
				Tick:   m.loadTick,
			}))
		}
	}

	if m.screen == ScreenDashboard && !m.loading && m.err == nil {
		b.WriteString(views.RenderFooterBar(m.snapshot, m.width))
	} else if help != "" {
		b.WriteString(renderStatusBar(m.width, m.status, help))
	} else if m.status != "" {
		b.WriteString(renderStatusBar(m.width, m.status, ""))
	}
	return b.String()
}

func renderStatusBar(width int, left, right string) string {
	if right == "" {
		right = left
		left = ""
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return "\n" + styleStatusBar.Width(width).Render(line)
}
