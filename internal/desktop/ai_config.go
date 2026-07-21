package desktop

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
)

// AIConfigView is the settings "IA" tab payload.
type AIConfigView struct {
	Provider         string   `json:"provider"`
	APIKeyMasked     string   `json:"apiKeyMasked"`
	HasAPIKey        bool     `json:"hasAPIKey"`
	ConfigPath       string   `json:"configPath"`
	GitModel         string   `json:"gitModel"`
	GitFallback      string   `json:"gitFallback"`
	ChatModel        string   `json:"chatModel"`
	ChatFallback     string   `json:"chatFallback"`
	ModelSuggestions []string `json:"modelSuggestions"`
}

// LoadAIConfig reads provider/models for the settings UI (does not require api_key).
func LoadAIConfig() (*AIConfigView, error) {
	cfg, path, err := config.LoadExisting()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		d := config.Default()
		cfg = &d
	}

	if env := config.APIKeyFromEnv(); env != "" {
		cfg.APIKey = env
	}

	provider := cfg.Provider
	if strings.TrimSpace(string(provider)) == "" {
		provider = config.ProviderOpenRouter
	}

	view := &AIConfigView{
		Provider:         string(provider),
		ConfigPath:       path,
		GitModel:         strings.TrimSpace(cfg.Model),
		GitFallback:      strings.TrimSpace(cfg.FallbackModel),
		ChatModel:        strings.TrimSpace(cfg.ChatModel),
		ChatFallback:     strings.TrimSpace(cfg.ChatFallbackModel),
		ModelSuggestions: config.ModelSuggestions(provider),
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		view.HasAPIKey = true
		view.APIKeyMasked = config.MaskAPIKey(cfg.APIKey)
	}
	// Show effective chat model in the field when chat_model is empty (inherits git).
	if view.ChatModel == "" {
		view.ChatModel = view.GitModel
	}
	return view, nil
}

// SaveAISettings persists provider, optional API key, and Chat/Git models.
// Empty apiKey keeps the existing key.
func SaveAISettings(
	provider, apiKey, gitModel, gitFallback, chatModel, chatFallback string,
) error {
	cfg, path, err := config.LoadExisting()
	if err != nil {
		return err
	}
	if path == "" {
		path, err = config.ConfigPath()
		if err != nil {
			return err
		}
	}
	if cfg == nil {
		d := config.Default()
		cfg = &d
	}

	provider = strings.TrimSpace(strings.ToLower(provider))
	apiKey = strings.TrimSpace(apiKey)
	gitModel = strings.TrimSpace(gitModel)
	gitFallback = strings.TrimSpace(gitFallback)
	chatModel = strings.TrimSpace(chatModel)
	chatFallback = strings.TrimSpace(chatFallback)

	if provider != "" {
		cfg.Provider = config.Provider(provider)
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if gitModel != "" {
		cfg.Model = gitModel
	}
	cfg.FallbackModel = gitFallback

	// If chat equals git primary, store empty chat_model so it keeps inheriting.
	if chatModel == "" || chatModel == cfg.Model {
		cfg.ChatModel = ""
	} else {
		cfg.ChatModel = chatModel
	}
	cfg.ChatFallbackModel = chatFallback

	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("api_key é obrigatória")
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

	return config.Save(path, *cfg)
}
