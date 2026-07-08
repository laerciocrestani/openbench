package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Run inicia a TUI fullscreen. Respeita GITAI_NO_UI e terminais não interativos.
func Run() error {
	if !ShouldLaunch() {
		return fmt.Errorf("TUI unavailable (not an interactive terminal or GITAI_NO_UI/CI)")
	}

	initTheme()
	cfg := loadRefreshConfig()
	m := newApp(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	var watcher *repoWatcher
	if cfg.watchFiles {
		if root, err := repoRoot(); err == nil {
			watcher, _ = startRepoWatcher(p, root)
		}
	}
	if watcher != nil {
		defer watcher.Close()
	}

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
