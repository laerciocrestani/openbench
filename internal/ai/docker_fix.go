package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	maxDockerFixFiles     = 6
	maxDockerFixFileRunes = 120_000
	maxDockerFixLogRunes  = 24_000
	maxDockerFixSteps     = 12
)

// DockerFixServiceStatus is one compose service from the failed action UI.
type DockerFixServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// DockerFixContext is the failure payload sent to the model.
type DockerFixContext struct {
	Action      string                   `json:"action"`
	Message     string                   `json:"message"`
	Lines       []string                 `json:"lines"`
	Services    []DockerFixServiceStatus `json:"services"`
	ComposeFile string                   `json:"compose_file,omitempty"`
	SkillBody   string                   `json:"-"`
}

// DockerFixFile is one proposed file write.
type DockerFixFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Reason  string `json:"reason"`
}

// DockerFixStep is one proposed remediation action.
type DockerFixStep struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"` // docker_stop | docker_rm | docker_up | write_file
	Title  string `json:"title"`
	Target string `json:"target"` // service name or relative file path
	Risk   string `json:"risk"`   // ok | warn | destructive
}

// DockerFixSuggestion is the structured AI response for fixing a Docker failure.
type DockerFixSuggestion struct {
	Problem    string          `json:"problem"`
	Resolution string          `json:"resolution"`
	Steps      []DockerFixStep `json:"steps"`
	Files      []DockerFixFile `json:"files"`
	Notes      []string        `json:"notes"`
}

var allowedDockerFixKinds = map[string]bool{
	"docker_stop":     true,
	"docker_rm":       true,
	"docker_up":       true,
	"docker_recreate": true,
	"write_file":      true,
}

