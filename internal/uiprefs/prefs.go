package uiprefs

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultAutoRefreshSeconds = 5
)

type filePrefs struct {
	InteractiveUI        *bool `yaml:"interactive_ui"`
	UIColor              *bool `yaml:"ui_color"`
	UIAutoRefreshSeconds *int  `yaml:"ui_auto_refresh_seconds"`
	UIWatchFiles         *bool `yaml:"ui_watch_files"`
}

// InteractiveUIEnabled indica se `gitai` sem subcomando deve abrir a TUI.
// GITAI_NO_UI=1 força overview CLI. Padrão: true (config interactive_ui).
func InteractiveUIEnabled() bool {
	if os.Getenv("GITAI_NO_UI") != "" || os.Getenv("CI") != "" {
		return false
	}
	prefs := loadPrefs()
	if prefs.InteractiveUI == nil {
		return true
	}
	return *prefs.InteractiveUI
}

// AutoRefreshInterval retorna o intervalo de polling do dashboard TUI.
// 0 desliga o polling (watcher fsnotify continua se ui_watch_files for true).
func AutoRefreshInterval() time.Duration {
	secs := defaultAutoRefreshSeconds
	prefs := loadPrefs()
	if prefs.UIAutoRefreshSeconds != nil {
		secs = *prefs.UIAutoRefreshSeconds
	}
	if secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// WatchFilesEnabled indica se a TUI observa mudanças no filesystem (fsnotify).
func WatchFilesEnabled() bool {
	prefs := loadPrefs()
	if prefs.UIWatchFiles == nil {
		return true
	}
	return *prefs.UIWatchFiles
}

// ColorsEnabled indica se cores ANSI/lipgloss estão ativas.
// NO_COLOR=1 (convenção Unix) força sem cores. Padrão: true (config ui_color).
func ColorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	prefs := loadPrefs()
	if prefs.UIColor == nil {
		return true
	}
	return *prefs.UIColor
}

func loadPrefs() filePrefs {
	var merged filePrefs
	for _, path := range configPaths() {
		p, ok, err := readPrefsFile(path)
		if err != nil || !ok {
			continue
		}
		if p.InteractiveUI != nil {
			merged.InteractiveUI = p.InteractiveUI
		}
		if p.UIColor != nil {
			merged.UIColor = p.UIColor
		}
		if p.UIAutoRefreshSeconds != nil {
			merged.UIAutoRefreshSeconds = p.UIAutoRefreshSeconds
		}
		if p.UIWatchFiles != nil {
			merged.UIWatchFiles = p.UIWatchFiles
		}
	}
	return merged
}

func readPrefsFile(path string) (filePrefs, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return filePrefs{}, false, nil
		}
		return filePrefs{}, false, err
	}
	var p filePrefs
	if err := yaml.Unmarshal(data, &p); err != nil {
		return filePrefs{}, false, err
	}
	return p, true, nil
}

func configPaths() []string {
	if local := localConfigPath(); local != "" {
		if _, err := os.Stat(local); err == nil {
			return []string{local}
		}
	}
	if env := os.Getenv("GITAI_CONFIG"); env != "" {
		return []string{env}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".config", "gitai", "config.yaml")}
}

func localConfigPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(wd, ".gitai.yaml")
}
