package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func InitInteractive() error {
	cfg := Default()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Configuração do gitia")
	fmt.Println()

	provider, err := promptChoice(reader, "Provider", []string{"openrouter", "openai", "gemini"}, string(cfg.Provider))
	if err != nil {
		return err
	}
	cfg.Provider = Provider(provider)

	fmt.Print("API Key: ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	cfg.APIKey = strings.TrimSpace(apiKey)

	defaultModel := defaultModelFor(cfg.Provider)
	fmt.Printf("Model [%s]: ", defaultModel)
	model, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}
	cfg.Model = model

	fmt.Printf("Idioma das mensagens [%s]: ", cfg.Language)
	lang, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	lang = strings.TrimSpace(lang)
	if lang != "" {
		cfg.Language = lang
	}

	fmt.Printf("Branch base [%s]: ", cfg.BaseBranch)
	base, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	base = strings.TrimSpace(base)
	if base != "" {
		cfg.BaseBranch = base
	}

	fmt.Print("Co-author trailer (opcional, ex: Co-authored-by: Name <email>): ")
	coAuthor, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	cfg.CoAuthor = strings.TrimSpace(coAuthor)

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := Save(path, cfg); err != nil {
		return err
	}

	fmt.Printf("\nConfig salva em %s\n", path)
	return nil
}

func defaultModelFor(p Provider) string {
	switch p {
	case ProviderOpenAI:
		return "gpt-4o-mini"
	case ProviderGemini:
		return "gemini-2.0-flash"
	default:
		return "deepseek/deepseek-chat"
	}
}

func promptChoice(reader *bufio.Reader, label string, options []string, defaultVal string) (string, error) {
	fmt.Printf("%s (%s) [%s]: ", label, strings.Join(options, ", "), defaultVal)
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
