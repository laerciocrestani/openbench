package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
)

// DetailSettingsView is the Settings payload for commit/PR detail levels.
type DetailSettingsView struct {
	CommitDetail string `json:"commitDetail"`
	PRDetail     string `json:"prDetail"`
	ConfigPath   string `json:"configPath"`
	HasProject   bool   `json:"hasProject"`
}

// LoadDetailSettings reads commit_detail / pr_detail for the project (or defaults).
func LoadDetailSettings(projectPath string) (*DetailSettingsView, error) {
	projectPath = strings.TrimSpace(projectPath)
	view := &DetailSettingsView{
		CommitDetail: string(config.DetailStandard),
		PRDetail:     string(config.DetailStandard),
		HasProject:   projectPath != "",
	}
	if projectPath == "" {
		return view, nil
	}

	localPath := config.LocalConfigPathIn(projectPath)
	view.ConfigPath = localPath

	if _, err := os.Stat(localPath); err != nil {
		return view, nil
	}

	cfg, path, err := config.LoadExistingForDir(projectPath)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		view.CommitDetail = string(cfg.EffectiveCommitDetail())
		view.PRDetail = string(cfg.EffectivePRDetail())
	}
	if path != "" {
		view.ConfigPath = path
	}
	return view, nil
}

// SaveDetailSettings persists commit/PR detail levels to the project's .openbench.yaml.
// Creates the local file from the effective config when missing.
func SaveDetailSettings(projectPath, commitDetail, prDetail string) error {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return fmt.Errorf("abra um projeto para salvar níveis de detalhe")
	}

	commitLevel, err := config.ParseDetailLevel(commitDetail)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	prLevel, err := config.ParseDetailLevel(prDetail)
	if err != nil {
		return fmt.Errorf("pr: %w", err)
	}

	cfg, _, err := config.LoadExistingForDir(projectPath)
	if err != nil {
		return err
	}
	if cfg == nil {
		d := config.Default()
		cfg = &d
	}

	cfg.CommitDetail = commitLevel
	cfg.PRDetail = prLevel

	if env := config.APIKeyFromEnv(); env != "" && strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = env
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("api_key é obrigatória para gravar .openbench.yaml — configure a aba IA primeiro")
	}
	if strings.TrimSpace(string(cfg.Provider)) == "" {
		cfg.Provider = config.ProviderOpenRouter
	}
	if strings.TrimSpace(cfg.Model) == "" {
		switch cfg.Provider {
		case config.ProviderGemini:
			cfg.Model = "gemini-2.5-flash-lite"
		case config.ProviderOpenAI:
			cfg.Model = "gpt-4o-mini"
		default:
			cfg.Model = "deepseek/deepseek-chat"
		}
	}

	savePath := config.LocalConfigPathIn(projectPath)
	if savePath == "" {
		savePath = filepath.Join(projectPath, config.LocalFileName)
	}
	return config.Save(savePath, *cfg)
}
