package config

import (
	"fmt"
	"strings"
)

// DetailLevel controls how much diff context and output density AI uses for commit/PR.
type DetailLevel string

const (
	DetailMinimal  DetailLevel = "minimal"
	DetailStandard DetailLevel = "standard"
	DetailThorough DetailLevel = "thorough"
)

// ParseDetailLevel accepts minimal|standard|thorough (case-insensitive).
func ParseDetailLevel(s string) (DetailLevel, error) {
	switch DetailLevel(strings.ToLower(strings.TrimSpace(s))) {
	case DetailMinimal:
		return DetailMinimal, nil
	case DetailStandard, "":
		return DetailStandard, nil
	case DetailThorough:
		return DetailThorough, nil
	default:
		return "", fmt.Errorf("detail inválido: %q (use minimal, standard ou thorough)", s)
	}
}

// NormalizeDetailLevel maps empty/unknown to Standard.
func NormalizeDetailLevel(level DetailLevel) DetailLevel {
	switch DetailLevel(strings.ToLower(strings.TrimSpace(string(level)))) {
	case DetailMinimal:
		return DetailMinimal
	case DetailThorough:
		return DetailThorough
	default:
		return DetailStandard
	}
}

// EffectiveCommitDetail returns the normalized commit detail preference.
func (c *Config) EffectiveCommitDetail() DetailLevel {
	if c == nil {
		return DetailStandard
	}
	return NormalizeDetailLevel(c.CommitDetail)
}

// EffectivePRDetail returns the normalized PR detail preference.
func (c *Config) EffectivePRDetail() DetailLevel {
	if c == nil {
		return DetailStandard
	}
	return NormalizeDetailLevel(c.PRDetail)
}

// DiffBytesFor returns the patch byte budget for a detail level.
// Minimal uses ~25% of max_diff_bytes (floor 4096); Standard/Thorough use the full cap.
func (c *Config) DiffBytesFor(level DetailLevel) int {
	max := 120000
	if c != nil && c.MaxDiffBytes > 0 {
		max = c.MaxDiffBytes
	}
	switch NormalizeDetailLevel(level) {
	case DetailMinimal:
		n := max / 4
		const floor = 4096
		if n < floor {
			return floor
		}
		return n
	default:
		return max
	}
}
