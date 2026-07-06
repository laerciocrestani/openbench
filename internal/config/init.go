package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/laerciocrestani/gitia/internal/ui"
)

func InitInteractive() error {
	sess := ui.New("config", false)
	sess.Header()

	existing, savePath, err := LoadExisting()
	if err != nil {
		return err
	}

	cfg := *existing
	reader := bufio.NewReader(os.Stdin)
	hadConfig := strings.TrimSpace(cfg.APIKey) != ""

	if err := sess.Step("Starting configuration wizard", func() error {
		return nil
	}); err != nil {
		return err
	}

	if hadConfig {
		sess.Info("Configuração atual detectada — Enter mantém cada valor entre colchetes")
	}

	prevProvider := cfg.Provider

	provider, err := promptChoice(sess, reader, "Provider", []string{"openrouter", "openai", "gemini"}, string(cfg.Provider))
	if err != nil {
		return err
	}
	cfg.Provider = Provider(provider)

	apiKey, err := promptKeep(sess, reader, "API Key", MaskAPIKey(cfg.APIKey), cfg.APIKey)
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
	model, err := promptKeep(sess, reader, "Model", modelDefault, modelKeep)
	if err != nil {
		return err
	}
	if model == "" {
		model = modelDefault
	}
	cfg.Model = model

	lang, err := promptKeep(sess, reader, "Idioma das mensagens", cfg.Language, cfg.Language)
	if err != nil {
		return err
	}
	cfg.Language = lang

	base, err := promptKeep(sess, reader, "Branch base", cfg.BaseBranch, cfg.BaseBranch)
	if err != nil {
		return err
	}
	cfg.BaseBranch = base

	coAuthorDefault := cfg.CoAuthor
	if coAuthorDefault == "" {
		coAuthorDefault = "(vazio)"
	}
	coAuthor, err := promptKeep(sess, reader, "Co-author trailer (opcional)", coAuthorDefault, cfg.CoAuthor)
	if err != nil {
		return err
	}
	cfg.CoAuthor = coAuthor

	fmt.Fprintln(os.Stderr)
	sess.Info("Limpar o terminal antes de cada comando deixa só a saída do Gitia visível,")
	sess.Info("sem misturar com histórico anterior no console.")
	clear, err := promptYesNo(sess, reader, "Ativar limpeza do terminal?", cfg.ClearScreen)
	if err != nil {
		return err
	}
	cfg.ClearScreen = clear

	if err := sess.Step("Saving configuration", func() error {
		return Save(savePath, cfg)
	}); err != nil {
		return err
	}

	sess.Detail(savePath)
	sess.Success("Configuration saved ✨")
	return nil
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

func promptChoice(sess *ui.Session, reader *bufio.Reader, label string, options []string, defaultVal string) (string, error) {
	sess.Prompt(fmt.Sprintf("%s (%s) [%s]: ", label, strings.Join(options, ", "), defaultVal))
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultVal, nil
	}
	for _, opt := range options {
		if input == opt {
			return opt, nil
		}
	}
	return "", fmt.Errorf("opção inválida: %q", input)
}

func promptKeep(sess *ui.Session, reader *bufio.Reader, label, displayDefault, current string) (string, error) {
	sess.Prompt(fmt.Sprintf("%s [%s]: ", label, displayDefault))
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return current, nil
	}
	return input, nil
}

func promptYesNo(sess *ui.Session, reader *bufio.Reader, label string, current bool) (bool, error) {
	defaultLabel := "n"
	if current {
		defaultLabel = "s"
	}
	sess.Prompt(fmt.Sprintf("%s (s/n) [%s]: ", label, defaultLabel))
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return current, nil
	}
	switch input {
	case "s", "sim", "y", "yes":
		return true, nil
	case "n", "nao", "não", "no":
		return false, nil
	default:
		return false, fmt.Errorf("resposta inválida: %q (use s ou n)", input)
	}
}
