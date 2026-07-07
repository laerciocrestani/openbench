package uiprefs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInteractiveUIEnabled_default(t *testing.T) {
	t.Setenv("GITAI_NO_UI", "")
	t.Setenv("CI", "")

	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("provider: openrouter\napi_key: test-key\nmodel: m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if !InteractiveUIEnabled() {
		t.Fatal("expected interactive UI by default")
	}
}

func TestInteractiveUIEnabled_configOff(t *testing.T) {
	t.Setenv("GITAI_NO_UI", "")
	t.Setenv("CI", "")

	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	yaml := "provider: openrouter\napi_key: test-key\nmodel: m\ninteractive_ui: false\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if InteractiveUIEnabled() {
		t.Fatal("expected interactive UI disabled from config")
	}
}

func TestInteractiveUIEnabled_envOverride(t *testing.T) {
	t.Setenv("GITAI_NO_UI", "1")
	t.Setenv("CI", "")

	if InteractiveUIEnabled() {
		t.Fatal("expected GITAI_NO_UI to disable TUI")
	}
}

func TestColorsEnabled_default(t *testing.T) {
	t.Setenv("NO_COLOR", "")

	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("provider: openrouter\napi_key: test-key\nmodel: m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if !ColorsEnabled() {
		t.Fatal("expected colors enabled by default")
	}
}

func TestColorsEnabled_configOff(t *testing.T) {
	t.Setenv("NO_COLOR", "")

	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	yaml := "provider: openrouter\napi_key: test-key\nmodel: m\nui_color: false\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if ColorsEnabled() {
		t.Fatal("expected colors disabled from config")
	}
}

func TestColorsEnabled_noColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if ColorsEnabled() {
		t.Fatal("expected NO_COLOR to disable colors")
	}
}

func TestAutoRefreshInterval_default(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("provider: openrouter\napi_key: test-key\nmodel: m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if got := AutoRefreshInterval(); got != 5*time.Second {
		t.Fatalf("expected 5s default, got %v", got)
	}
}

func TestAutoRefreshInterval_configOff(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	yaml := "provider: openrouter\napi_key: test-key\nmodel: m\nui_auto_refresh_seconds: 0\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if got := AutoRefreshInterval(); got != 0 {
		t.Fatalf("expected polling disabled, got %v", got)
	}
}

func TestWatchFilesEnabled_configOff(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	yaml := "provider: openrouter\napi_key: test-key\nmodel: m\nui_watch_files: false\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if WatchFilesEnabled() {
		t.Fatal("expected file watcher disabled from config")
	}
}

func TestFontSize_default(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("provider: openrouter\napi_key: test-key\nmodel: m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if got := FontSize(); got != FontNormal {
		t.Fatalf("expected normal font size, got %q", got)
	}
}

func TestFontSize_configLarge(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	yaml := "provider: openrouter\napi_key: test-key\nmodel: m\nui_font_size: large\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	if got := FontSize(); got != FontLarge {
		t.Fatalf("expected large font size, got %q", got)
	}
}

func TestMinTerminalSize_scalesWithFont(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "config.yaml")
	yaml := "provider: openrouter\napi_key: test-key\nmodel: m\nui_font_size: small\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITAI_CONFIG", path)

	w, h := MinTerminalSize()
	if w != 70 || h != 20 {
		t.Fatalf("expected 70x20 for small font, got %dx%d", w, h)
	}
}
