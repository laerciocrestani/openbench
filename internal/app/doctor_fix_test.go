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
	if !plan.NeedsMergedAction {
		t.Fatal("expected NeedsMergedAction")
	}
	if plan.SuggestedMergedAction != MergedActionContinue {
		t.Fatalf("dirty merged default: %q", plan.SuggestedMergedAction)
	}
	if !plan.NeedsBranchName {
		t.Fatal("expected NeedsBranchName for continue")
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

func TestBuildDoctorFixPlan_workOnMergedBranch_returnBase(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:  "feature/chat-5",
		Base:    "main",
		OnBase:  false,
		IsDirty: false,
	}
	issues := analyzeHealthIssues(snap, &prpkg.PRView{Number: 13, State: "MERGED"})
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{})
	if plan.SuggestedMergedAction != MergedActionReturnBase {
		t.Fatalf("clean merged default: %q want %q", plan.SuggestedMergedAction, MergedActionReturnBase)
	}
	if plan.NeedsBranchName {
		t.Fatal("return_base must not require branch name")
	}
	kinds := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		kinds = append(kinds, s.Kind)
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, DoctorStepCheckout) || strings.Contains(joined, DoctorStepCreateBranch) {
		t.Fatalf("expected checkout base without create_branch, got %v", kinds)
	}

	cont := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{MergedAction: MergedActionContinue})
	if !cont.NeedsBranchName {
		t.Fatal("continue requires branch name")
	}
	foundCreate := false
	for _, s := range cont.Steps {
		if s.Kind == DoctorStepCreateBranch {
			foundCreate = true
		}
	}
	if !foundCreate {
		t.Fatalf("continue should create branch, steps=%v", cont.Steps)
	}
}

func TestBuildDoctorFixPlan_returnBaseDirtyKeepsStash(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:   "feature/chat-5",
		Base:     "main",
		OnBase:   false,
		IsDirty:  true,
		Modified: 3,
	}
	issues := analyzeHealthIssues(snap, &prpkg.PRView{Number: 13, State: "MERGED"})
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{MergedAction: MergedActionReturnBase})
	if plan.NeedsBranchName {
		t.Fatal("return_base must not require branch name")
	}
	kinds := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		kinds = append(kinds, s.Kind)
		if s.Kind == DoctorStepStashPop {
			t.Fatal("return_base must not stash-pop onto main")
		}
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, DoctorStepStashPush) || !strings.Contains(joined, DoctorStepCheckout) {
		t.Fatalf("expected stash+checkout, got %v", kinds)
	}
	foundWIPWarn := false
	for _, w := range plan.Warnings {
		if strings.Contains(w, "stash") {
			foundWIPWarn = true
			break
		}
	}
	if !foundWIPWarn {
		t.Fatalf("expected stash warning, got %v", plan.Warnings)
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
