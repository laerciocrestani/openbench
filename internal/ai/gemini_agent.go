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

func completeGeminiChatTurn(
	ctx context.Context,
	cfg *config.Config,
	messages []ChatMessage,
	withTools bool,
) (*chatTurnResult, error) {
	model := resolveGeminiModel(strings.TrimSpace(cfg.Model))
	if model == "" {
		return nil, fmt.Errorf("modelo não configurado")
	}

	systemParts, contents, err := toGeminiAgentContents(messages)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return nil, fmt.Errorf("nenhuma mensagem de usuário")
	}

	reqBody := geminiAgentRequest{Contents: contents}
	if len(systemParts) > 0 {
		reqBody.SystemInstruction = &geminiChatContent{
			Parts: []geminiPart{{Text: strings.Join(systemParts, "\n\n")}},
		}
	}
	if withTools {
		reqBody.Tools = []geminiTool{{
			FunctionDeclarations: GeminiFunctionDeclarations(),
		}}
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
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

	var parsed geminiAgentResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse Gemini: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("erro Gemini: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 {
		return nil, fmt.Errorf("Gemini retornou resposta vazia")
	}

	parts := parsed.Candidates[0].Content.Parts
	var text strings.Builder
	var toolCalls []ToolCall
	for i, p := range parts {
		if p.Text != "" {
			text.WriteString(p.Text)
		}
		if p.FunctionCall != nil && p.FunctionCall.Name != "" {
			args := p.FunctionCall.Args
			if args == nil {
				args = map[string]any{}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:   fmt.Sprintf("gemini-%d-%s", i, p.FunctionCall.Name),
				Name: p.FunctionCall.Name,
				Args: args,
			})
		}
	}

	promptTok, completionTok, totalTok := 0, 0, 0
	if parsed.UsageMetadata != nil {
		promptTok = parsed.UsageMetadata.PromptTokenCount
		completionTok = parsed.UsageMetadata.CandidatesTokenCount
		totalTok = parsed.UsageMetadata.TotalTokenCount
	}
	rec := buildUsageRecord("chat", promptTok, completionTok, totalTok, nil, cfg, model)
	return &chatTurnResult{
		Content:   text.String(),
		ToolCalls: toolCalls,
		Usage:     rec,
	}, nil
}

type geminiAgentRequest struct {
	SystemInstruction *geminiChatContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiChatContent `json:"contents"`
	Tools             []geminiTool        `json:"tools,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []map[string]any `json:"functionDeclarations"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiAgentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func toGeminiAgentContents(messages []ChatMessage) (systemParts []string, contents []geminiChatContent, err error) {
	for i := 0; i < len(messages); i++ {
		m := messages[i]
		role := strings.TrimSpace(m.Role)
		switch role {
		case "system":
			systemParts = append(systemParts, m.Content)
		case "assistant":
			parts := make([]geminiPart, 0, 1+len(m.ToolCalls))
			if m.Content != "" {
				parts = append(parts, geminiPart{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				args := tc.Args
				if args == nil {
					args = map[string]any{}
				}
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFunctionCall{
						Name: tc.Name,
						Args: args,
					},
				})
			}
			if len(parts) == 0 {
				parts = append(parts, geminiPart{Text: ""})
			}
			contents = append(contents, geminiChatContent{Role: "model", Parts: parts})
		case "tool":
			// Batch consecutive tool results into one user turn (Gemini expects that).
			parts := make([]geminiPart, 0, 2)
			for {
				name := messages[i].Name
				if name == "" {
					name = "tool"
				}
				parts = append(parts, geminiPart{
					FunctionResponse: &geminiFunctionResponse{
						Name: name,
						Response: map[string]any{
							"result": messages[i].Content,
						},
					},
				})
				if i+1 >= len(messages) || strings.TrimSpace(messages[i+1].Role) != "tool" {
					break
				}
				i++
			}
			contents = append(contents, geminiChatContent{Role: "user", Parts: parts})
		default: // user
			contents = append(contents, geminiChatContent{
				Role:  "user",
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}
	return systemParts, contents, nil
}
