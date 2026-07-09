package config

import "testing"

func TestIsValidGeminiAPIKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"AIzaSyAbc123", true},
		{"AQ.Ab8RN6J7GBYUsM1w", true},
		{"sk-or-v1-abc", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isValidGeminiAPIKey(tt.key); got != tt.want {
			t.Fatalf("isValidGeminiAPIKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestValidateGeminiAQKey(t *testing.T) {
	cfg := &Config{
		Provider: ProviderGemini,
		APIKey:   "AQ.Ab8RN6J7GBYUsM1w",
		Model:    "gemini-2.5-flash-lite",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil for AQ. key", err)
	}
}
