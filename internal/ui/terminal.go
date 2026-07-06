package ui

import (
	"fmt"
	"os"
)

// ClearScreen limpa o terminal (ANSI). Respeita GITAI_NO_CLEAR e terminais não interativos.
func ClearScreen() {
	if os.Getenv("GITAI_NO_CLEAR") != "" || os.Getenv("GITAI_NO_UI") != "" {
		return
	}
	if os.Getenv("CI") != "" {
		return
	}
	fi, err := os.Stdout.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return
	}
	fmt.Fprint(os.Stdout, "\033[H\033[2J\033[3J")
}
