package ai

import (
	"fmt"

	"github.com/laerciocrestani/gitai/internal/config"
	"github.com/laerciocrestani/gitai/internal/ui"
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
	s.PrintWith(nil)
}

func (s UsageSummary) PrintWith(sess *ui.Session) {
	lines := s.FormatLines()
	if len(lines) == 0 {
		return
	}
	if sess != nil {
		sess.UsageBlock(lines)
		return
	}
	fmt.Println("--- Uso de IA ---")
	for _, line := range lines {
		fmt.Println(line)
	}
}

func (s UsageSummary) FormatLines() []string {
	if len(s.Records) == 0 {
		return nil
	}

	lines := make([]string, 0, len(s.Records)+1)
	for _, r := range s.Records {
		line := fmt.Sprintf("%s: %d prompt + %d completion = %d tokens",
			r.Label, r.PromptTokens, r.CompletionTokens, r.TotalTokens)
		if r.CostUSD != nil {
			line += " | " + formatCost(*r.CostUSD, r.CostSource)
		}
		lines = append(lines, line)
	}

	p, c, t, cost, hasCost := s.Totals()
	total := fmt.Sprintf("Total: %d prompt + %d completion = %d tokens", p, c, t)
	if hasCost {
		total += fmt.Sprintf(" | custo total: $%.6f USD", cost)
	} else {
		total += " | custo: não informado"
	}
	lines = append(lines, total)
	return lines
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

	inPrice, outPrice := ResolvePrices(cfg)
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
