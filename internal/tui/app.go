package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/laerciocrestani/gitai/internal/app"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
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

type appModel struct {
	screen         Screen
	snapshot       *app.WorkspaceSnapshot
	width          int
	height         int
	loading        bool
	err            error
	status         string
	diff           diffModel
	report         reportModel
	action         *actionState
	refresh        refreshConfig
	refreshPending bool
}

func newApp(cfg refreshConfig) appModel {
	return appModel{
		screen:  ScreenDashboard,
		loading: true,
		status:  "Carregando repositório…",
		diff:    newDiffModel(),
		report:  newReportModel(),
		refresh: cfg,
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(loadSnapshot, initRefreshCmds(m.refresh))
}

func loadSnapshot() tea.Msg {
	snap, err := app.LoadWorkspaceSnapshot()
	return snapshotMsg{snap: snap, err: err}
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
		m.status = "Pronto"
	} else {
		m.status = "Atualizado"
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.screen == ScreenDiff {
			m.diff.SetSize(m.width, m.height)
		}
		if m.screen == ScreenReport {
			m.report.SetSize(m.width, m.height)
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

	case reportLoadedMsg:
		m.report.Load(msg)
		m.report.SetSize(m.width, m.height)
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = "Uso de IA"
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
				m.status = "Atualizando…"
				return m, loadSnapshot
			}
		}

		switch m.screen {
		case ScreenDashboard:
			return m.updateDashboard(msg)
		case ScreenDiff:
			return m.updateDiff(msg)
		case ScreenAction:
			return m.updateAction(msg)
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

	if m.screen == ScreenReport {
		var cmd tea.Cmd
		m.report, cmd = m.report.Update(msg)
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
				m.status = "Atualizando…"
				return m, loadSnapshot
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
			m.status = "Atualizando…"
			return m, loadSnapshot
		}
	}

	if dashKey, ok := parseDashboardKey(msg, m.snapshot); ok {
		switch dashKey {
		case dashKeyDiff:
			m.screen = ScreenDiff
			m.status = "Diff"
			return m, loadDiffCmd(m.snapshot)
		case dashKeyCommit:
			m.screen = ScreenAction
			m.action = newActionState(ActionCommit)
			m.status = "Commit"
			return m, m.action.previewCmd()
		case dashKeyPush:
			m.screen = ScreenAction
			m.action = newActionState(ActionPush)
			m.status = "Push"
			return m, m.action.directCmd()
		case dashKeyPR:
			m.screen = ScreenAction
			m.action = newActionState(ActionPR)
			m.status = "PR"
			return m, m.action.previewCmd()
		case dashKeySync:
			m.screen = ScreenAction
			m.action = newActionState(ActionSync)
			m.status = "Sync"
			return m, m.action.directCmd()
		case dashKeyOpenPR:
			m.screen = ScreenAction
			m.action = newActionState(ActionOpenPR)
			m.status = "Abrindo PR"
			return m, m.action.directCmd()
		case dashKeyReport:
			m.screen = ScreenReport
			m.status = "Uso de IA"
			return m, loadReportCmd(m.report.period)
		case dashKeyHelp:
			m.screen = ScreenHelp
			m.status = "Ajuda"
			return m, nil
		}
	}

	return m, nil
}

func (m appModel) updateDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenDashboard
		m.status = "Pronto"
		return m, nil
	}
	var cmd tea.Cmd
	m.diff, cmd = m.diff.Update(msg)
	return m, cmd
}

func (m appModel) updateReport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = ScreenDashboard
		m.status = "Pronto"
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
		m.status = "Pronto"
		return m, nil
	}
	return m, nil
}

