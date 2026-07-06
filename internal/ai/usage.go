package ai

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/config"
	"github.com/laerciocrestani/gitai/internal/ui"
)

type UsageRecord struct {
	Label            string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          *float64
	CostSource       string
}

type UsageSummary struct {
	Records []UsageRecord
}

func (s *UsageSummary) Add(r UsageRecord) {
	s.Records = append(s.Records, r)
}

func (s UsageSummary) Totals() (prompt, completion, total int, costUSD float64, hasCost bool) {
	for _, r := range s.Records {
		prompt += r.PromptTokens
		completion += r.CompletionTokens
		total += r.TotalTokens
		if r.CostUSD != nil {
			costUSD += *r.CostUSD
			hasCost = true
		}
	}
	return
}

func (s UsageSummary) Print() {
	s.PrintWith(nil, nil)
}

func (s UsageSummary) PrintWith(sess *ui.Session, cfg *config.Config) {
	lines := s.FormatLines(cfg)
	if len(lines) == 0 {
		return
	}
	if sess != nil {
		sess.UsageBlock(lines)
		return
	}
	fmt.Println("Uso de IA")
	for _, line := range lines {
		fmt.Println("  • " + line)
	}
}

func (s UsageSummary) FormatLines(cfg *config.Config) []string {
	if len(s.Records) == 0 {
		return nil
	}

	model := strings.TrimSpace(s.Records[len(s.Records)-1].Model)
	if model == "" && cfg != nil {
		model = cfg.Model
	}

	provider := providerDisplayName(cfg, s.Records[len(s.Records)-1].CostSource)
	p, c, t, cost, hasCost := s.Totals()

	lines := []string{
		fmt.Sprintf("🤖 Modelo: %s (%s)", model, provider),
		fmt.Sprintf("🔢 %d prompt + %d completion = %d tokens", p, c, t),
	}
	if hasCost {
		lines = append(lines, fmt.Sprintf("💰 custo total: $%.6f USD", cost))
	} else {
		lines = append(lines, "💰 custo: não informado")
	}
	return lines
}

func providerDisplayName(cfg *config.Config, costSource string) string {
	if cfg != nil {
		switch cfg.Provider {
		case config.ProviderGemini:
			return "Gemini"
		case config.ProviderOpenAI:
			return "OpenAI"
		case config.ProviderOpenRouter:
			return "OpenRouter"
		}
	}
	switch costSource {
	case "gemini":
		return "Gemini"
	case "openrouter":
		return "OpenRouter"
	case "estimated":
		return "estimativa"
	default:
		return "IA"
	}
}

func formatCost(cost float64, source string) string {
	switch source {
	case "openrouter":
		return fmt.Sprintf("$%.6f USD (OpenRouter)", cost)
	case "gemini":
		return fmt.Sprintf("$%.6f USD (Gemini)", cost)
	case "estimated":
		return fmt.Sprintf("$%.6f USD (estimativa)", cost)
	default:
		return fmt.Sprintf("$%.6f USD", cost)
	}
}

func buildUsageRecord(label string, prompt, completion, total int, apiCost *float64, cfg *config.Config, model string) UsageRecord {
	if total == 0 {
		total = prompt + completion
	}

	record := UsageRecord{
		Label:            label,
		Model:            model,
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
	}

	if apiCost != nil {
		record.CostUSD = apiCost
		record.CostSource = "openrouter"
		return record
	}

	inPrice, outPrice := ResolvePricesForModel(cfg, model)
	if inPrice > 0 || outPrice > 0 {
		cost := float64(prompt)*inPrice/1_000_000 +
			float64(completion)*outPrice/1_000_000
		record.CostUSD = &cost
		record.CostSource = costSourceFor(cfg.Provider)
	}

	return record
}

func usageLabel(base string, attempt int) string {
	if attempt == 0 {
		return base
	}
	return fmt.Sprintf("%s (retry %d)", base, attempt)
}
