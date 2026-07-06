package ai

import (
	"testing"

	"github.com/laerciocrestani/gitai/internal/config"
)

func TestUsageSummaryFormatLines(t *testing.T) {
	cost := 0.000272
	summary := UsageSummary{
		Records: []UsageRecord{{
			Model:            "gemini-2.0-flash-lite",
			PromptTokens:     2146,
			CompletionTokens: 143,
			TotalTokens:      2289,
			CostUSD:          &cost,
			CostSource:       "gemini",
		}},
	}
	cfg := &config.Config{Provider: config.ProviderGemini, Model: "gemini-2.5-flash-lite"}
	lines := summary.FormatLines(cfg)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "🤖 Modelo: gemini-2.0-flash-lite (Gemini)" {
		t.Fatalf("unexpected model line: %q", lines[0])
	}
	if lines[1] != "🔢 2146 prompt + 143 completion = 2289 tokens" {
		t.Fatalf("unexpected tokens line: %q", lines[1])
	}
	if lines[2] != "💰 custo total: $0.000272 USD" {
		t.Fatalf("unexpected cost line: %q", lines[2])
	}
}
