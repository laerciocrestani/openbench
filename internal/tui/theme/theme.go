package theme

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/laerciocrestani/gitai/internal/uiprefs"
)

// Styles holds lipgloss styles for the TUI.
type Styles struct {
	Title     lipgloss.Style
	Header    lipgloss.Style
	Section   lipgloss.Style
	Current   lipgloss.Style
	Hint      lipgloss.Style
	StatusBar lipgloss.Style
	Error     lipgloss.Style
	Key       lipgloss.Style
	Modified  lipgloss.Style
	New       lipgloss.Style
	Untracked lipgloss.Style
	Yellow    lipgloss.Style
	Warn      lipgloss.Style
	Panel     lipgloss.Style
	PanelTitle lipgloss.Style
	Success   lipgloss.Style
	Info      lipgloss.Style
	Disabled  lipgloss.Style
	Magenta   lipgloss.Style
	RightShade lipgloss.Style
}

var S Styles

func init() {
	Init()
}

// Init rebuilds the global style set (e.g. after color preference changes).
func Init() {
	if !uiprefs.ColorsEnabled() {
		plain := lipgloss.NewStyle()
		bold := lipgloss.NewStyle().Bold(true)
		S = Styles{
			Title:      bold,
			Header:     plain,
			Section:    bold,
			Current:    bold,
			Hint:       plain,
			StatusBar:  lipgloss.NewStyle().Padding(0, 1),
			Error:      bold,
			Key:        bold,
			Modified:   plain,
			New:        plain,
			Untracked:  plain,
			Yellow:     plain,
			Warn:       plain,
			Panel:      plain,
			PanelTitle: bold,
			Success:    plain,
			Info:       plain,
			Disabled:   plain,
			Magenta:    bold,
			RightShade: plain,
		}
		return
	}

	S = Styles{
		Title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		Header:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Section:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		Current:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		Hint:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		StatusBar: lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Background(lipgloss.Color("236")).Padding(0, 1),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		Key:       lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true),
		Modified:  lipgloss.NewStyle().Foreground(lipgloss.Color("213")),
		New:       lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		Untracked: lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Yellow:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Warn:      lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Panel:     lipgloss.NewStyle(),
		PanelTitle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		Info:      lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		Disabled:  lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		Magenta:   lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true),
		RightShade: lipgloss.NewStyle().Background(lipgloss.Color("236")),
	}
}

func Plain() bool {
	return !uiprefs.ColorsEnabled()
}
