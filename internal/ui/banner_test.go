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
	if !strings.Contains(out, "┏━┓") {
		t.Fatalf("banner missing title: %q", out)
	}
	if !strings.Contains(out, "●──●") {
		t.Fatalf("banner missing git graph: %q", out)
	}
	if idx := strings.Index(out, "┏━┓"); idx < 0 || idx > strings.Index(out, "●──●") {
		t.Fatalf("title should appear before graph: %q", out)
	}
	if !strings.Contains(out, "AI-powered Git Workflow") {
		t.Fatalf("banner missing tagline: %q", out)
	}
	if strings.Count(out, "\n") > 12 {
		t.Fatalf("banner too tall: %d lines", strings.Count(out, "\n"))
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

func TestBannerTitleStyleFade(t *testing.T) {
	total := len(bannerTitle)
	if len(bannerGraph) != total {
		t.Fatalf("bannerGraph height = %d, want %d", len(bannerGraph), total)
	}
	first := bannerTitleStyle(0)
	last := bannerTitleStyle(total - 1)
	if first == last {
		t.Fatal("first and last title lines should use different fade colors")
	}
	if !strings.Contains(first, ";0;255;0m") {
		t.Fatalf("first line should be full green, got %q", first)
	}
	if !strings.Contains(last, ";0;25;0m") {
		t.Fatalf("last line should fade to 10%% green, got %q", last)
	}
	for i := 0; i < total; i++ {
		if bannerTitleStyle(i) == "" {
			t.Fatalf("line %d has empty style", i)
		}
	}
}

func TestBannerGraphLastDotAccent(t *testing.T) {
	var buf bytes.Buffer
	writeBanner(&buf, false, nil, func(text, code string) string {
		if code == bold+magenta {
			return "PINK:" + text
		}
		return text
	})
	if !strings.Contains(buf.String(), "PINK:●") {
		t.Fatalf("last graph dot should use accent color: %q", buf.String())
	}
}

func TestWriteBannerMismatchedHeights(t *testing.T) {
	origTitle := bannerTitle
	origGraph := bannerGraph
	t.Cleanup(func() {
		bannerTitle = origTitle
		bannerGraph = origGraph
	})

	bannerTitle = []string{"a", "b", "c", "d"}
	bannerGraph = []string{"1", "2", "3"}

	var buf bytes.Buffer
	writeBanner(&buf, false, nil, func(text, _ string) string { return text })

	if !strings.Contains(buf.String(), "a") || !strings.Contains(buf.String(), "3") {
		t.Fatalf("expected banner with partial art: %q", buf.String())
	}
}

func TestWriteBannerDryRun(t *testing.T) {
	var buf bytes.Buffer
	writeBanner(&buf, true, nil, func(text, _ string) string { return text })

	if !strings.Contains(buf.String(), "dry-run") {
		t.Fatalf("banner missing dry-run mode: %q", buf.String())
	}
}
