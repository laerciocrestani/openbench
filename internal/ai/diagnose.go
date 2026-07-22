package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// HealthExplanation é a resposta estruturada da IA para o doctor.
type HealthExplanation struct {
	Summary  string   `json:"summary"`
	Cause    string   `json:"cause"`
	Risk     string   `json:"risk"`
	Steps    []string `json:"steps"`
	Warnings []string `json:"warnings"`
}

func buildHealthPrompt(facts, lang string) string {
	var b strings.Builder
	b.WriteString(`Analise o panorama de saúde do repositório Git abaixo e oriente o desenvolvedor.

Responda SOMENTE com JSON válido, sem markdown, sem explicações:
{
  "summary": "visão geral em 1-2 frases",
  "cause": "causa raiz do principal problema (ou 'Nenhum problema crítico' se saudável)",
  "risk": "low|medium|high",
  "steps": ["comando ou ação concreta 1", "passo 2"],
  "warnings": ["alertas sobre comandos destrutivos ou riscos — ou array vazio"]
}

Regras:
- Idioma: `)
	b.WriteString(lang)
	b.WriteString(`
- steps: 2-5 passos acionáveis; prefira comandos git/ob quando aplicável
- se commits locais parecem artefatos de build, sugira descartá-los com cautela
- marque reset --hard ou branch -D em warnings, nunca como passo casual
- não invente arquivos ou commits que não aparecem nos fatos
- risk=low quando working tree limpa e sincronizado
- se a PR estiver MERGED: NÃO sugira push/PR de novo na mesma branch; oriente salvar o work (commit/stash), atualizar a base (Pull/Sync) e criar uma NOVA feature branch a partir da base
- se o ÚNICO problema for working tree dirty numa feature branch (PR não MERGED): NÃO sugira stash push/pop; oriente Commit → push → abrir/atualizar PR
- stash só quando for necessário limpar a tree para outra operação (pull/rebase/troca de branch), nunca como passo principal isolado
- se houver "Open PR" (state=OPEN), aí sim commit/push/atualização da PR atual fazem sentido

Fatos do repositório:
`)
	b.WriteString(facts)
	return b.String()
}

func parseHealthExplanation(raw string) (*HealthExplanation, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var explanation HealthExplanation
	if err := json.Unmarshal([]byte(raw), &explanation); err != nil {
		return nil, fmt.Errorf("parse JSON da IA: %w\nresposta: %s", err, raw)
	}

	explanation.Summary = strings.TrimSpace(explanation.Summary)
	explanation.Cause = strings.TrimSpace(explanation.Cause)
	explanation.Risk = strings.TrimSpace(explanation.Risk)

	if explanation.Summary == "" {
		return nil, fmt.Errorf("resposta da IA incompleta: summary é obrigatório")
	}
	if len(explanation.Steps) == 0 {
		return nil, fmt.Errorf("resposta da IA incompleta: steps é obrigatório")
	}

	return &explanation, nil
}

type healthAPICall func(ctx context.Context, prompt, label string) (string, error)

func explainHealthWithRetry(ctx context.Context, facts, lang string, call healthAPICall) (*HealthExplanation, error) {
	prompt := buildHealthPrompt(facts, lang)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		content, err := call(ctx, prompt, usageLabel("doctor", attempt))
		if err != nil {
			return nil, err
		}
		explanation, err := parseHealthExplanation(content)
		if err == nil {
			return explanation, nil
		}
		lastErr = err
		prompt = buildHealthPrompt(facts, lang) + "\n\nERRO: resposta anterior inválida. Retorne APENAS JSON válido."
	}
	return nil, lastErr
}