func (m appModel) updateAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.action == nil {
		m.screen = ScreenDashboard
		return m, nil
	}

	switch m.action.phase {
	case PhaseConfirm:
		switch msg.String() {
		case "esc":
			return m.closeAction(), nil
		case "enter":
			m.action.phase = PhaseConfirming
			return m, m.action.confirmCmd()
		case "d":
			if m.action.kind == ActionPR {
				m.action.toggleDraft()
			}
		}

	case PhaseConfirming:
		// aguarda mensagem async

	case PhaseDone, PhaseError:
		switch msg.String() {
		case "enter", "esc", "q":
			return m.closeActionAndRefresh(), loadSnapshot
		}
	}

	return m, nil
}

func (m appModel) closeAction() tea.Model {
	m.screen = ScreenDashboard
	m.action = nil
	m.status = "Pronto"
	return m
}

func (m appModel) closeActionAndRefresh() tea.Model {
	m.screen = ScreenDashboard
	m.action = nil
	m.loading = true
	m.status = "Atualizando…"
	return m
}

func (m appModel) View() string {
	if m.width == 0 {
		return "Iniciando…"
	}

	var b strings.Builder

	if terminalTooSmall(m.width, m.height) {
		b.WriteString(styleWarn.Render(fmt.Sprintf(
			"  Terminal pequeno (%dx%d) — recomendado %dx%d+\n",
			m.width, m.height, minWidth, minHeight,
		)))
	}

	logo := styleTitle.Render("●──────────────●")
	title := styleTitle.Render("GITAI")
	tagline := styleHeader.Render("AI-powered Git Workflow · " + ui.Version())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, logo, "    ", title))
	b.WriteString("\n")
	b.WriteString("  " + tagline)
	b.WriteString("\n\n")

	help := dashboardHelpLine()

	switch m.screen {
	case ScreenDiff:
		b.WriteString(m.diff.View(m.width))
		help = diffHelpLine()
	case ScreenReport:
		b.WriteString(m.report.View())
		help = reportHelpLine()
	case ScreenHelp:
		b.WriteString("\n")
		b.WriteString(helpContent())
		help = helpHelpLine()
	case ScreenAction:
		if m.action != nil {
			b.WriteString(m.action.View(m.width))
			help = actionHelpLine()
			if m.action.phase == PhaseConfirm {
				help = actionConfirmHelp(m.action)
			}
			if m.action.phase == PhaseDone || m.action.phase == PhaseError {
				help = styleKey.Render("enter") + " voltar"
			}
		}
	default:
		if m.loading {
			b.WriteString("\n")
			b.WriteString(styleHint.Render("  " + m.status))
		} else if m.err != nil {
			b.WriteString("\n")
			b.WriteString(styleError.Render("  ✗ " + m.err.Error()))
		} else {
			b.WriteString(renderDashboard(m.snapshot))
		}
	}

	b.WriteString(renderStatusBar(m.width, m.status, help))
	return b.String()
}

