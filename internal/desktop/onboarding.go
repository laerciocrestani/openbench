package desktop

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// OnboardingIssue is a single setup problem with guidance.
type OnboardingIssue struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Hint     string `json:"hint"`
	Blocking bool   `json:"blocking"`
}

// OnboardingStatus summarizes local prerequisites for Commit/PR.
type OnboardingStatus struct {
	NeedsOnboarding bool              `json:"needsOnboarding"`
	APIKeyOK        bool              `json:"apiKeyOK"`
	APIKeyMasked    string            `json:"apiKeyMasked"`
	Provider        string            `json:"provider"`
	Model           string            `json:"model"`
	ConfigPath      string            `json:"configPath"`
	GhInstalled     bool              `json:"ghInstalled"`
	GhAuthenticated bool              `json:"ghAuthenticated"`
	HasRemote       bool              `json:"hasRemote"`
	RemoteURL       string            `json:"remoteURL"`
	Issues          []OnboardingIssue `json:"issues"`
}

// CheckOnboarding inspects config, gh and optional project remote.
func CheckOnboarding(projectPath string) (*OnboardingStatus, error) {
	st := &OnboardingStatus{Issues: []OnboardingIssue{}}

	cfg, path, err := config.LoadExisting()
	if err != nil {
		return nil, err
	}
	st.ConfigPath = path
	st.Provider = string(cfg.Provider)
	st.Model = cfg.Model

	if env := config.APIKeyFromEnv(); env != "" {
		cfg.APIKey = env
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		st.APIKeyOK = true
		st.APIKeyMasked = config.MaskAPIKey(cfg.APIKey)
	} else {
		st.Issues = append(st.Issues, OnboardingIssue{
			ID:       "api_key",
			Title:    "API key ausente",
			Hint:     "Informe a chave do provider (OpenAI, OpenRouter ou Gemini) abaixo.",
			Blocking: true,
		})
	}

	if _, err := exec.LookPath("gh"); err == nil {
		st.GhInstalled = true
		cmd := exec.Command("gh", "auth", "status")
		if err := cmd.Run(); err == nil {
			st.GhAuthenticated = true
		} else {
			st.Issues = append(st.Issues, OnboardingIssue{
				ID:       "gh_auth",
				Title:    "GitHub CLI sem autenticação",
				Hint:     "No terminal: gh auth login",
				Blocking: true,
			})
		}
	} else {
		st.Issues = append(st.Issues, OnboardingIssue{
			ID:       "gh_install",
			Title:    "GitHub CLI (gh) não encontrado",
			Hint:     "Instale: https://cli.github.com/ — necessário para criar PRs.",
			Blocking: true,
		})
	}

	if strings.TrimSpace(projectPath) != "" {
		repo, err := gitpkg.Open(projectPath)
		if err == nil && repo.IsRepo() == nil {
			if url, err := repo.RemoteOriginURL(); err == nil && strings.TrimSpace(url) != "" {
				st.HasRemote = true
				st.RemoteURL = url
			} else {
				st.Issues = append(st.Issues, OnboardingIssue{
					ID:       "remote",
					Title:    "Remote origin ausente",
					Hint:     "Configure: git remote add origin <url>",
					Blocking: true,
				})
			}
		}
	}

	st.NeedsOnboarding = len(st.Issues) > 0
	return st, nil
}

// SaveAIConfig writes provider/api key/model to the global config file.
func SaveAIConfig(provider, apiKey, model string) error {
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

	provider = strings.TrimSpace(strings.ToLower(provider))
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)

	if provider != "" {
		cfg.Provider = config.Provider(provider)
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if model != "" {
		cfg.Model = model
		// Onboarding sets a single model — use it for chat too when unset.
		if strings.TrimSpace(cfg.ChatModel) == "" {
			cfg.ChatModel = model
		}
	}

	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("api_key é obrigatória")
	}
	if strings.TrimSpace(string(cfg.Provider)) == "" {
		cfg.Provider = config.ProviderOpenRouter
	}
	if strings.TrimSpace(cfg.Model) == "" {
		switch cfg.Provider {
		case config.ProviderGemini:
			cfg.Model = "gemini-2.0-flash"
		case config.ProviderOpenAI:
			cfg.Model = "gpt-4o-mini"
		default:
			cfg.Model = "deepseek/deepseek-chat"
		}
	}

	return config.Save(path, *cfg)
}
