package desktop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/laerciocrestani/openbench/internal/config"
	"gopkg.in/yaml.v3"
)

func TestLoadDetailSettings_defaultsWithoutProject(t *testing.T) {
	view, err := LoadDetailSettings("")
	if err != nil {
		t.Fatal(err)
	}
	if view.HasProject {
		t.Fatal("expected HasProject=false")
	}
	if view.CommitDetail != "standard" || view.PRDetail != "standard" {
		t.Fatalf("defaults = %+v", view)
	}
}

func TestSaveAndLoadDetailSettings(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.APIKey = "sk-test-key-for-detail"
	cfg.Provider = config.ProviderOpenRouter
	cfg.Model = "deepseek/deepseek-chat"
	cfg.CommitDetail = config.DetailMinimal
	cfg.PRDetail = config.DetailThorough

	path := filepath.Join(dir, config.LocalFileName)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	view, err := LoadDetailSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !view.HasProject {
		t.Fatal("expected HasProject")
	}
	if view.CommitDetail != "minimal" || view.PRDetail != "thorough" {
		t.Fatalf("loaded = %+v", view)
	}

	if err := SaveDetailSettings(dir, "thorough", "minimal"); err != nil {
		t.Fatal(err)
	}
	view2, err := LoadDetailSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if view2.CommitDetail != "thorough" || view2.PRDetail != "minimal" {
		t.Fatalf("after save = %+v", view2)
	}
}

func TestSaveDetailSettings_requiresProject(t *testing.T) {
	if err := SaveDetailSettings("", "standard", "standard"); err == nil {
		t.Fatal("expected error without project")
	}
}
