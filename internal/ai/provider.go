package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/config"
)

type CommitSuggestion struct {
	Type  string   `json:"type"`
	Scope string   `json:"scope"`
	Title string   `json:"title"`
	Body  []string `json:"body"`
}

type Provider interface {
	SuggestCommit(ctx context.Context, diff string, lang string) (*CommitSuggestion, error)
	SuggestPR(ctx context.Context, diff, branch, base, lang, commitLog string) (*PRSuggestion, error)
	UsageStats() UsageSummary
}

func New(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case config.ProviderOpenAI:
		return NewOpenAI(cfg, "https://api.openai.com/v1/chat/completions"), nil
	case config.ProviderOpenRouter:
		return NewOpenAI(cfg, "https://openrouter.ai/api/v1/chat/completions"), nil
	case config.ProviderGemini:
		return NewGemini(cfg), nil
	default:
		return nil, fmt.Errorf("provider desconhecido: %s", cfg.Provider)
	}
}

func buildPrompt(diff string, lang string) string {
	return fmt.Sprintf(`Analise o git diff abaixo e gere uma mensagem de commit no formato Conventional Commits.

Responda SOMENTE com JSON válido, sem markdown, sem explicações:
{
  "type": "fix|feat|refactor|docs|test|chore|perf|ci|build|style",
  "scope": "escopo opcional do módulo",
  "title": "título curto em imperativo",
  "body": ["bullet 1", "bullet 2"]
}

Regras:
- Idioma: %s
- type deve ser um dos valores Conventional Commits
- title sem ponto final, máximo 72 caracteres
- body com 1-4 bullets explicando o porquê, não o quê linha a linha
- se scope não aplicável, use string vazia

Diff:
%s`, lang, diff)
}

func parseSuggestion(raw string) (*CommitSuggestion, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var suggestion CommitSuggestion
	if err := json.Unmarshal([]byte(raw), &suggestion); err != nil {
		return nil, fmt.Errorf("parse JSON da IA: %w\nresposta: %s", err, raw)
	}

	suggestion.Type = strings.TrimSpace(suggestion.Type)
	suggestion.Scope = strings.TrimSpace(suggestion.Scope)
	suggestion.Title = strings.TrimSpace(suggestion.Title)

	if suggestion.Type == "" || suggestion.Title == "" {
		return nil, fmt.Errorf("resposta da IA incompleta: type e title são obrigatórios")
	}

	return &suggestion, nil
}

func truncateDiff(diff string, maxBytes int) string {
	if len(diff) <= maxBytes {
		return diff
	}
	return diff[:maxBytes] + "\n\n... [diff truncado] ..."
}
