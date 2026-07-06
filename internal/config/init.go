package config

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/laerciocrestani/gitai/internal/ui"
)

func InitInteractive() error {
	sess := ui.New("config", false)
	sess.Header()

	existing, savePath, err := LoadExisting()
	if err != nil {
		return err
	}

	cfg := *existing
	reader := ui.StdinReader()
	hadConfig := hasSavedConfig(existing)

	sess.SectionFirst("Configuração")
	if hadConfig {
		sess.Info("Configuração atual detectada — Enter mantém cada valor entre colchetes")
	} else {
		sess.Info("Configure o provedor e o modelo de IA; a chave API vem em seguida")
	}

	prevProvider := cfg.Provider

	provider, err := sess.Select(reader, ui.SelectConfig{
		Label:      "Provedor",
		Options:    []string{"openrouter", "openai", "gemini"},
		Default:    string(cfg.Provider),
		AllowOther: true,
	})
	if err != nil {
		return err
	}
	p, err := normalizeProvider(provider)
	if err != nil {
		return err
	}
	cfg.Provider = p

	modelDefault := cfg.Model
	if cfg.Provider != prevProvider || modelDefault == "" {
		modelDefault = defaultModelFor(cfg.Provider)
	}
	modelKeep := cfg.Model
	if cfg.Provider != prevProvider {
		modelKeep = modelDefault
	}

	modelOptions := modelSuggestions(cfg.Provider)
	if modelKeep != "" && !slices.Contains(modelOptions, modelKeep) {
		modelOptions = append([]string{modelKeep}, modelOptions...)
	}

	model, err := sess.Select(reader, ui.SelectConfig{
		Label:      "Modelo",
		Options:    modelOptions,
		Default:    modelKeep,
		AllowOther: true,
	})
	if err != nil {
		return err
	}
	if model == "" {
		model = modelDefault
	}
	cfg.Model = model

	apiKey, err := promptAPIKey(sess, reader, cfg.Provider, cfg.APIKey)
	if err != nil {
		return err
	}
	cfg.APIKey = apiKey

	sess.Section("Preferências")

	langDefault := cfg.Language
	if langDefault == "" {
		langDefault = "pt-BR"
	}
	lang, err := sess.Select(reader, ui.SelectConfig{
		Label:      "Idioma das mensagens",
		Options:    []string{"pt-BR", "en-US", "pt", "en"},
		Default:    langDefault,
		AllowOther: true,
	})
	if err != nil {
		return err
	}
	cfg.Language = lang

	baseDefault := cfg.BaseBranch
	if baseDefault == "" {
		baseDefault = "main"
	}
	base, err := sess.Select(reader, ui.SelectConfig{
		Label:      "Branch base",
		Options:    []string{"main", "master", "develop"},
		Default:    baseDefault,
		AllowOther: true,
	})
	if err != nil {
		return err
	}
	cfg.BaseBranch = base

	coAuthorOptions := []string{"(nenhum)"}
	coAuthorDefault := "(nenhum)"
	if strings.TrimSpace(cfg.CoAuthor) != "" {
		coAuthorOptions = append(coAuthorOptions, cfg.CoAuthor)
		coAuthorDefault = cfg.CoAuthor
	}
	coAuthorChoice, err := sess.Select(reader, ui.SelectConfig{
		Label:      "Co-author trailer (opcional)",
		Options:    coAuthorOptions,
		Default:    coAuthorDefault,
		AllowOther: true,
	})
	if err != nil {
		return err
	}
	if coAuthorChoice == "(nenhum)" {
		cfg.CoAuthor = ""
	} else {
		cfg.CoAuthor = coAuthorChoice
	}

	clearDefault := "Não"
	if cfg.ClearScreen {
		clearDefault = "Sim"
	}
	clearChoice, err := sess.Select(reader, ui.SelectConfig{
		Label:      "Limpar terminal antes de cada comando",
		Options:    []string{"Sim", "Não"},
		Default:    clearDefault,
		AllowOther: false,
	})
	if err != nil {
		return err
	}
	cfg.ClearScreen = clearChoice == "Sim"

	if strings.TrimSpace(cfg.APIKey) == "" && strings.TrimSpace(os.Getenv(EnvAPIKey)) == "" {
		return fmt.Errorf("chave API obrigatória — defina no wizard ou na variável %s", EnvAPIKey)
	}

	if err := sess.Step("Saving configuration", func() error {
		return Save(savePath, cfg)
	}); err != nil {
		return err
	}

	sess.Detail(savePath)
	sess.Success("Configuration saved ✨")
	return nil
}

func normalizeProvider(raw string) (Provider, error) {
	p := Provider(strings.ToLower(strings.TrimSpace(raw)))
	switch p {
	case ProviderOpenAI, ProviderGemini, ProviderOpenRouter:
		return p, nil
	default:
		return "", fmt.Errorf("provedor %q inválido — use openrouter, openai ou gemini", raw)
	}
}

func hasSavedConfig(cfg *Config) bool {
	if strings.TrimSpace(cfg.APIKey) != "" {
		return true
	}
	path, err := ConfigPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	localPath := LocalConfigPath()
	if localPath == "" {
		return false
	}
	_, err = os.Stat(localPath)
	return err == nil
}

func defaultModelFor(p Provider) string {
	switch p {
	case ProviderOpenAI:
		return "gpt-4o-mini"
	case ProviderGemini:
		return "gemini-2.5-flash-lite"
	default:
		return "deepseek/deepseek-chat"
	}
}

func modelSuggestions(p Provider) []string {
	switch p {
	case ProviderOpenAI:
		return []string{"gpt-4o-mini", "gpt-4o", "gpt-4.1-mini"}
	case ProviderGemini:
		return []string{"gemini-2.5-flash-lite", "gemini-2.5-flash", "gemini-2.0-flash"}
	default:
		return []string{"deepseek/deepseek-chat", "anthropic/claude-sonnet-4", "google/gemini-2.5-flash-lite"}
	}
}

func apiKeyHint(p Provider) string {
	switch p {
	case ProviderOpenAI:
		return "https://platform.openai.com/api-keys"
	case ProviderGemini:
		return "https://aistudio.google.com/apikey"
	default:
		return "https://openrouter.ai/keys"
	}
}

func promptAPIKey(sess *ui.Session, reader *bufio.Reader, provider Provider, current string) (string, error) {
	current = strings.TrimSpace(current)

	if current != "" {
		keepLabel := fmt.Sprintf("Manter atual (%s)", MaskAPIKey(current))
		choice, err := sess.Select(reader, ui.SelectConfig{
			Label:      "Chave API",
			Options:    []string{keepLabel, "Digitar nova chave"},
			Default:    keepLabel,
			AllowOther: true,
		})
		if err != nil {
			return "", err
		}
		if choice == keepLabel {
			return current, nil
		}
		if choice != "Digitar nova chave" {
			return choice, nil
		}
	}

	sess.Info("Chave em " + apiKeyHint(provider))
	sess.Prompt("Chave API: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" && current != "" {
		return current, nil
	}
	if input == "" {
		return "", fmt.Errorf("chave API obrigatória")
	}
	return input, nil
}
