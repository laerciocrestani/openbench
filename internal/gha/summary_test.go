package gha

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSummaryFromErrorUsesCacheOffline(t *testing.T) {
	cached := &Summary{State: SummaryPass, Label: "CI 2✓", Pass: 2}
	got := SummaryFromError("/tmp/repo", &Error{Kind: ErrNetwork, Message: "net"}, cached)
	if got == nil {
		t.Fatal("expected summary")
	}
	if !got.FromCache {
		t.Fatal("expected FromCache")
	}
	if got.State != SummaryOffline {
		t.Fatalf("state=%q want offline", got.State)
	}
	if !strings.Contains(got.Label, "off") {
		t.Fatalf("label=%q", got.Label)
	}
}

func TestSummaryFromErrorAuthUnavailable(t *testing.T) {
	got := SummaryFromError("/tmp/repo", &Error{Kind: ErrGhAuth, Message: "auth"}, nil)
	if got.State != SummaryUnavailable {
		t.Fatalf("state=%q", got.State)
	}
	if !errors.Is(&Error{Kind: ErrGhAuth}, ErrGhAuth) {
		t.Fatal("Error.Is should match Kind")
	}
}

func TestSummaryFromErrorCachedAuthKeepsLabel(t *testing.T) {
	cached := &Summary{State: SummaryFail, Label: "CI 1✗", Fail: 1}
	got := SummaryFromError("/tmp/repo", &Error{Kind: ErrGhAuth, Message: "auth"}, cached)
	if !got.FromCache {
		t.Fatal("expected cache")
	}
	if got.Label != "CI 1✗" {
		t.Fatalf("label=%q", got.Label)
	}
}

func TestLatestRunsPerWorkflowIgnoresOlderFailure(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	runs := []WorkflowRun{
		{ID: 9, WorkflowID: 1, WorkflowName: "Deploy Cloudflare", Conclusion: "success", Status: "completed", CreatedAt: now},
		{ID: 8, WorkflowID: 2, WorkflowName: "Deploy DigitalOcean", Conclusion: "success", Status: "completed", CreatedAt: now.Add(-time.Minute)},
		{ID: 7, WorkflowID: 1, WorkflowName: "Deploy Cloudflare", Conclusion: "failure", Status: "completed", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 6, WorkflowID: 2, WorkflowName: "Deploy DigitalOcean", Conclusion: "success", Status: "completed", CreatedAt: now.Add(-3 * time.Hour)},
	}
	latest := latestRunsPerWorkflow(runs)
	if len(latest) != 2 {
		t.Fatalf("latest=%d want 2: %#v", len(latest), latest)
	}
	sum := FinalizeSummary(&Summary{Host: "github.com"}, latest)
	if sum.Fail != 0 || sum.Pass != 2 || sum.State != SummaryPass {
		t.Fatalf("sum=%#v", sum)
	}
	if sum.Label != "CI 2✓" {
		t.Fatalf("label=%q", sum.Label)
	}
}

func TestLatestRunsPerWorkflowKeepsCurrentFailure(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	runs := []WorkflowRun{
		{ID: 3, WorkflowID: 1, WorkflowName: "Deploy Cloudflare", Conclusion: "failure", Status: "completed", CreatedAt: now},
		{ID: 2, WorkflowID: 1, WorkflowName: "Deploy Cloudflare", Conclusion: "success", Status: "completed", CreatedAt: now.Add(-time.Hour)},
		{ID: 1, WorkflowID: 2, WorkflowName: "Deploy DigitalOcean", Conclusion: "success", Status: "completed", CreatedAt: now.Add(-2 * time.Minute)},
	}
	sum := FinalizeSummary(&Summary{}, latestRunsPerWorkflow(runs))
	if sum.Fail != 1 || sum.Pass != 1 || sum.State != SummaryFail {
		t.Fatalf("sum=%#v", sum)
	}
	if sum.Label != "CI 1✗" {
		t.Fatalf("label=%q", sum.Label)
	}
}
