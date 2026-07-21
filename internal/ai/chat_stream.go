package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
)

// ChatMessage is one turn in a conversational chat.
type ChatMessage struct {
	Role       string     `json:"role"` // system | user | assistant | tool
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ChatStreamResult is returned after a successful stream.
type ChatStreamResult struct {
	Content string
	Usage   UsageRecord
}

// StreamChat streams a multi-turn chat completion. onChunk is called for each
// text delta (may be empty for usage-only final events).
func StreamChat(
	ctx context.Context,
	cfg *config.Config,
	messages []ChatMessage,
	onChunk func(delta string),
) (*ChatStreamResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config ausente")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("api_key não configurada")
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("nenhuma mensagem")
	}

	switch cfg.Provider {
	case config.ProviderOpenAI:
		return streamOpenAIChat(ctx, cfg, "https://api.openai.com/v1/chat/completions", messages, onChunk)
	case config.ProviderOpenRouter:
		return streamOpenAIChat(ctx, cfg, "https://openrouter.ai/api/v1/chat/completions", messages, onChunk)
	case config.ProviderGemini:
		return streamGeminiChat(ctx, cfg, messages, onChunk)
	default:
		return nil, fmt.Errorf("provider desconhecido: %s", cfg.Provider)
	}
}

type streamChatRequest struct {
	Model         string        `json:"model"`
	Messages      []chatMessage `json:"messages"`
	Stream        bool          `json:"stream"`
	StreamOptions *streamOpts   `json:"stream_options,omitempty"`
}

type streamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int      `json:"prompt_tokens"`
		CompletionTokens int      `json:"completion_tokens"`
		TotalTokens      int      `json:"total_tokens"`
		Cost             *float64 `json:"cost"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func streamOpenAIChat(
	ctx context.Context,
	cfg *config.Config,
	endpoint string,
	messages []ChatMessage,
	onChunk func(delta string),
) (*ChatStreamResult, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("modelo não configurado")
	}

	msgs := make([]chatMessage, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role == "" {
			role = "user"
		}
		msgs = append(msgs, chatMessage{Role: role, Content: m.Content})
	}

	reqBody := streamChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   true,
		StreamOptions: &streamOpts{
			IncludeUsage: true,
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Accept", "text/event-stream")
	if cfg.Provider == config.ProviderOpenRouter {
		req.Header.Set("HTTP-Referer", "https://github.com/laerciocrestani/openbench")
		req.Header.Set("X-Title", "openbench")
	}

	client := &http.Client{Timeout: 0} // stream; cancel via ctx
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamada API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, &APIError{
			Provider:   string(cfg.Provider),
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var (
		content strings.Builder
		usage   *struct {
			PromptTokens     int
			CompletionTokens int
			TotalTokens      int
			Cost             *float64
		}
	)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return nil, fmt.Errorf("erro da API: %s", chunk.Error.Message)
		}
		if chunk.Usage != nil {
			usage = &struct {
				PromptTokens     int
				CompletionTokens int
				TotalTokens      int
				Cost             *float64
			}{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
				Cost:             chunk.Usage.Cost,
			}
		}
		for _, ch := range chunk.Choices {
			delta := ch.Delta.Content
			if delta == "" {
				continue
			}
			content.WriteString(delta)
			if onChunk != nil {
				onChunk(delta)
			}
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return nil, fmt.Errorf("leitura do stream: %w", err)
	}

	text := content.String()
	if text == "" {
		return nil, fmt.Errorf("API retornou resposta vazia")
	}

	var apiCost *float64
	promptTok, completionTok, totalTok := 0, 0, 0
	if usage != nil {
		promptTok = usage.PromptTokens
		completionTok = usage.CompletionTokens
		totalTok = usage.TotalTokens
		if cfg.Provider == config.ProviderOpenRouter {
			apiCost = usage.Cost
		}
	} else {
		// Fallback rough estimate when provider omits stream usage.
		promptTok = estimateInputTokens(joinChatContents(messages), "chat")
		completionTok = len(text) / 4
		totalTok = promptTok + completionTok
	}

	rec := buildUsageRecord("chat", promptTok, completionTok, totalTok, apiCost, cfg, model)
	return &ChatStreamResult{Content: text, Usage: rec}, nil
}

func joinChatContents(messages []ChatMessage) string {
	var b strings.Builder
	for _, m := range messages {
		b.WriteString(m.Content)
		b.WriteByte('\n')
	}
	return b.String()
}
