package main

import (
	"path/filepath"
	"runtime"

	"github.com/laerciocrestani/openbench/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func setupSystemTray(app *application.App, window *application.WebviewWindow, svc *AppService) {
	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		// Template icon: macOS paints it white/black to match the menu bar theme.
		tray.SetTemplateIcon(trayIconTemplate)
	} else {
		tray.SetIcon(appIcon)
	}
	tray.SetTooltip("openbench")
	tray.SetLabel("") // icon only — no text beside the tray icon

	tray.OnClick(func() {
		toggleWindow(window)
	})

	rebuild := func() {
		rebuildTrayMenu(app, tray, window, svc)
	}
	svc.setTrayRefresh(rebuild)
	rebuild()
}

func rebuildTrayMenu(app *application.App, tray *application.SystemTray, window *application.WebviewWindow, svc *AppService) {
	menu := app.NewMenu()
	activePath := svc.currentPath()
	hasProject := activePath != ""

	// Status line (menu only — tray stays icon-only)
	statusLabel := "Nenhum projeto aberto"
	if hasProject {
		st := desktop.LoadProjectStatus(activePath, false)
		name := st.RepoName
		if name == "" {
			name = filepath.Base(activePath)
		}
		statusLabel = name
		if st.Branch != "" {
			statusLabel += " · " + st.Branch
		}
		if st.Dirty {
			statusLabel += " · dirty"
		}
		tray.SetTooltip("openbench — " + statusLabel)
	} else {
		tray.SetTooltip("openbench")
	}
	tray.SetLabel("")

	statusItem := menu.Add(statusLabel)
	statusItem.SetEnabled(false)

	menu.AddSeparator()

	menu.Add("Abrir openbench").OnClick(func(ctx *application.Context) {
		showWindow(window)
	})

	menu.Add("Open project…").OnClick(func(ctx *application.Context) {
		showWindow(window)
		app.Event.Emit("tray:action", "open-project")
	})

	menu.AddSeparator()

	commitItem := menu.Add("Commit…")
	commitItem.SetEnabled(hasProject)
	commitItem.OnClick(func(ctx *application.Context) {
		showWindow(window)
		app.Event.Emit("tray:action", "commit")
	})

	prItem := menu.Add("PR…")
	prItem.SetEnabled(hasProject)
	prItem.OnClick(func(ctx *application.Context) {
		showWindow(window)
		app.Event.Emit("tray:action", "pr")
	})

	if showDocker, canUp, canDown := svc.dockerTrayFlags(); showDocker {
		menu.AddSeparator()
		if canUp {
			menu.Add("Docker Up").OnClick(func(ctx *application.Context) {
				showWindow(window)
				app.Event.Emit("tray:action", "docker-up")
			})
		}
		if canDown {
			menu.Add("Docker Down").OnClick(func(ctx *application.Context) {
				showWindow(window)
				app.Event.Emit("tray:action", "docker-down")
			})
		}
		if !canUp && !canDown {
			menu.Add("Docker…").OnClick(func(ctx *application.Context) {
				showWindow(window)
				app.Event.Emit("tray:action", "docker-panel")
			})
		}
	}

	if pinned := svc.pinnedTrayEntries(); len(pinned) > 0 {
		menu.AddSeparator()
		sub := menu.AddSubmenu("Projetos")
		for _, p := range pinned {
			path := p.Path
			label := p.Label
			if desktop.SamePath(path, activePath) {
				label = "✓ " + label
			}
			sub.Add(label).OnClick(func(ctx *application.Context) {
				showWindow(window)
				app.Event.Emit("tray:action", "switch:"+path)
			})
		}
	}

	menu.AddSeparator()

	menu.Add("Settings…").OnClick(func(ctx *application.Context) {
		showWindow(window)
		app.Event.Emit("tray:action", "settings")
	})
	menu.Add("Setup…").OnClick(func(ctx *application.Context) {
		showWindow(window)
		app.Event.Emit("tray:action", "setup")
	})

	menu.AddSeparator()

	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	tray.SetMenu(menu)
}

type trayPinned struct {
	Path  string
	Label string
}

func (s *AppService) pinnedTrayEntries() []trayPinned {
	prefs, err := desktop.LoadPrefs()
	if err != nil {
		return nil
	}
	out := make([]trayPinned, 0, len(prefs.Pinned))
	for _, p := range prefs.Pinned {
		label := p.Alias
		if label == "" {
			label = filepath.Base(p.Path)
		}
		out = append(out, trayPinned{Path: p.Path, Label: label})
	}
	return out
}

func (s *AppService) dockerTrayFlags() (show, canUp, canDown bool) {
	path := s.currentPath()
	if path == "" {
		return false, false, false
	}
	st, err := desktop.LoadDockerStatus(path)
	if err != nil || !st.Visible {
		return false, false, false
	}
	show = true
	canUp = st.Available && st.DaemonRunning && st.ComposeFile != ""
	canDown = canUp && st.Running > 0
	return show, canUp, canDown
}

func toggleWindow(window *application.WebviewWindow) {
	if window.IsVisible() {
		window.Hide()
		return
	}
	showWindow(window)
}

func showWindow(window *application.WebviewWindow) {
	window.Show()
	window.Focus()
}