func buildDockerFixPrompt(fc DockerFixContext, lang string) string {
	var b strings.Builder
	b.WriteString(`Você é o especialista de Docker Compose do openbench.
Analise a falha de orquestração abaixo e proponha a menor correção segura executável pelo app.

Responda SOMENTE com JSON válido, sem markdown:
{
  "problem": "o que falhou e por quê (1-3 frases, com evidência do log)",
  "resolution": "o que será feito para corrigir (1-3 frases)",
  "steps": [
    {
      "id": "1",
      "kind": "docker_stop|docker_rm|docker_up|docker_recreate|write_file",
      "title": "título curto",
      "target": "nome-do-servico OU path/relativo",
      "risk": "ok|warn|destructive"
    }
  ],
  "files": [
    {
      "path": "caminho/relativo",
      "content": "conteúdo COMPLETO do arquivo após a correção",
      "reason": "por que este arquivo precisa mudar"
    }
  ],
  "notes": ["avisos opcionais — ex.: processo no host ocupando a porta; NÃO proponha kill como step"]
}

Kinds permitidos (apenas estes):
- docker_stop: docker compose stop <service>
- docker_rm: docker compose rm -f <service> (risk: warn)
- docker_up: docker compose up -d <service>
- docker_recreate: docker compose up -d --force-recreate --no-deps <service> (útil quando o container Started e depois Exited)
- write_file: grava files[].content no path (target = path)

Regras:
- Idioma: `)
	b.WriteString(lang)
	b.WriteString(`
- Ancore-se no erro depositado (message/lines/services); não invente serviços ou portas
- Preferir menor ação: se a porta está alocada por outro container do mesmo compose, proponha stop/rm desse serviço
- NÃO proponha kill de processo no host, down -v, prune ou apagar volumes como steps (pode mencionar em notes)
- path sempre relativo ao root do projeto; sem .. nem absolutos
- todo write_file precisa de entry correspondente em files com reason não vazio
- no máximo `)
	b.WriteString(fmt.Sprintf("%d", maxDockerFixSteps))
	b.WriteString(` steps e `)
	b.WriteString(fmt.Sprintf("%d", maxDockerFixFiles))
	b.WriteString(` arquivos
- se não for possível corrigir com confiança, retorne steps=[] e files=[] e explique em problem/resolution/notes
`)

	if body := strings.TrimSpace(fc.SkillBody); body != "" {
		b.WriteString("\n## Playbook docker-debug\n\n")
		b.WriteString(truncateRunes(body, 8_000))
		b.WriteString("\n")
	}

	b.WriteString("\n## Falha Docker\n\n")
	b.WriteString("Ação: docker ")
	b.WriteString(strings.TrimSpace(fc.Action))
	b.WriteString("\n")
	if cf := strings.TrimSpace(fc.ComposeFile); cf != "" {
		b.WriteString("Compose: ")
		b.WriteString(cf)
		b.WriteString("\n")
	}
	b.WriteString("Mensagem: ")
	b.WriteString(strings.TrimSpace(fc.Message))
	b.WriteString("\n\nServiços:\n")
	if len(fc.Services) == 0 {
		b.WriteString("(nenhum)\n")
	} else {
		for _, s := range fc.Services {
			b.WriteString("- ")
			b.WriteString(s.Name)
			b.WriteString(" [")
			b.WriteString(s.Status)
			b.WriteString("]")
			if d := strings.TrimSpace(s.Detail); d != "" {
				b.WriteString(": ")
				b.WriteString(truncateRunes(d, 200))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\nLog (compose):\n")
	logText := strings.Join(fc.Lines, "\n")
	if strings.TrimSpace(logText) == "" {
		logText = "(vazio)"
	}
	b.WriteString(truncateRunes(logText, maxDockerFixLogRunes))
	return b.String()
}

func parseDockerFixSuggestion(raw string) (*DockerFixSuggestion, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var sug DockerFixSuggestion
	if err := json.Unmarshal([]byte(raw), &sug); err != nil {
		return nil, fmt.Errorf("parse JSON da IA: %w\nresposta: %s", err, raw)
	}
	sug.Problem = strings.TrimSpace(sug.Problem)
	sug.Resolution = strings.TrimSpace(sug.Resolution)
	if sug.Problem == "" {
		return nil, fmt.Errorf("resposta da IA incompleta: problem é obrigatório")
	}
	if sug.Resolution == "" {
		return nil, fmt.Errorf("resposta da IA incompleta: resolution é obrigatório")
	}

	if len(sug.Steps) > maxDockerFixSteps {
		sug.Steps = sug.Steps[:maxDockerFixSteps]
	}
	cleanedSteps := make([]DockerFixStep, 0, len(sug.Steps))
	seenIDs := map[string]bool{}
	for i, st := range sug.Steps {
		kind := strings.TrimSpace(st.Kind)
		if !allowedDockerFixKinds[kind] {
			continue
		}
		id := strings.TrimSpace(st.ID)
		if id == "" {
			id = fmt.Sprintf("%d", i+1)
		}
		if seenIDs[id] {
			id = fmt.Sprintf("%s-%d", id, i+1)
		}
		seenIDs[id] = true
		target := strings.TrimSpace(st.Target)
		target = strings.TrimPrefix(target, "./")
		if target == "" {
			continue
		}
		if kind == "write_file" {
			if strings.Contains(target, "..") || strings.HasPrefix(target, "/") {
				continue
			}
		} else {
			// service name: conservative
			if strings.ContainsAny(target, "/\\ \t\n\x00") || strings.Contains(target, "..") {
				continue
			}
		}
		risk := strings.TrimSpace(st.Risk)
		if risk != "ok" && risk != "warn" && risk != "destructive" {
			if kind == "docker_rm" {
				risk = "warn"
			} else {
				risk = "ok"
			}
		}
		title := strings.TrimSpace(st.Title)
		if title == "" {
			title = kind + " " + target
		}
		cleanedSteps = append(cleanedSteps, DockerFixStep{
			ID:     id,
			Kind:   kind,
			Title:  title,
			Target: target,
			Risk:   risk,
		})
	}
	sug.Steps = cleanedSteps

	if len(sug.Files) > maxDockerFixFiles {
		sug.Files = sug.Files[:maxDockerFixFiles]
	}
	cleanedFiles := make([]DockerFixFile, 0, len(sug.Files))
	for _, f := range sug.Files {
		path := strings.TrimSpace(f.Path)
		path = strings.TrimPrefix(path, "./")
		if path == "" || strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
			continue
		}
		if strings.ContainsAny(path, "\x00") {
			continue
		}
		reason := strings.TrimSpace(f.Reason)
		if reason == "" {
			continue
		}
		content := f.Content
		if runeCount(content) > maxDockerFixFileRunes {
			return nil, fmt.Errorf("arquivo %s excede limite de tamanho da correção", path)
		}
		cleanedFiles = append(cleanedFiles, DockerFixFile{
			Path:    path,
			Content: content,
			Reason:  reason,
		})
	}
	sug.Files = cleanedFiles

	// Drop write_file steps without matching file.
	fileSet := map[string]bool{}
	for _, f := range sug.Files {
		fileSet[f.Path] = true
	}
	filtered := sug.Steps[:0]
	for _, st := range sug.Steps {
		if st.Kind == "write_file" && !fileSet[st.Target] {
			continue
		}
		filtered = append(filtered, st)
	}
	sug.Steps = filtered

	return &sug, nil
}

func suggestDockerFixWithRetry(ctx context.Context, fc DockerFixContext, lang string, call apiCall) (*DockerFixSuggestion, error) {
	if strings.TrimSpace(lang) == "" {
		lang = "pt-BR"
	}
	prompt := buildDockerFixPrompt(fc, lang)
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := call(ctx, prompt, "docker-fix")
		if err != nil {
			return nil, err
		}
		sug, err := parseDockerFixSuggestion(raw)
		if err == nil {
			return sug, nil
		}
		lastErr = err
		prompt = buildDockerFixPrompt(fc, lang) + "\n\nA resposta anterior era inválida. Retorne APENAS JSON válido."
	}
	return nil, lastErr
}
