package config

import "testing"

func TestParseDetailLevel(t *testing.T) {
	tests := []struct {
		in      string
		want    DetailLevel
		wantErr bool
	}{
		{"minimal", DetailMinimal, false},
		{"STANDARD", DetailStandard, false},
		{"Thorough", DetailThorough, false},
		{"", DetailStandard, false},
		{"  minimal  ", DetailMinimal, false},
		{"deep", "", true},
	}
	for _, tt := range tests {
		got, err := ParseDetailLevel(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseDetailLevel(%q) err=nil, want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseDetailLevel(%q) = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseDetailLevel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDiffBytesFor(t *testing.T) {
	cfg := &Config{MaxDiffBytes: 120_000}
	if got := cfg.DiffBytesFor(DetailMinimal); got != 30_000 {
		t.Fatalf("Minimal = %d, want 30000", got)
	}
	if got := cfg.DiffBytesFor(DetailStandard); got != 120_000 {
		t.Fatalf("Standard = %d, want 120000", got)
	}
	if got := cfg.DiffBytesFor(DetailThorough); got != 120_000 {
		t.Fatalf("Thorough = %d, want 120000", got)
	}

	tiny := &Config{MaxDiffBytes: 1000}
	if got := tiny.DiffBytesFor(DetailMinimal); got != 4096 {
		t.Fatalf("Minimal floor = %d, want 4096", got)
	}
}

func TestNormalizeDetailLevel(t *testing.T) {
	if got := NormalizeDetailLevel(""); got != DetailStandard {
		t.Fatalf("empty = %q, want standard", got)
	}
	if got := NormalizeDetailLevel("nope"); got != DetailStandard {
		t.Fatalf("unknown = %q, want standard", got)
	}
}
