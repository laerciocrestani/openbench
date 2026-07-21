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

type openAIToolChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIAgentMsg    `json:"messages"`
	Tools    []map[string]any    `json:"tools,omitempty"`
	Stream   bool                `json:"stream"`
}

type openAIAgentMsg struct {
	Role       string              `json:"role"`
	Content    *string             `json:"content"`
	ToolCalls  []openAIToolCallMsg `json:"tool_calls,omitempty"`
	ToolCallID string              `json:"tool_call_id,omitempty"`
	Name       string              `json:"name,omitempty"`
}

type openAIToolCallMsg struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIToolChatResponse struct {
	Choices []struct {
		Message struct {
			Content   *string             `json:"content"`
			ToolCalls []openAIToolCallMsg `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
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

func completeOpenAIChatTurn(
	ctx context.Context,
	cfg *config.Config,
	endpoint string,
	messages []ChatMessage,
	withTools bool,
) (*chatTurnResult, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("modelo não configurado")
	}

	msgs, err := toOpenAIAgentMessages(messages)
	if err != nil {
		return nil, err
	}

	reqBody := openAIToolChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	}
	if withTools {
		reqBody.Tools = ChatToolDefinitions()
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
	if cfg.Provider == config.ProviderOpenRouter {
		req.Header.Set("HTTP-Referer", "https://github.com/laerciocrestani/openbench")
		req.Header.Set("X-Title", "openbench")
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamada API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{
			Provider:   string(cfg.Provider),
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var parsed openAIToolChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse resposta: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("erro da API: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("API retornou resposta vazia")
	}

	msg := parsed.Choices[0].Message
	content := ""
	if msg.Content != nil {
		content = *msg.Content
	}

	var toolCalls []ToolCall
	for _, tc := range msg.ToolCalls {
		args := map[string]any{}
		if strings.TrimSpace(tc.Function.Arguments) != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		})
	}

	var apiCost *float64
	promptTok, completionTok, totalTok := 0, 0, 0
	if parsed.Usage != nil {
		promptTok = parsed.Usage.PromptTokens
		completionTok = parsed.Usage.CompletionTokens
		totalTok = parsed.Usage.TotalTokens
		if cfg.Provider == config.ProviderOpenRouter {
			apiCost = parsed.Usage.Cost
		}
	}

	rec := buildUsageRecord("chat", promptTok, completionTok, totalTok, apiCost, cfg, model)
	return &chatTurnResult{
		Content:   content,
		ToolCalls: toolCalls,
		Usage:     rec,
	}, nil
}

func toOpenAIAgentMessages(messages []ChatMessage) ([]openAIAgentMsg, error) {
	out := make([]openAIAgentMsg, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		switch role {
		case "system", "user", "assistant", "tool":
		default:
			role = "user"
		}

		msg := openAIAgentMsg{Role: role}
		if role == "tool" {
			content := m.Content
			msg.Content = &content
			msg.ToolCallID = m.ToolCallID
			if msg.ToolCallID == "" {
				msg.ToolCallID = "tool-" + m.Name
			}
			msg.Name = m.Name
			out = append(out, msg)
			continue
		}

		if len(m.ToolCalls) > 0 {
			if m.Content != "" {
				c := m.Content
				msg.Content = &c
			} else {
				msg.Content = nil
			}
			for _, tc := range m.ToolCalls {
				id := tc.ID
				if id == "" {
					id = fmt.Sprintf("call_%s", tc.Name)
				}
				args, err := json.Marshal(tc.Args)
				if err != nil {
					args = []byte("{}")
				}
				item := openAIToolCallMsg{ID: id, Type: "function"}
				item.Function.Name = tc.Name
				item.Function.Arguments = string(args)
				msg.ToolCalls = append(msg.ToolCalls, item)
			}
			out = append(out, msg)
			continue
		}

		c := m.Content
		msg.Content = &c
		out = append(out, msg)
	}
	return out, nil
}
