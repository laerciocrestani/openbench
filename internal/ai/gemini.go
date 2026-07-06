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

type geminiClient struct {
	cfg        *config.Config
	httpClient *http.Client
	usage      UsageSummary
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
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

func NewGemini(cfg *config.Config) Provider {
	return &geminiClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *geminiClient) UsageStats() UsageSummary {
	return c.usage
}

func (c *geminiClient) SuggestCommit(ctx context.Context, diff string, lang string) (*CommitSuggestion, error) {
	diff = truncateDiff(diff, c.cfg.MaxDiffBytes)
	prompt := buildPrompt(diff, lang)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		content, err := c.generate(ctx, prompt, usageLabel("commit", attempt))
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

func (c *geminiClient) SuggestPR(ctx context.Context, diff, branch, base, lang, commitLog string) (*PRSuggestion, error) {
	return suggestPRWithRetry(ctx, diff, branch, base, lang, commitLog, c.cfg.MaxDiffBytes, c.generate)
}

func (c *geminiClient) generate(ctx context.Context, prompt, label string) (string, error) {
	return callWithRetry(ctx, "Gemini", func() (string, error) {
		return c.generateOnce(ctx, prompt, label)
	})
}

func (c *geminiClient) generateOnce(ctx context.Context, prompt, label string) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		c.cfg.Model,
		c.cfg.APIKey,
	)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chamada Gemini: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", &APIError{
			Provider:   "Gemini",
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("parse resposta Gemini: %w", err)
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("erro Gemini: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini retornou resposta vazia")
	}

	if geminiResp.UsageMetadata != nil {
		meta := geminiResp.UsageMetadata
		c.usage.Add(buildUsageRecord(
			label,
			meta.PromptTokenCount,
			meta.CandidatesTokenCount,
			meta.TotalTokenCount,
			nil,
			c.cfg,
		))
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}
