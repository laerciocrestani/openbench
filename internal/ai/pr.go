package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type PRSuggestion struct {
	Title    string   `json:"title"`
	Summary  []string `json:"summary"`
	Changes  []string `json:"changes"`
	TestPlan []string `json:"test_plan"`
	Notes    []string `json:"notes"`
}

func buildPRPrompt(diff, branch, base, lang, commitLog string) string {
	var b strings.Builder
	b.WriteString(`Analise o git diff abaixo (alterações da branch em relação à base) e gere um Pull Request detalhado para revisão.

Responda SOMENTE com JSON válido, sem markdown, sem explicações:
{
  "title": "título claro e específico do PR",
  "summary": ["visão geral em 2-4 bullets — o porquê e o impacto"],
  "changes": ["detalhe técnico por área/arquivo — o que mudou e como"],
  "test_plan": ["passos concretos para validar as alterações"],
  "notes": ["riscos, breaking changes, migrations ou follow-ups — ou array vazio"]
}

Regras:
- Idioma: `)
	b.WriteString(lang)
	b.WriteString(`
- title: máximo 72 caracteres, sem ponto final
- summary: foco em valor de negócio e motivação
- changes: 3-8 bullets técnicos, agrupados por contexto quando fizer sentido
- test_plan: 3-6 passos acionáveis com verbos no imperativo
- notes: só inclua itens relevantes; use [] se não houver
- Não invente funcionalidades que não aparecem no diff

Branch: `)
	b.WriteString(branch)
	b.WriteString(`
Base: `)
	b.WriteString(base)

	if strings.TrimSpace(commitLog) != "" {
		b.WriteString(`

Commits na branch:
`)
		b.WriteString(commitLog)
	}

	b.WriteString(`

Diff:
`)
	b.WriteString(diff)

	return b.String()
}

func parsePRSuggestion(raw string) (*PRSuggestion, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var suggestion PRSuggestion
	if err := json.Unmarshal([]byte(raw), &suggestion); err != nil {
		return nil, fmt.Errorf("parse JSON da IA: %w\nresposta: %s", err, raw)
	}

	suggestion.Title = strings.TrimSpace(suggestion.Title)
	if suggestion.Title == "" {
		return nil, fmt.Errorf("resposta da IA incompleta: title é obrigatório")
	}
	if len(suggestion.Summary) == 0 {
		return nil, fmt.Errorf("resposta da IA incompleta: summary é obrigatório")
	}
	if len(suggestion.Changes) == 0 {
		return nil, fmt.Errorf("resposta da IA incompleta: changes é obrigatório")
	}
	if len(suggestion.TestPlan) == 0 {
		return nil, fmt.Errorf("resposta da IA incompleta: test_plan é obrigatório")
	}

	return &suggestion, nil
}

type apiCall func(ctx context.Context, prompt, label string) (string, error)

func suggestPRWithRetry(
	ctx context.Context,
	diff string,
	branch, base, lang, commitLog string,
	maxBytes int,
	call apiCall,
) (*PRSuggestion, error) {
	diff = truncateDiff(diff, maxBytes)
	prompt := buildPRPrompt(diff, branch, base, lang, commitLog)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		content, err := call(ctx, prompt, usageLabel("pr", attempt))
		if err != nil {
			return nil, err
		}
		suggestion, err := parsePRSuggestion(content)
		if err == nil {
			return suggestion, nil
		}
		lastErr = err
		prompt = buildPRPrompt(diff, branch, base, lang, commitLog) +
			"\n\nERRO: resposta anterior inválida. Retorne APENAS JSON válido."
	}
	return nil, lastErr
}
