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
	got = SuggestDoctorBranchName("main", "main", func(n string) bool { return exists[n] })
	if got != "feature/ajuste" {
		t.Fatalf("on base got %q want feature/ajuste", got)
	}
}

func TestBuildDoctorFixPlan_baseLocalAheadPrefersFeatureBranch(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:  "main",
		Base:    "main",
		OnBase:  true,
		IsDirty: true,
		Staged:  1,
		Modified: 13,
		BaseDivergence: &gitpkg.DivergenceReport{
			LocalRef:    "main",
			RemoteRef:   "origin/main",
			LocalAhead:  4,
			RemoteAhead: 0,
			LocalAnalyses: []gitpkg.CommitAnalysis{
				{Hash: "abc", Subject: "feat: cobrança", LikelyDiscardable: false},
			},
		},
	}
	issues := analyzeHealthIssues(snap, nil)
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{})
	if plan.SuggestedBaseAction != BaseActionBranch {
		t.Fatalf("suggested action=%q want %q", plan.SuggestedBaseAction, BaseActionBranch)
	}
	if !plan.NeedsBranchName {
		t.Fatal("expected NeedsBranchName for branch move")
	}
	kinds := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		kinds = append(kinds, s.Kind)
		if s.Kind == DoctorStepStashPush || s.Kind == DoctorStepStashPop {
			t.Fatalf("branch move must not stash dirty tree: %v", kinds)
		}
		if s.Kind == DoctorStepPushBase {
			t.Fatalf("default must not push main: %v", kinds)
		}
	}
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, DoctorStepCreateBranch) || !strings.Contains(joined, DoctorStepResetBase) {
		t.Fatalf("expected create_branch + reset_base, got %v", kinds)
	}

	pushPlan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{BaseAction: BaseActionPush})
	foundPush := false
	for _, s := range pushPlan.Steps {
		if s.Kind == DoctorStepPushBase {
			foundPush = true
		}
		if s.Kind == DoctorStepStashPush || s.Kind == DoctorStepStashPop {
			t.Fatalf("push with dirty tree must not stash: steps=%v", stepKindsOf(pushPlan))
		}
	}
	if !foundPush {
		t.Fatalf("explicit push action should push base, steps=%v", stepKindsOf(pushPlan))
	}
}

func stepKindsOf(plan *DoctorFixPlan) []string {
	out := make([]string, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		out = append(out, s.Kind)
	}
	return out
}

func TestBuildDoctorFixPlan_dirtyOnBaseMovesToFeature(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:   "main",
		Base:     "main",
		OnBase:   true,
		IsDirty:  true,
		Modified: 2,
	}
	issues := analyzeHealthIssues(snap, nil)
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{})
	if !plan.CanAutoFix || !plan.NeedsBranchName {
		t.Fatalf("expected autofix with branch name, got %+v", plan)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Kind != DoctorStepCreateBranch {
		t.Fatalf("expected single create_branch, got %+v", plan.Steps)
	}
}

func TestBuildHealthRecommendations_baseLocalAheadNoDirectPushPrimary(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:  "main",
		Base:    "main",
		OnBase:  true,
		IsDirty: true,
		BaseDivergence: &gitpkg.DivergenceReport{
			LocalRef:    "main",
			RemoteRef:   "origin/main",
			LocalAhead:  4,
			RemoteAhead: 0,
			LocalAnalyses: []gitpkg.CommitAnalysis{
				{Hash: "abc", Subject: "feat: x", LikelyDiscardable: false},
			},
		},
	}
	issues := analyzeHealthIssues(snap, nil)
	recs := buildHealthRecommendations(snap, issues, nil)
	joined := strings.ToLower(strings.Join(recs, "\n"))
	if strings.Contains(joined, "commit as alterações nesta branch") {
		t.Fatalf("must not recommend commit-on-main: %v", recs)
	}
	if !strings.Contains(joined, "feature branch") {
		t.Fatalf("expected feature branch guidance, got %v", recs)
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

func TestManualHintPushLargeFile(t *testing.T) {
	err := errors.New("GH001: Large files detected. File public/TeamViewer.dmg is 105.43 MB; this exceeds GitHub's file size limit of 100.00 MB")
	hint := manualHintForStep(DoctorStepPushBase, "main", "", err)
	low := strings.ToLower(hint)
	if !strings.Contains(low, "limpeza") {
		t.Fatalf("expected cleanup guidance, got %q", hint)
	}
}

func TestBuildDoctorFixPlan_unpushedBlockedPrefersCleanup(t *testing.T) {
	snap := &gitpkg.HealthSnapshot{
		Branch:  "main",
		Base:    "main",
		OnBase:  true,
		IsDirty: true,
		BaseDivergence: &gitpkg.DivergenceReport{
			LocalRef:    "main",
			RemoteRef:   "origin/main",
			LocalAhead:  4,
			RemoteAhead: 0,
			LocalAnalyses: []gitpkg.CommitAnalysis{
				{Hash: "abc", Subject: "chore: ignore", LikelyDiscardable: false},
			},
		},
		UnpushedBlockers: &gitpkg.UnpushedBlockers{
			LargeFiles: []gitpkg.LargeFileHit{{Path: "public/TeamViewer.dmg", Size: 110 * 1024 * 1024}},
			JunkPaths:  []string{"node_modules", ".env", "vendor"},
		},
	}
	issues := analyzeHealthIssues(snap, nil)
	foundBlocked := false
	for _, iss := range issues {
		if iss.Code == "unpushed_blocked" {
			foundBlocked = true
		}
	}
	if !foundBlocked {
		t.Fatalf("expected unpushed_blocked issue, got %#v", issues)
	}
	plan := buildDoctorFixPlan(nil, snap, issues, DoctorFixOptions{})
	if plan.SuggestedBaseAction != BaseActionCleanup {
		t.Fatalf("suggested=%q want cleanup", plan.SuggestedBaseAction)
	}
	kinds := stepKindsOf(plan)
	joined := strings.Join(kinds, ",")
	if !strings.Contains(joined, DoctorStepMixedReset) {
		t.Fatalf("expected mixed_reset, got %v", kinds)
	}
	if strings.Contains(joined, DoctorStepSoftReset) || strings.Contains(joined, DoctorStepUnstageBlockers) {
		t.Fatalf("cleanup must use mixed reset only, got %v", kinds)
	}
	if strings.Contains(joined, DoctorStepPushBase) {
		t.Fatalf("default must not push when blocked: %v", kinds)
	}
	if strings.Contains(joined, DoctorStepStashPush) || strings.Contains(joined, DoctorStepStashPop) {
		t.Fatalf("cleanup must not stash (index.lock races): %v", kinds)
	}
}
