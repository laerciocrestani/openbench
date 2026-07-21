package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/laerciocrestani/openbench/internal/config"
)

// streamGeminiChat uses a non-SSE generateContent call and emits the full
// reply as one chunk (Gemini multi-turn streaming can be added later).
func streamGeminiChat(
	ctx context.Context,
	cfg *config.Config,
	messages []ChatMessage,
	onChunk func(delta string),
) (*ChatStreamResult, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("modelo não configurado")
	}

	var systemParts []string
	var contents []geminiChatContent
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		switch role {
		case "system":
			systemParts = append(systemParts, m.Content)
		case "assistant":
			contents = append(contents, geminiChatContent{
				Role:  "model",
				Parts: []geminiPart{{Text: m.Content}},
			})
		default:
			contents = append(contents, geminiChatContent{
				Role:  "user",
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}
	if len(contents) == 0 {
		return nil, fmt.Errorf("nenhuma mensagem de usuário")
	}

	reqBody := geminiChatRequest{Contents: contents}
	if len(systemParts) > 0 {
		reqBody.SystemInstruction = &geminiChatContent{
			Parts: []geminiPart{{Text: strings.Join(systemParts, "\n\n")}},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		model,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", cfg.APIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamada Gemini: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{
			Provider:   "Gemini",
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse Gemini: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("erro Gemini: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini retornou resposta vazia")
	}

	var text strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	out := text.String()
	if onChunk != nil && out != "" {
		onChunk(out)
	}

	promptTok, completionTok, totalTok := 0, 0, 0
	if parsed.UsageMetadata != nil {
		promptTok = parsed.UsageMetadata.PromptTokenCount
		completionTok = parsed.UsageMetadata.CandidatesTokenCount
		totalTok = parsed.UsageMetadata.TotalTokenCount
	}
	rec := buildUsageRecord("chat", promptTok, completionTok, totalTok, nil, cfg, model)
	return &ChatStreamResult{Content: out, Usage: rec}, nil
}

type geminiChatRequest struct {
	SystemInstruction *geminiChatContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiChatContent `json:"contents"`
}

type geminiChatContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}
