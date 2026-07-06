package ai

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/config"
	"github.com/laerciocrestani/gitai/internal/pricing"
)

type CostEstimate struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	HasCost      bool
}

func ResolvePrices(cfg *config.Config) (inputPer1M, outputPer1M float64) {
	return ResolvePricesForModel(cfg, cfg.Model)
}

func ResolvePricesForModel(cfg *config.Config, model string) (inputPer1M, outputPer1M float64) {
	if cfg.InputPricePer1M > 0 || cfg.OutputPricePer1M > 0 {
		return cfg.InputPricePer1M, cfg.OutputPricePer1M
	}

	if cfg.Provider == config.ProviderGemini {
		if store, err := pricing.Load(); err == nil {
			if in, out, ok := store.PricesForModel(model); ok {
				return in, out
			}
		}
		return geminiDefaultPrices(model)
	}

	switch cfg.Provider {
	case config.ProviderOpenAI:
		return 0.15, 0.60
	case config.ProviderOpenRouter:
		return 0.14, 0.28
	default:
		return 0, 0
	}
}

func geminiDefaultPrices(model string) (float64, float64) {
	switch model {
	case "gemini-2.5-flash-lite", "gemini-2.0-flash-lite":
		return 0.10, 0.40

	case "gemini-2.5-flash":
		return 0.30, 2.50

	case "gemini-2.0-flash":
		return 0.10, 0.40 // modelo legado

	case "gemini-2.5-pro":
		return 1.25, 10.00

	case "gemini-3.1-flash-lite":
		return 0.25, 1.50

	case "gemini-3-flash", "gemini-3-flash-preview":
		return 0.50, 3.00

	case "gemini-3.1-pro", "gemini-3.1-pro-preview":
		return 2.00, 12.00

	default:
		return 0.10, 0.40
	}
}

func EstimateCost(cfg *config.Config, diff string, task string) CostEstimate {
	inputTokens := estimateInputTokens(diff, task)
	outputTokens := estimateOutputTokens(task)

	inPrice, outPrice := ResolvePrices(cfg)
	if inPrice == 0 && outPrice == 0 {
		return CostEstimate{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		}
	}

	cost := float64(inputTokens)*inPrice/1_000_000 +
		float64(outputTokens)*outPrice/1_000_000

	return CostEstimate{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CostUSD:      cost,
		HasCost:      true,
	}
}

func (e CostEstimate) Format(provider config.Provider) string {
	total := e.InputTokens + e.OutputTokens
	if !e.HasCost {
		return fmt.Sprintf("~%d tokens estimados (input ~%d + output ~%d)",
			total, e.InputTokens, e.OutputTokens)
	}
	source := costSourceFor(provider)
	return fmt.Sprintf("~%d tokens · %s (input ~%d + output ~%d)",
		total, formatCost(e.CostUSD, source), e.InputTokens, e.OutputTokens)
}

// DescribePreparedInput resume o que será enviado ao modelo antes da chamada.
func DescribePreparedInput(cfg *config.Config, diff, task string) string {
	diff = truncateDiff(diff, cfg.MaxDiffBytes)
	tokens := estimateInputTokens(diff, task)
	line := fmt.Sprintf("Input: ~%d tokens · modelo %s", tokens, cfg.Model)
	if fb := strings.TrimSpace(cfg.FallbackModel); fb != "" && fb != cfg.Model {
		line += fmt.Sprintf(" · fallback %s", fb)
	}
	return line
}

// FormatLatestUsage formata tokens e custo reais da última chamada.
func FormatLatestUsage(summary UsageSummary) string {
	if len(summary.Records) == 0 {
		return ""
	}
	r := summary.Records[len(summary.Records)-1]
	line := fmt.Sprintf("Uso: %d input + %d output = %d tokens", r.PromptTokens, r.CompletionTokens, r.TotalTokens)
	if r.Model != "" {
		line += " · " + r.Model
	}
	if r.CostUSD != nil {
		line += " · " + formatCost(*r.CostUSD, r.CostSource)
	}
	return line
}

func costSourceFor(provider config.Provider) string {
	switch provider {
	case config.ProviderOpenRouter:
		return "openrouter"
	case config.ProviderGemini:
		return "gemini"
	default:
		return "estimated"
	}
}

func estimateInputTokens(diff, task string) int {
	tokens := len(diff) / 4
	tokens += promptOverhead(task)
	return tokens
}

func estimateOutputTokens(task string) int {
	switch task {
	case "pr":
		return 700
	default:
		return 250
	}
}

func promptOverhead(task string) int {
	switch task {
	case "pr":
		return 900
	default:
		return 500
	}
}
