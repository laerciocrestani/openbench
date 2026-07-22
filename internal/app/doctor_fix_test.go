package app

import (
	"errors"
	"strings"
	"testing"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

func TestSuggestDoctorBranchName(t *testing.T) {
	exists := map[string]bool{"feature/chat-2": true}
	got := SuggestDoctorBranchName("feature/chat", "main", func(n string) bool { return exists[n] })
	if got != "feature/chat-3" {
		t.Fatalf("got %q want feature/chat-3", got)
	}
}

func TestBuildDoctorFixPlan_workOnMergedBranch(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:    "feature/chat",
		Base:      "main",
		OnBase:    false,
		IsDirty:   true,
		Modified:  2,
		Untracked: 1,
	}
	issues := analyzeHealthIssues(snap, &prpkg.PRView{Number: 9, State: "MERGED"})
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{})
	if !plan.NeedsBranchName {
		t.Fatal("expected NeedsBranchName")
	}
	if plan.SuggestedBranch != "feature/chat-2" {
		t.Fatalf("suggested branch: %q", plan.SuggestedBranch)
	}
	kinds := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		kinds = append(kinds, s.Kind)
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, DoctorStepStashPush) || !strings.Contains(joined, DoctorStepCreateBranch) {
		t.Fatalf("expected stash+create_branch, got %v", kinds)
	}
	for _, s := range plan.Steps {
		if s.Kind == DoctorStepPullFF {
			t.Fatal("merged flow should not pull current feature branch")
		}
	}
}

func TestBuildDoctorFixPlan_behindRemote(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:  "feature/x",
		Base:    "main",
		Behind:  3,
		IsDirty: false,
	}
	issues := analyzeHealthIssues(snap, nil)
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{})
	if !plan.CanAutoFix || len(plan.Steps) == 0 {
		t.Fatalf("expected autofix plan, got %+v", plan)
	}
	found := false
	for _, s := range plan.Steps {
		if s.Kind == DoctorStepPullFF {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected pull_ff, steps=%v", plan.Steps)
	}
}

func TestManualHintConflict(t *testing.T) {
	hint := manualHintForStep(DoctorStepStashPop, "main", "feature/x", errors.New("CONFLICT (content)"))
	if !strings.Contains(strings.ToLower(hint), "resolva") {
		t.Fatalf("hint=%q", hint)
	}
}
