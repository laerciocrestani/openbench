package main

import (
	"embed"
	"log"

	"github.com/laerciocrestani/openbench/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// Tray template icon: black + alpha (macOS tints it to match the menu bar theme).
//
//go:embed build/trayicon-template.png
var trayIconTemplate []byte

func init() {
	application.RegisterEvent[string]("tray:action")
	application.RegisterEvent[desktop.ProjectStatus]("project:status")
	application.RegisterEvent[desktop.Dashboard]("project:dashboard")
	application.RegisterEvent[string]("update:available")
	application.RegisterEvent[UpdateCheckResult]("update:prompt")
	application.RegisterEvent[string]("terminal:data")
	application.RegisterEvent[string]("terminal:exit")
}

func main() {
	svc := &AppService{}

	app := application.New(application.Options{
		Name:        "openbench",
		Description: "Desktop app for AI commits, PRs and Docker workflows",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	svc.setApp(app)
	if err := initUpdater(app); err != nil {
		log.Printf("updater: %v (continuando sem auto-update)", err)
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "openbench",
		Width:  1100,
		Height: 720,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 52,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(14, 16, 22),
		URL:              "/",
	})

	setupSystemTray(app, window, svc)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
