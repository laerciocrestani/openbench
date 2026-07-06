package pricing

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	SourceGoogle = "https://ai.google.dev/gemini-api/docs/pricing"
)

type ModelPrice struct {
	InputPer1M  float64 `yaml:"input_per_1m"`
	OutputPer1M float64 `yaml:"output_per_1m"`
}

type Store struct {
	UpdatedAt time.Time             `yaml:"updated_at"`
	Source    string                `yaml:"source"`
	Provider  string                `yaml:"provider"`
	Models    map[string]ModelPrice `yaml:"models"`
}

func StorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gitai", "pricing.yaml"), nil
}

func Load() (*Store, error) {
	path, err := StorePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var store Store
	if err := yaml.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return &store, nil
}

func Save(store Store) error {
	path, err := StorePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(store)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) PricesForModel(model string) (input, output float64, ok bool) {
	if s == nil || len(s.Models) == 0 {
		return 0, 0, false
	}
	if p, found := s.Models[model]; found {
		return p.InputPer1M, p.OutputPer1M, true
	}
	return 0, 0, false
}

func BuiltInGeminiDefaults() map[string]ModelPrice {
	return map[string]ModelPrice{
		"gemini-2.5-flash-lite": {InputPer1M: 0.10, OutputPer1M: 0.40},
		"gemini-2.0-flash-lite": {InputPer1M: 0.10, OutputPer1M: 0.40},
		"gemini-2.5-flash":      {InputPer1M: 0.30, OutputPer1M: 2.50},
		"gemini-2.0-flash":      {InputPer1M: 0.10, OutputPer1M: 0.40},
		"gemini-2.5-pro":        {InputPer1M: 1.25, OutputPer1M: 10.00},
		"gemini-3.1-flash-lite": {InputPer1M: 0.25, OutputPer1M: 1.50},
		"gemini-3-flash":        {InputPer1M: 0.50, OutputPer1M: 3.00},
		"gemini-3-flash-preview": {InputPer1M: 0.50, OutputPer1M: 3.00},
		"gemini-3.1-pro":        {InputPer1M: 2.00, OutputPer1M: 12.00},
		"gemini-3.1-pro-preview": {InputPer1M: 2.00, OutputPer1M: 12.00},
	}
}
