package desktop

import (
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
)

// ChatModelsView lists selectable chat models for the UI.
type ChatModelsView struct {
	Provider     string   `json:"provider"`
	DefaultModel string   `json:"defaultModel"`
	Models       []string `json:"models"`
}

// LoadChatModels returns configured + suggested models for the active provider.
func LoadChatModels() (*ChatModelsView, error) {
	cfg, _, err := config.LoadExisting()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		d := config.Default()
		cfg = &d
	}

	provider := cfg.Provider
	if strings.TrimSpace(string(provider)) == "" {
		provider = config.ProviderOpenRouter
	}

	seen := map[string]bool{}
	models := make([]string, 0, 8)
	add := func(m string) {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			return
		}
		seen[m] = true
		models = append(models, m)
	}

	def := cfg.EffectiveChatModel()
	add(def)
	add(cfg.EffectiveChatFallback())
	add(cfg.Model)
	add(cfg.FallbackModel)
	for _, m := range config.ModelSuggestions(provider) {
		add(m)
	}

	if def == "" && len(models) > 0 {
		def = models[0]
	}

	return &ChatModelsView{
		Provider:     string(provider),
		DefaultModel: def,
		Models:       models,
	}, nil
}