func renderDashboard(snap *app.WorkspaceSnapshot) string {
	if snap == nil || snap.Overview == nil {
		return ""
	}
	o := snap.Overview
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styleHeader.Render(fmt.Sprintf("  %s · %s", repoName(o), branchLabel(o))))
	if o.Upstream != "" {
		b.WriteString(styleHeader.Render(fmt.Sprintf(" · %s", syncLabel(o.Ahead, o.Behind))))
	}
	b.WriteString("\n")

	if snap.ConfigErr != nil {
		b.WriteString(styleHint.Render("  Config: não configurado — gitai config\n"))
	} else if snap.Config != nil {
		b.WriteString(styleHint.Render(fmt.Sprintf("  Provider: %s · Model: %s\n", snap.Config.Provider, snap.Config.Model)))
	}

	if snap.OpenPR != nil {
		pr := snap.OpenPR
		state := strings.ToLower(pr.State)
		if pr.IsDraft {
			state = "draft"
		}
		b.WriteString(styleHint.Render(fmt.Sprintf("  PR #%d %s (%s)\n", pr.Number, truncate(pr.Title, 50), state)))
	}

	b.WriteString(styleSection.Render("Branches"))
	b.WriteString("\n")
	limit := min(len(o.Branches), 8)
	for _, br := range o.Branches[:limit] {
		marker := " "
		name := br.Name
		if br.Current {
			marker = styleCurrent.Render("*")
			name = styleCurrent.Render(name)
		}
		line := fmt.Sprintf("  %s %s", marker, name)
		if br.Upstream != "" && (br.Ahead > 0 || br.Behind > 0) {
			line += styleHint.Render(fmt.Sprintf(" (↑%d ↓%d)", br.Ahead, br.Behind))
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	if len(o.FileChanges) > 0 {
		b.WriteString(styleSection.Render("Changed files"))
		b.WriteString("\n")
		fileLimit := min(len(o.FileChanges), 12)
		for _, f := range o.FileChanges[:fileLimit] {
			tag := fileStatusStyle(f.Status).Render(statusTag(f.Status))
			stats := ""
			if f.Insertions > 0 || f.Deletions > 0 {
				stats = styleHint.Render(fmt.Sprintf(" +%d -%d", f.Insertions, f.Deletions))
			}
			b.WriteString(fmt.Sprintf("  %s %s%s\n", tag, f.Path, stats))
		}
	}

	if len(o.RecentCommits) > 0 {
		b.WriteString(styleSection.Render("Recent commits"))
		b.WriteString("\n")
		for _, c := range o.RecentCommits {
			b.WriteString(styleHint.Render("  " + c))
			b.WriteString("\n")
		}
	}

	b.WriteString(styleSection.Render("Next steps"))
	b.WriteString("\n")
	for _, step := range snap.NextSteps {
		switch {
		case step.Plain:
			b.WriteString(styleHint.Render("  • " + step.Command))
		case step.Muted && step.Note != "":
			b.WriteString(fmt.Sprintf("  → %s %s\n", styleHint.Render(step.Command), styleHint.Render(step.Note)))
		case step.Muted:
			b.WriteString(fmt.Sprintf("  → %s\n", styleHint.Render(step.Command)))
		case step.Note != "":
			b.WriteString(fmt.Sprintf("  → %s %s\n", styleKey.Render(step.Command), styleHint.Render(step.Note)))
		default:
			b.WriteString(fmt.Sprintf("  → %s\n", styleKey.Render(step.Command)))
		}
	}

	return b.String()
}

func renderStatusBar(width int, left, right string) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return "\n" + styleStatusBar.Width(width).Render(line)
}

func repoName(o *gitpkg.Overview) string {
	if o == nil {
		return ""
	}
	if o.RemoteURL != "" {
		name := o.RemoteURL
		name = strings.TrimSuffix(name, ".git")
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		if i := strings.LastIndex(name, ":"); i >= 0 {
			name = name[i+1:]
		}
		if name != "" {
			return name
		}
	}
	return o.Root
}

func branchLabel(o *gitpkg.Overview) string {
	if o.Detached {
		return "detached HEAD"
	}
	return o.Branch
}

func syncLabel(ahead, behind int) string {
	switch {
	case ahead > 0 && behind > 0:
		return fmt.Sprintf("↑%d ↓%d", ahead, behind)
	case ahead > 0:
		return fmt.Sprintf("↑%d ahead", ahead)
	case behind > 0:
		return fmt.Sprintf("↓%d behind", behind)
	default:
		return "in sync"
	}
}

func statusTag(status string) string {
	switch status {
	case "untracked":
		return "?"
	case "deleted":
		return "D"
	case "new", "staged":
		return "A"
	case "modified", "staged+modified":
		return "M"
	default:
		return "·"
	}
}

func fileStatusStyle(status string) lipgloss.Style {
	switch status {
	case "untracked":
		return styleUntracked
	case "deleted":
		return styleError
	case "new", "staged":
		return styleNew
	case "modified", "staged+modified":
		return styleModified
	default:
		return styleHint
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
