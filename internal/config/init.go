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

	existing, savePath, err := LoadExisting()
	if err != nil {
		return err
	}

	cfg := *existing
	reader := ui.StdinReader()
	hadConfig := hasSavedConfig(existing)

	intro := "Configure provedor, API, modelo principal e fallback"
	if hadConfig {
		intro = "Configuração atual detectada — Enter mantém cada valor entre colchetes"
	}
	wiz := ui.NewWizard(sess, "Configuração", intro)

	prevProvider := cfg.Provider

	provider, err := wiz.Select(reader, ui.SelectConfig{
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

	apiKey, err := promptAPIKey(wiz, reader, cfg.Provider, cfg.APIKey)
	if err != nil {
		return err
	}
	cfg.APIKey = apiKey

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

	model, err := wiz.Select(reader, ui.SelectConfig{
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

	fallbackDefault := cfg.FallbackModel
	if fallbackDefault == "" {
		fallbackDefault = defaultFallbackFor(cfg.Provider, cfg.Model)
	}
	fallbackOptions := fallbackSuggestions(cfg.Provider, cfg.Model)
	if fallbackDefault != "" && fallbackDefault != "(nenhum)" && !slices.Contains(fallbackOptions, fallbackDefault) {
		fallbackOptions = append([]string{fallbackDefault}, fallbackOptions...)
	}

	fallbackChoice, err := wiz.Select(reader, ui.SelectConfig{
		Label:      "Modelo fallback",
		Options:    append([]string{"(nenhum)"}, fallbackOptions...),
		Default:    fallbackDefault,
		AllowOther: true,
	})
	if err != nil {
		return err
	}
	if fallbackChoice == "(nenhum)" {
		cfg.FallbackModel = ""
	} else {
		cfg.FallbackModel = fallbackChoice
	}

	wiz.AddSection("Preferências")

	langDefault := cfg.Language
	if langDefault == "" {
		langDefault = "pt-BR"
	}
	lang, err := wiz.Select(reader, ui.SelectConfig{
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
	base, err := wiz.Select(reader, ui.SelectConfig{
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
	coAuthorChoice, err := wiz.Select(reader, ui.SelectConfig{
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
	clearChoice, err := wiz.Select(reader, ui.SelectConfig{
		Label:      "Limpar terminal antes de cada comando",
		Options:    []string{"Sim", "Não"},
		Default:    clearDefault,
		AllowOther: false,
	})
	if err != nil {
		return err
	}
	cfg.ClearScreen = clearChoice == "Sim"

	interactiveDefault := "Sim"
	if !cfg.InteractiveUI {
		interactiveDefault = "Não"
	}
	interactiveChoice, err := wiz.Select(reader, ui.SelectConfig{
		Label:      "Interface interativa ao rodar gitai",
		Options:    []string{"Sim", "Não"},
		Default:    interactiveDefault,
		AllowOther: false,
	})
	if err != nil {
		return err
	}
	cfg.InteractiveUI = interactiveChoice == "Sim"

	colorDefault := "Sim"
	if !cfg.UIColor {
		colorDefault = "Não"
	}
	colorChoice, err := wiz.Select(reader, ui.SelectConfig{
		Label:      "Cores na interface (CLI e TUI)",
		Options:    []string{"Sim", "Não"},
		Default:    colorDefault,
		AllowOther: false,
	})
	if err != nil {
		return err
	}
	cfg.UIColor = colorChoice == "Sim"

	fontDefault := fontSizeLabel(cfg.UIFontSize)
	fontChoice, err := wiz.Select(reader, ui.SelectConfig{
		Label:      "Tamanho da fonte na interface",
		Options:    []string{"Pequeno", "Normal", "Grande"},
		Default:    fontDefault,
		AllowOther: false,
	})
	if err != nil {
		return err
	}
	cfg.UIFontSize = fontSizeValue(fontChoice)

	if strings.TrimSpace(cfg.APIKey) == "" && strings.TrimSpace(os.Getenv(EnvAPIKey)) == "" {
		return fmt.Errorf("chave API obrigatória — defina no wizard ou na variável %s", EnvAPIKey)
	}

	if err := Save(savePath, cfg); err != nil {
		return err
	}

	wiz.Record("Salvo em", savePath)
	wiz.Finish()
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

func fontSizeLabel(size string) string {
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "small", "pequeno":
		return "Pequeno"
	case "large", "grande":
		return "Grande"
	default:
		return "Normal"
	}
}

func fontSizeValue(label string) string {
	switch label {
	case "Pequeno":
		return "small"
	case "Grande":
		return "large"
	default:
		return "normal"
	}
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

func defaultFallbackFor(p Provider, primary string) string {
	switch p {
	case ProviderGemini:
		switch primary {
		case "gemini-2.5-flash-lite":
			return "gemini-2.5-flash"
		case "gemini-2.5-flash":
			return "gemini-3.1-flash-lite"
		default:
			return "gemini-3.1-flash-lite"
		}
	case ProviderOpenAI:
		return "gpt-4o-mini"
	case ProviderOpenRouter:
		return "deepseek/deepseek-chat"
	default:
		return ""
	}
}

func modelSuggestions(p Provider) []string {
	switch p {
	case ProviderOpenAI:
		return []string{"gpt-4o-mini", "gpt-4o", "gpt-4.1-mini"}
	case ProviderGemini:
		return []string{"gemini-2.5-flash-lite", "gemini-3.1-flash-lite", "gemini-2.5-flash"}
	default:
		return []string{"deepseek/deepseek-chat", "anthropic/claude-sonnet-4", "google/gemini-2.5-flash-lite"}
	}
}

func fallbackSuggestions(p Provider, primary string) []string {
	var opts []string
	switch p {
	case ProviderOpenAI:
		opts = []string{"gpt-4o-mini", "gpt-4.1-mini"}
	case ProviderGemini:
		opts = []string{"gemini-2.5-flash", "gemini-3.1-flash-lite", "gemini-3.5-flash"}
	default:
		opts = []string{"deepseek/deepseek-chat", "google/gemini-2.0-flash"}
	}
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		if o != primary {
			out = append(out, o)
		}
	}
	return out
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

func promptAPIKey(wiz *ui.Wizard, reader *bufio.Reader, provider Provider, current string) (string, error) {
	current = strings.TrimSpace(current)

	if current != "" {
		keepLabel := fmt.Sprintf("Manter atual (%s)", MaskAPIKey(current))
		choice, err := wiz.Select(reader, ui.SelectConfig{
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
		wiz.UndoLast()
	}

	key, err := wiz.Ask(reader, "Chave API", "Chave em "+apiKeyHint(provider))
	if err != nil {
		return "", err
	}
	if key == "" && current != "" {
		return current, nil
	}
	if key == "" {
		return "", fmt.Errorf("chave API obrigatória")
	}
	display := MaskAPIKey(key)
	if display == "****" {
		display = "informada"
	}
	wiz.Record("Chave API", display)
	return key, nil
}
