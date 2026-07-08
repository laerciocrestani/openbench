package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/components"
)

type reportPeriod int

const (
	report24h reportPeriod = iota
	report7d
	reportMonth
	reportAll
)

type reportModel struct {
	period   reportPeriod
	viewport viewport.Model
	content  string
	ready    bool
	err      error
}

func newReportModel() reportModel {
	return reportModel{period: report24h}
}

func (m reportModel) opts() app.ReportOptions {
	switch m.period {
	case report7d:
		return app.ReportOptions{Days: 7}
	case reportMonth:
		return app.ReportOptions{Month: true}
	case reportAll:
		return app.ReportOptions{All: true}
	default:
		return app.ReportOptions{}
	}
}

func loadReportCmd(period reportPeriod) tea.Cmd {
	return func() tea.Msg {
		m := reportModel{period: period}
		snap, err := app.LoadUsageReport(m.opts())
		if err != nil {
			return reportLoadedMsg{period: period, err: err}
		}
		return reportLoadedMsg{
			period:  period,
			content: strings.Join(snap.Lines, "\n"),
		}
	}
}

type reportLoadedMsg struct {
	period  reportPeriod
	content string
	err     error
}

func (m *reportModel) SetSize(width, height int) {
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

func (m *reportModel) Load(msg reportLoadedMsg) {
	m.period = msg.period
	m.content = msg.content
	m.err = msg.err
	m.ready = false
}

func (m reportModel) Update(msg tea.Msg) (reportModel, tea.Cmd) {
	if m.err != nil {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m reportModel) View(tick int) string {
	if m.err != nil {
		return styleError.Render("  ✗ " + m.err.Error())
	}

	var b strings.Builder
	b.WriteString(styleSection.Render("AI Usage"))
	b.WriteString("\n")
	b.WriteString(styleHint.Render("  " + periodLabel(m.period)))
	b.WriteString("\n\n")
	if !m.ready {
		b.WriteString(components.RenderSpinnerLine("Loading", tick))
		return b.String()
	}
	b.WriteString(m.viewport.View())
	return b.String()
}

func periodLabel(p reportPeriod) string {
	switch p {
	case report7d:
		return "last 7 days"
	case reportMonth:
		return "current month"
	case reportAll:
		return "all history"
	default:
		return "last 24 hours"
	}
}

func reportHelpLine() string {
	return styleKey.Render("1") + " 24h  " +
		styleKey.Render("2") + " 7d  " +
		styleKey.Render("3") + " month  " +
		styleKey.Render("a") + " all  " +
		styleKey.Render("esc") + " back"
}
