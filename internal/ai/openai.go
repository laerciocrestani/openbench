package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/laerciocrestani/gitai/internal/config"
)

type openAIClient struct {
	cfg        *config.Config
	endpoint   string
	httpClient *http.Client
	usage      UsageSummary
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
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

func NewOpenAI(cfg *config.Config, endpoint string) Provider {
	return &openAIClient{
		cfg:      cfg,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *openAIClient) UsageStats() UsageSummary {
	return c.usage
}

func (c *openAIClient) SuggestCommit(ctx context.Context, diff string, lang string) (*CommitSuggestion, error) {
	diff = truncateDiff(diff, c.cfg.MaxDiffBytes)
	prompt := buildPrompt(diff, lang)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		content, err := c.chat(ctx, prompt, usageLabel("commit", attempt))
		if err != nil {
			return nil, err
		}
		suggestion, err := parseSuggestion(content)
		if err == nil {
			return suggestion, nil
		}
		lastErr = err
		prompt = buildPrompt(diff, lang) + "\n\nERRO: resposta anterior inválida. Retorne APENAS JSON válido."
	}
	return nil, lastErr
}

func (c *openAIClient) SuggestPR(ctx context.Context, diff, branch, base, lang, commitLog string) (*PRSuggestion, error) {
	return suggestPRWithRetry(ctx, diff, branch, base, lang, commitLog, c.cfg.MaxDiffBytes, c.chat)
}

func (c *openAIClient) chat(ctx context.Context, prompt, label string) (string, error) {
	return callWithRetry(ctx, c.providerName(), func() (string, error) {
		return c.chatOnce(ctx, prompt, label)
	})
}

func (c *openAIClient) chatOnce(ctx context.Context, prompt, label string) (string, error) {
	reqBody := chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	if c.cfg.Provider == config.ProviderOpenRouter {
		req.Header.Set("HTTP-Referer", "https://github.com/laerciocrestani/gitai")
		req.Header.Set("X-Title", "gitai")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chamada API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", &APIError{
			Provider:   c.providerName(),
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse resposta API: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("erro da API: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API retornou resposta vazia")
	}

	if chatResp.Usage != nil {
		var apiCost *float64
		if c.cfg.Provider == config.ProviderOpenRouter {
			apiCost = chatResp.Usage.Cost
		}
		c.usage.Add(buildUsageRecord(
			label,
			chatResp.Usage.PromptTokens,
			chatResp.Usage.CompletionTokens,
			chatResp.Usage.TotalTokens,
			apiCost,
			c.cfg,
		))
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *openAIClient) providerName() string {
	switch c.cfg.Provider {
	case config.ProviderOpenRouter:
		return "OpenRouter"
	case config.ProviderOpenAI:
		return "OpenAI"
	default:
		return "API"
	}
}
