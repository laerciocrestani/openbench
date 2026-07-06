package ai

import (
	"strings"
	"testing"

	"github.com/laerciocrestani/gitai/internal/config"
)

func TestGeminiDefaultPrices(t *testing.T) {
	tests := map[string][2]float64{
		"gemini-2.5-flash-lite": {0.10, 0.40},
		"gemini-2.5-flash":      {0.30, 2.50},
		"gemini-2.5-pro":        {1.25, 10.00},
		"gemini-3.1-flash-lite": {0.25, 1.50},
		"gemini-3-flash":        {0.50, 3.00},
		"gemini-3.1-pro":        {2.00, 12.00},
	}
	for model, want := range tests {
		in, out := geminiDefaultPrices(model)
		if in != want[0] || out != want[1] {
			t.Fatalf("%s: got %.2f/%.2f, want %.2f/%.2f", model, in, out, want[0], want[1])
		}
	}
}

func TestEstimateCostGemini(t *testing.T) {
	cfg := &config.Config{
		Provider: config.ProviderGemini,
		Model:    "gemini-2.5-flash-lite",
	}
	diff := strings.Repeat("a", 4000) // ~1000 tokens from diff

	est := EstimateCost(cfg, diff, "commit")
	if !est.HasCost {
		t.Fatal("expected cost estimate")
	}
	if est.InputTokens < 1400 {
		t.Fatalf("input tokens too low: %d", est.InputTokens)
	}
	if est.OutputTokens != 250 {
		t.Fatalf("output tokens: got %d, want 250", est.OutputTokens)
	}
	if est.CostUSD <= 0 {
		t.Fatalf("expected positive cost, got %f", est.CostUSD)
	}

	formatted := est.Format(cfg.Provider)
	if !strings.Contains(formatted, "Gemini") {
		t.Fatalf("format missing Gemini: %s", formatted)
	}
}

func TestResolvePricesConfigOverride(t *testing.T) {
	cfg := &config.Config{
		Provider:          config.ProviderGemini,
		Model:             "gemini-2.5-flash-lite",
		InputPricePer1M:   0.50,
		OutputPricePer1M:  1.00,
	}
	in, out := ResolvePrices(cfg)
	if in != 0.50 || out != 1.00 {
		t.Fatalf("override: got %.2f/%.2f", in, out)
	}
}
