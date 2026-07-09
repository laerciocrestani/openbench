package git

import "testing"

func TestParseBranchVVLine_goneUpstream(t *testing.T) {
	name, tracking, ok := parseBranchVVLine("  feature/foo  abc1234 [origin/feature/foo: gone] message")
	if !ok {
		t.Fatal("expected ok")
	}
	if name != "feature/foo" {
		t.Fatalf("name = %q", name)
	}
	if tracking != "origin/feature/foo: gone" {
		t.Fatalf("tracking = %q", tracking)
	}
	if !isGoneUpstream(tracking) {
		t.Fatal("expected gone upstream")
	}
}

func TestParseBranchVVLine_currentBranch(t *testing.T) {
	name, tracking, ok := parseBranchVVLine("* main  abc1234 [origin/main] message")
	if !ok {
		t.Fatal("expected ok")
	}
	if name != "main" {
		t.Fatalf("name = %q", name)
	}
	if tracking != "origin/main" {
		t.Fatalf("tracking = %q", tracking)
	}
	if isGoneUpstream(tracking) {
		t.Fatal("did not expect gone upstream")
	}
}

func TestParseBranchVVLine_noUpstream(t *testing.T) {
	name, tracking, ok := parseBranchVVLine("  wip  abc1234 wip work")
	if !ok {
		t.Fatal("expected ok")
	}
	if name != "wip" {
		t.Fatalf("name = %q", name)
	}
	if tracking != "" {
		t.Fatalf("tracking = %q, want empty", tracking)
	}
}

func TestIsGoneUpstreamTrack(t *testing.T) {
	cases := []struct {
		track string
		want  bool
	}{
		{"[gone]", true},
		{"origin/feature/foo: gone", true},
		{"", false},
		{"ahead 1", false},
	}
	for _, tc := range cases {
		if got := isGoneUpstreamTrack(tc.track); got != tc.want {
			t.Fatalf("track %q: got %v want %v", tc.track, got, tc.want)
		}
	}
}
