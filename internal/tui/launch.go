package tui

import (
	"os"

	"github.com/laerciocrestani/gitai/internal/uiprefs"
)

// ShouldLaunch indica se o comando padrão deve abrir a TUI.
func ShouldLaunch() bool {
	if !uiprefs.InteractiveUIEnabled() {
		return false
	}
	return isTerminal(os.Stdout) && isTerminal(os.Stdin)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func terminalTooSmall(width, height int) bool {
	minW, minH := uiprefs.MinTerminalSize()
	return width < minW || height < minH
}

func terminalMinSize() (int, int) {
	return uiprefs.MinTerminalSize()
}
