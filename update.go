package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/laerciocrestani/openbench/internal/version"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

//go:embed build/updater/updater.key.pub
var updaterPublicKey []byte

const (
	githubUpdaterRepo = "laerciocrestani/openbench"
	// Signed Wails update manifest published as a GitHub Release asset.
	updaterManifestURL = "https://github.com/laerciocrestani/openbench/releases/latest/download/manifest.json"
)

// UpdateCheckResult is returned to the Settings UI and update:prompt dialog.
type UpdateCheckResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Available      bool   `json:"available"`
	Name           string `json:"name"`
	Notes          string `json:"notes"`
	Message        string `json:"message"`
}

func appSemver() string {
	info, err := version.Compute(".")
	if err != nil {
		return version.DefaultBase
	}
	return strings.TrimPrefix(info.Version, "v")
}

func initUpdater(app *application.App) error {
	// Prefer signed manifest (digest + ed25519ph). Fall back to GitHub asset
	// matching when the manifest asset is missing (first boot / older releases).
	ep, err := endpoint.New(endpoint.Config{
		URL:     updaterManifestURL,
		Channel: "stable",
	})
	if err != nil {
		return fmt.Errorf("endpoint updater provider: %w", err)
	}
	gh, err := github.New(github.Config{
		Repository: githubUpdaterRepo,
	})
	if err != nil {
		return fmt.Errorf("github updater provider: %w", err)
	}

	cfg := updater.Config{
		CurrentVersion: appSemver(),
		Providers:      []updater.Provider{ep, gh},
		PublicKey:      updaterPublicKey,
		CheckInterval:  6 * time.Hour,
	}
	if err := app.Updater.Init(cfg); err != nil {
		return err
	}

	app.Event.On(updater.EventUpdateAvailable, func(e *application.CustomEvent) {
		rel, ok := e.Data.(*updater.Release)
		if !ok || rel == nil {
			return
		}
		log.Printf("update available: %s", rel.Version)
		current := app.Updater.CurrentVersion()
		if current == "" {
			current = appSemver()
		}
		app.Event.Emit("update:available", rel.Version)
		app.Event.Emit("update:prompt", UpdateCheckResult{
			CurrentVersion: current,
			LatestVersion:  rel.Version,
			Available:      true,
			Name:           rel.Name,
			Notes:          rel.Notes,
			Message:        fmt.Sprintf("Nova versão disponível: %s", rel.Version),
		})
	})
	app.Event.On(updater.EventError, func(e *application.CustomEvent) {
		info, ok := e.Data.(updater.ErrorInfo)
		if !ok {
			return
		}
		log.Printf("updater error (%s): %s", info.Stage, info.Message)
	})

	return nil
}

// CheckForUpdates looks for a newer GitHub release (does not install).
func (s *AppService) CheckForUpdates() (*UpdateCheckResult, error) {
	s.mu.RLock()
	app := s.app
	s.mu.RUnlock()
	if app == nil || app.Updater == nil {
		return nil, fmt.Errorf("updater not ready")
	}

	current := app.Updater.CurrentVersion()
	if current == "" {
		current = appSemver()
	}

	rel, err := app.Updater.Check(context.Background())
	if err != nil {
		return &UpdateCheckResult{
			CurrentVersion: current,
			Message:        err.Error(),
		}, err
	}
	if rel == nil {
		return &UpdateCheckResult{
			CurrentVersion: current,
			LatestVersion:  current,
			Available:      false,
			Message:        "Você já está na versão mais recente.",
		}, nil
	}

	return &UpdateCheckResult{
		CurrentVersion: current,
		LatestVersion:  rel.Version,
		Available:      true,
		Name:           rel.Name,
		Notes:          rel.Notes,
		Message:        fmt.Sprintf("Nova versão disponível: %s", rel.Version),
	}, nil
}

// InstallUpdate downloads and stages the latest update after Check.
func (s *AppService) InstallUpdate() (*UpdateCheckResult, error) {
	s.mu.RLock()
	app := s.app
	s.mu.RUnlock()
	if app == nil || app.Updater == nil {
		return nil, fmt.Errorf("updater not ready")
	}

	current := app.Updater.CurrentVersion()
	if err := app.Updater.DownloadAndInstall(context.Background()); err != nil {
		return &UpdateCheckResult{
			CurrentVersion: current,
			Message:        err.Error(),
		}, err
	}
	return &UpdateCheckResult{
		CurrentVersion: current,
		Available:      true,
		Message:        "Atualização baixada. Reinicie o app para aplicar.",
	}, nil
}

// RestartAfterUpdate swaps in the downloaded update and relaunches.
func (s *AppService) RestartAfterUpdate() error {
	s.mu.RLock()
	app := s.app
	s.mu.RUnlock()
	if app == nil || app.Updater == nil {
		return fmt.Errorf("updater not ready")
	}
	return app.Updater.Restart(context.Background())
}
