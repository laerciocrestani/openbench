package app

import (
	"strings"
	"testing"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

func TestAnalyzeHealthIssues_baseDivergedWithBuildArtifacts(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch: "main",
		Base:   "main",
		OnBase: true,
		BaseDivergence: &gitpkg.DivergenceReport{
			LocalRef:    "main",
			RemoteRef:   "origin/main",
			MergeBase:   "907f5954abc",
			LocalAhead:  2,
			RemoteAhead: 7,
			LocalCommits: []string{
				"ab0a7ef Update dependencies",
				"d42b5c6 data-store",
			},
			LocalAnalyses: []gitpkg.CommitAnalysis{
				{Hash: "ab0a7ef", Subject: "Update dependencies", FileCount: 9800, BuildArtifactFiles: 9500, LikelyDiscardable: true},
				{Hash: "d42b5c6", Subject: "data-store", FileCount: 1, BuildArtifactFiles: 1, LikelyDiscardable: false},
			},
		},
	}

	issues := analyzeHealthIssues(snap, nil)
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}

	foundBase := false
	foundBuild := false
	for _, issue := range issues {
		if issue.Code == "base_diverged" {
			foundBase = true
		}
		if issue.Code == "build_artifacts" {
			foundBuild = true
		}
	}
	if !foundBase {
		t.Fatal("expected base_diverged issue")
	}
	if !foundBuild {
		t.Fatal("expected build_artifacts issue")
	}

	recs := buildHealthRecommendations(snap, issues, nil)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}
}

func TestAnalyzeHealthIssues_workOnMergedBranch(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:    "feature/chat",
		Base:      "main",
		OnBase:    false,
		IsDirty:   true,
		Modified:  2,
		Untracked: 1,
	}
	pr := &prpkg.PRView{Number: 9, Title: "done", State: "MERGED"}
	issues := analyzeHealthIssues(snap, pr)
	found := false
	for _, issue := range issues {
		if issue.Code == "work_on_merged_branch" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected work_on_merged_branch")
	}
	recs := buildHealthRecommendations(snap, issues, pr)
	joined := strings.Join(recs, "\n")
	if !strings.Contains(joined, "NOVA feature") && !strings.Contains(strings.ToLower(joined), "nova feature") {
		t.Fatalf("expected new-branch guidance, got %v", recs)
	}
	for _, rec := range recs {
		if strings.Contains(rec, "ob commit") {
			t.Fatalf("should not push commit-on-same-branch as primary path when PR is merged: %v", recs)
		}
	}
}

func TestBuildHealthRecommendations_dirtyOnlyPrefersCommit(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:   "feature/chat-2",
		Base:     "main",
		OnBase:   false,
		IsDirty:  true,
		Modified: 3,
	}
	issues := analyzeHealthIssues(snap, nil)
	recs := buildHealthRecommendations(snap, issues, nil)
	joined := strings.ToLower(strings.Join(recs, "\n"))
	if !strings.Contains(joined, "commit") {
		t.Fatalf("expected commit guidance, got %v", recs)
	}
	for _, rec := range recs {
		low := strings.ToLower(rec)
		if strings.Contains(low, "stash") {
			t.Fatalf("dirty-only should not recommend stash as next step: %v", recs)
		}
	}
}

func TestBuildDoctorFixPlan_dirtyOnlyNoStashPop(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:  "feature/chat-2",
		Base:    "main",
		OnBase:  false,
		IsDirty: true,
	}
	issues := analyzeHealthIssues(snap, nil)
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{})
	if plan.CanAutoFix {
		t.Fatal("dirty-only should not auto-fix via stash")
	}
	if !strings.Contains(strings.ToLower(plan.BlockReason), "commit") {
		t.Fatalf("block reason should point to commit, got %q", plan.BlockReason)
	}
	for _, s := range plan.Steps {
		if s.Kind == DoctorStepStashPush || s.Kind == DoctorStepStashPop {
			t.Fatalf("unexpected stash step in dirty-only plan: %+v", plan.Steps)
		}
	}
}

func TestOverallHealth_clean(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch: "feature/x",
		Base:   "main",
	}
	level := overallHealth(analyzeHealthIssues(snap, nil), snap)
	if level != gitpkg.HealthOK {
		t.Fatalf("expected ok, got %s", level)
	}
}

func TestIsFastForwardError(t *testing.T) {
	if !isFastForwardError(fmtError("fatal: Not possible to fast-forward, aborting.")) {
		t.Fatal("expected fast-forward detection")
	}
}

func fmtError(msg string) error {
	return &wrappedError{msg: msg}
}

type wrappedError struct{ msg string }

func (e *wrappedError) Error() string { return e.msg }
