package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
)

type CommitSuggestion struct {
	Type  string   `json:"type"`
	Scope string   `json:"scope"`
	Title string   `json:"title"`
	Body  []string `json:"body"`
	Notes []string `json:"notes"`
}

type Provider interface {
	SuggestCommit(ctx context.Context, diff, diffStat, lang string) (*CommitSuggestion, error)
	SuggestPR(ctx context.Context, diff, branch, base, lang, commitLog string) (*PRSuggestion, error)
	ExplainHealth(ctx context.Context, facts, lang string) (*HealthExplanation, error)
	SuggestCIFix(ctx context.Context, logWindow, lang, branch string) (*CIFixSuggestion, error)
	SuggestDockerFix(ctx context.Context, fc DockerFixContext, lang string) (*DockerFixSuggestion, error)
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

func buildPrompt(diff, diffStat, lang string, detail config.DetailLevel) string {
	detail = config.NormalizeDetailLevel(detail)
	var b strings.Builder
	b.WriteString(`Analise o git diff abaixo e gere uma mensagem de commit no formato Conventional Commits.

Responda SOMENTE com JSON válido, sem markdown, sem explicações:
{
  "type": "fix|feat|refactor|docs|test|chore|perf|ci|build|style",
  "scope": "escopo opcional do módulo",
  "title": "título curto em imperativo",
  "body": ["bullet 1", "bullet 2"],
  "notes": ["opcional — riscos ou sugestão de split"]
}

Regras:
- Idioma: `)
	b.WriteString(lang)
	b.WriteString(`
- type deve ser um dos valores Conventional Commits
- title sem ponto final, máximo 72 caracteres
`)
	b.WriteString(commitDetailRules(detail))
	b.WriteString(`
- se scope não aplicável ou abrange múltiplas áreas, use string vazia
- não invente funcionalidades que não aparecem no diff
- foque no porquê e no impacto, não em linha a linha

Arquivos alterados (git diff --stat):
`)
	b.WriteString(strings.TrimSpace(diffStat))
	b.WriteString(`

Diff:
`)
	b.WriteString(diff)
	return b.String()
}

func commitDetailRules(detail config.DetailLevel) string {
	switch config.NormalizeDetailLevel(detail) {
	case config.DetailMinimal:
		return `- body com 1-3 bullets curtos e de alto nível (visão geral)
- cubra as áreas principais do resumo (--stat); omita microrrefactors
- notes: use [] salvo risco óbvio ou split claramente necessário`
	case config.DetailThorough:
		return `- body com 4-8 bullets densos cobrindo TODAS as áreas alteradas (agrupadas por contexto)
- cada área/arquivo relevante do resumo deve aparecer no body; não omita mudanças
- explique impacto, motivações e interações entre áreas quando aparente no diff
- se houver mudanças não relacionadas, use title amplo e detalhe cada área no body
- se o ideal for commits separados, inclua em notes uma sugestão objetiva de split
- notes: inclua riscos, breaking changes ou follow-ups quando houver indício no diff`
	default:
		return `- body com 2-6 bullets cobrindo TODAS as áreas alteradas (agrupadas por contexto)
- cada área/arquivo relevante do resumo deve aparecer no body; não omita mudanças
- se houver mudanças não relacionadas entre si, use title amplo (sem scope restrito) e detalhe cada área no body
- se o ideal for commits separados, inclua em notes uma sugestão objetiva de split`
	}
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
	if maxBytes <= 0 || len(diff) <= maxBytes {
		return diff
	}
	return diff[:maxBytes] + "\n\n... [diff truncado] ..."
}
