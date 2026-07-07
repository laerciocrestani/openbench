package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteBannerContainsTitleAndVersion(t *testing.T) {
	var buf bytes.Buffer
	writeBanner(&buf, false, nil, func(text, _ string) string { return text })

	out := buf.String()
	if !strings.Contains(out, "██████╗") {
		t.Fatalf("banner missing title: %q", out)
	}
	if !strings.Contains(out, "AI-powered Git Workflow") {
		t.Fatalf("banner missing tagline: %q", out)
	}
	if !strings.Contains(out, "●──────────────●") {
		t.Fatalf("banner missing logo: %q", out)
	}
}

func TestWriteBannerContext(t *testing.T) {
	var buf bytes.Buffer
	ctx := &BannerContext{
		Repo:     "gitai",
		Branch:   "main",
		Sync:     "in sync",
		Provider: "gemini",
		Model:    "gemini-2.5-flash-lite",
	}
	writeBanner(&buf, false, ctx, func(text, _ string) string { return text })

	out := buf.String()
	if !strings.Contains(out, "gitai · main · in sync") {
		t.Fatalf("banner missing repo status: %q", out)
	}
	if !strings.Contains(out, "Provider: gemini · Model: gemini-2.5-flash-lite") {
		t.Fatalf("banner missing provider line: %q", out)
	}
}

func TestWriteBannerDryRun(t *testing.T) {
	var buf bytes.Buffer
	writeBanner(&buf, true, nil, func(text, _ string) string { return text })

	if !strings.Contains(buf.String(), "dry-run") {
		t.Fatalf("banner missing dry-run mode: %q", buf.String())
	}
}
