package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderGemini     Provider = "gemini"
	ProviderOpenRouter Provider = "openrouter"
)

type Config struct {
	Provider             Provider `yaml:"provider"`
	APIKey               string   `yaml:"api_key"`
	Model                string   `yaml:"model"` // Git IA (commit/PR) — primary
	FallbackModel        string   `yaml:"fallback_model,omitempty"`
	ChatModel            string   `yaml:"chat_model,omitempty"`
	ChatFallbackModel    string   `yaml:"chat_fallback_model,omitempty"`
	Language             string   `yaml:"language"`
	BaseBranch           string   `yaml:"base_branch"`
	CoAuthor             string   `yaml:"co_author"`
	MaxDiffBytes         int      `yaml:"max_diff_bytes"`
	InputPricePer1M      float64  `yaml:"input_price_per_1m,omitempty"`
	OutputPricePer1M     float64  `yaml:"output_price_per_1m,omitempty"`
	ClearScreen          bool     `yaml:"clear_screen,omitempty"`
	InteractiveUI        bool     `yaml:"interactive_ui,omitempty"`
	UIColor              bool     `yaml:"ui_color,omitempty"`
	UIAutoRefreshSeconds int      `yaml:"ui_auto_refresh_seconds,omitempty"`
	UIWatchFiles         bool     `yaml:"ui_watch_files,omitempty"`
}

func Default() Config {
	return Config{
		Provider:             ProviderOpenRouter,
		Model:                "deepseek/deepseek-chat",
		Language:             "pt-BR",
		BaseBranch:           "main",
		MaxDiffBytes:         120000,
		InteractiveUI:        true,
		UIColor:              true,
		UIAutoRefreshSeconds: 5,
		UIWatchFiles:         true,
	}
}

func Load() (*Config, error) {
	cfg := Default()

	localPath := LocalConfigPath()
	if fileExists(localPath) {
		if err := loadFile(localPath, &cfg); err != nil {
			return nil, fmt.Errorf("carregar %s: %w", localPath, err)
		}
	} else {
		path, err := ConfigPath()
		if err != nil {
			return nil, err
		}
		if !fileExists(path) {
			return nil, fmt.Errorf("config não encontrada. Execute: ob config init")
		}
		if err := loadFile(path, &cfg); err != nil {
			return nil, fmt.Errorf("carregar %s: %w", path, err)
		}
	}

	if envKey := APIKeyFromEnv(); envKey != "" {
		cfg.APIKey = envKey
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadExisting reads saved config without requiring api_key (wizard and preferences).
func LoadExisting() (*Config, string, error) {
	cfg := Default()

	localPath := LocalConfigPath()
	if fileExists(localPath) {
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
	if !fileExists(path) {
		return &cfg, path, nil
	}
	if err := loadFile(path, &cfg); err != nil {
		return nil, "", fmt.Errorf("carregar %s: %w", path, err)
	}
	cfg.normalize()
	return &cfg, path, nil
}

// ClearScreenEnabled reports whether the terminal should be cleared before each command.
func ClearScreenEnabled() bool {
	if NoClearFromEnv() {
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

// EffectiveChatModel returns the chat primary model, falling back to git model.
func (c *Config) EffectiveChatModel() string {
	if c == nil {
		return ""
	}
	if m := strings.TrimSpace(c.ChatModel); m != "" {
		return m
	}
	return strings.TrimSpace(c.Model)
}

// EffectiveChatFallback returns the chat fallback model (may be empty).
func (c *Config) EffectiveChatFallback() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.ChatFallbackModel)
}

// ApplyChatModels sets Model/FallbackModel from chat-specific fields for a chat request.
func (c *Config) ApplyChatModels() {
	if c == nil {
		return
	}
	if m := c.EffectiveChatModel(); m != "" {
		c.Model = m
	}
	if fb := c.EffectiveChatFallback(); fb != "" {
		c.FallbackModel = fb
	}
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
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
	if c.Provider == ProviderGemini && !isValidGeminiAPIKey(c.APIKey) {
		return fmt.Errorf(
			"chave Gemini inválida — crie uma em https://aistudio.google.com/apikey (formato AIza... ou AQ....)",
		)
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

func isValidGeminiAPIKey(key string) bool {
	key = strings.TrimSpace(key)
	return strings.HasPrefix(key, "AIza") || strings.HasPrefix(key, "AQ.")
}

// EnsureDataDir creates ~/.config/openbench if needed.
func EnsureDataDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
