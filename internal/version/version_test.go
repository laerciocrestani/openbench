package version

import "testing"

func TestBumpPatch(t *testing.T) {
	tests := []struct {
		base  string
		extra int
		want  string
	}{
		{"v0.1.0", 0, "v0.1.0"},
		{"v0.1.0", 3, "v0.1.3"},
		{"v0.1.0", 12, "v0.1.12"},
		{"v1.2.5", 1, "v1.2.6"},
	}
	for _, tc := range tests {
		got, err := bumpPatch(tc.base, tc.extra)
		if err != nil {
			t.Fatalf("bumpPatch(%q, %d): %v", tc.base, tc.extra, err)
		}
		if got != tc.want {
			t.Errorf("bumpPatch(%q, %d) = %q, want %q", tc.base, tc.extra, got, tc.want)
		}
	}
}

func TestInfoDisplay(t *testing.T) {
	info := Info{Version: "v0.1.12", Commit: "1bbc815"}
	if info.Display() != "v0.1.12 · 1bbc815" {
		t.Errorf("display = %q", info.Display())
	}

	plain := Info{Version: "v0.1.0"}
	if plain.Display() != "v0.1.0" {
		t.Errorf("plain display = %q", plain.Display())
	}
}

func TestParseSemver(t *testing.T) {
	major, minor, patch, err := parseSemver("v0.1.0")
	if err != nil || major != 0 || minor != 1 || patch != 0 {
		t.Fatalf("parseSemver failed: %d.%d.%d %v", major, minor, patch, err)
	}
}

func TestSemverPrefersBuildVersion(t *testing.T) {
	prev := BuildVersion
	t.Cleanup(func() { BuildVersion = prev })

	BuildVersion = ""
	// Without BuildVersion, Semver falls back to git or DefaultBase — just ensure no panic.
	_ = Semver()

	BuildVersion = "0.2.1"
	if got := Semver(); got != "0.2.1" {
		t.Fatalf("Semver() = %q, want 0.2.1", got)
	}
	if got := DisplayCurrent(); got != "v0.2.1" {
		t.Fatalf("DisplayCurrent() = %q, want v0.2.1", got)
	}

	BuildVersion = "v0.3.0"
	if got := Semver(); got != "0.3.0" {
		t.Fatalf("Semver() = %q, want 0.3.0", got)
	}
}
