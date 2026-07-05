package ai

import (
	"fmt"

	"github.com/laerciocrestani/gitia/internal/config"
)

type UsageRecord struct {
	Label            string
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
	if len(s.Records) == 0 {
		return
	}

	fmt.Println("--- Uso de IA ---")
	for _, r := range s.Records {
		fmt.Printf("%s: %d prompt + %d completion = %d tokens",
			r.Label, r.PromptTokens, r.CompletionTokens, r.TotalTokens)
		if r.CostUSD != nil {
			fmt.Printf(" | %s", formatCost(*r.CostUSD, r.CostSource))
		}
		fmt.Println()
	}

	p, c, t, cost, hasCost := s.Totals()
	fmt.Printf("Total: %d prompt + %d completion = %d tokens", p, c, t)
	if hasCost {
		fmt.Printf(" | custo total: $%.6f USD\n", cost)
	} else {
		fmt.Println(" | custo: não informado (use openrouter ou configure input_price_per_1m / output_price_per_1m)")
	}
}

func formatCost(cost float64, source string) string {
	switch source {
	case "openrouter":
		return fmt.Sprintf("$%.6f USD (OpenRouter)", cost)
	case "estimated":
		return fmt.Sprintf("$%.6f USD (estimativa)", cost)
	default:
		return fmt.Sprintf("$%.6f USD", cost)
	}
}

func buildUsageRecord(label string, prompt, completion, total int, apiCost *float64, cfg *config.Config) UsageRecord {
	if total == 0 {
		total = prompt + completion
	}

	record := UsageRecord{
		Label:            label,
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
	}

	if apiCost != nil {
		record.CostUSD = apiCost
		record.CostSource = "openrouter"
		return record
	}

	if cfg.InputPricePer1M > 0 || cfg.OutputPricePer1M > 0 {
		cost := float64(prompt)*cfg.InputPricePer1M/1_000_000 +
			float64(completion)*cfg.OutputPricePer1M/1_000_000
		record.CostUSD = &cost
		record.CostSource = "estimated"
	}

	return record
}

func usageLabel(base string, attempt int) string {
	if attempt == 0 {
		return base
	}
	return fmt.Sprintf("%s (retry %d)", base, attempt)
}
