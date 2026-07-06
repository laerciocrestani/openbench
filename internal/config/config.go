package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	EnvAPIKey = "GITAI_API_KEY"
)

type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderGemini     Provider = "gemini"
	ProviderOpenRouter Provider = "openrouter"
)

type Config struct {
	Provider         Provider `yaml:"provider"`
	APIKey           string   `yaml:"api_key"`
	Model            string   `yaml:"model"`
	FallbackModel    string   `yaml:"fallback_model,omitempty"`
	Language         string   `yaml:"language"`
	BaseBranch       string   `yaml:"base_branch"`
	CoAuthor         string   `yaml:"co_author"`
	MaxDiffBytes     int      `yaml:"max_diff_bytes"`
	InputPricePer1M  float64  `yaml:"input_price_per_1m,omitempty"`
	OutputPricePer1M float64  `yaml:"output_price_per_1m,omitempty"`
	ClearScreen      bool     `yaml:"clear_screen,omitempty"`
	InteractiveUI        bool `yaml:"interactive_ui,omitempty"`
	UIColor              bool `yaml:"ui_color,omitempty"`
	UIAutoRefreshSeconds int  `yaml:"ui_auto_refresh_seconds,omitempty"`
	UIWatchFiles         bool `yaml:"ui_watch_files,omitempty"`
}

func Default() Config {
	return Config{
		Provider:      ProviderOpenRouter,
		Model:         "deepseek/deepseek-chat",
		Language:      "pt-BR",
		BaseBranch:    "main",
		MaxDiffBytes:  120000,
		InteractiveUI:        true,
		UIColor:              true,
		UIAutoRefreshSeconds: 5,
		UIWatchFiles:         true,
	}
}

func ConfigPath() (string, error) {
	if env := os.Getenv("GITAI_CONFIG"); env != "" {
		return env, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gitai", "config.yaml"), nil
}

func LocalConfigPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(wd, ".gitai.yaml")
}

func Load() (*Config, error) {
	cfg := Default()

	localPath := LocalConfigPath()
	if _, err := os.Stat(localPath); err == nil {
		if err := loadFile(localPath, &cfg); err != nil {
			return nil, fmt.Errorf("carregar %s: %w", localPath, err)
		}
	} else {
		path, err := ConfigPath()
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("config não encontrada. Execute: gitai config init")
			}
			return nil, err
		}
		if err := loadFile(path, &cfg); err != nil {
			return nil, fmt.Errorf("carregar %s: %w", path, err)
		}
	}

	if envKey := os.Getenv(EnvAPIKey); envKey != "" {
		cfg.APIKey = envKey
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadExisting lê a config salva sem exigir api_key (para o wizard e preferências).
func LoadExisting() (*Config, string, error) {
	cfg := Default()

	localPath := LocalConfigPath()
	if _, err := os.Stat(localPath); err == nil {
		if err := loadFile(localPath, &cfg); err != nil {
			return nil, "", fmt.Errorf("carregar %s: %w", localPath, err)
		}
		cfg.normalize()
		return &cfg, localPath, nil
	}

	path, err := ConfigPath()
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cfg, path, nil
		}
		return nil, "", err
	}
	if err := loadFile(path, &cfg); err != nil {
		return nil, "", fmt.Errorf("carregar %s: %w", path, err)
	}
	cfg.normalize()
	return &cfg, path, nil
}

// ClearScreenEnabled indica se o terminal deve ser limpo antes de cada comando.
func ClearScreenEnabled() bool {
	if os.Getenv("GITAI_NO_CLEAR") != "" {
		return false
	}
	cfg, _, err := LoadExisting()
	if err != nil {
		return false
	}
	return cfg.ClearScreen
}

func (c *Config) normalize() {
	if c.Language == "" {
		c.Language = "pt-BR"
	}
	if c.BaseBranch == "" {
		c.BaseBranch = "main"
	}
	if c.MaxDiffBytes <= 0 {
		c.MaxDiffBytes = 120000
	}
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

func (c *Config) Validate() error {
	switch c.Provider {
	case ProviderOpenAI, ProviderGemini, ProviderOpenRouter:
	default:
		return fmt.Errorf("provider inválido: %q (use openai, gemini ou openrouter)", c.Provider)
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("api_key não configurada. Defina em config.yaml ou %s", EnvAPIKey)
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("model não configurado")
	}
	if c.Language == "" {
		c.Language = "pt-BR"
	}
	if c.BaseBranch == "" {
		c.BaseBranch = "main"
	}
	if c.MaxDiffBytes <= 0 {
		c.MaxDiffBytes = 120000
	}
	return nil
}

func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (c Config) Display() string {
	c.APIKey = MaskAPIKey(c.APIKey)
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("%+v", c)
	}
	return string(data)
}
