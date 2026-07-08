package tui

// Screen identifica a tela ativa na TUI.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenDiff
	ScreenLogs
	ScreenBranches
	ScreenAction
	ScreenReport
	ScreenHelp
)
