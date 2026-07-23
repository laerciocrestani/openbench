package desktop

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/app"
)

// DoctorFixStepView is one planned/executed step for the UI timeline.
type DoctorFixStepView struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Command    string   `json:"command"`
	Risk       string   `json:"risk"`
	Status     string   `json:"status,omitempty"`
	Detail     string   `json:"detail,omitempty"`
	ManualHint string   `json:"manualHint,omitempty"`
	IssueCodes []string `json:"issueCodes,omitempty"`
}

// DoctorFixPlanView is the preview dialog payload.
type DoctorFixPlanView struct {
	Steps                   []DoctorFixStepView `json:"steps"`
	SuggestedBranch         string              `json:"suggestedBranch"`
	NeedsBranchName         bool                `json:"needsBranchName"`
	NeedsBaseAction         bool                `json:"needsBaseAction"`
	BaseActionOptions       []string            `json:"baseActionOptions,omitempty"`
	SuggestedBaseAction     string              `json:"suggestedBaseAction,omitempty"`
	NeedsMergedAction       bool                `json:"needsMergedAction"`
	MergedActionOptions     []string            `json:"mergedActionOptions,omitempty"`
	SuggestedMergedAction   string              `json:"suggestedMergedAction,omitempty"`
	NeedsDestructiveConfirm bool                `json:"needsDestructiveConfirm"`
	Summary                 string              `json:"summary"`
	Warnings                []string            `json:"warnings,omitempty"`
	CanAutoFix              bool                `json:"canAutoFix"`
	BlockReason             string              `json:"blockReason,omitempty"`
	IssueCodes              []string            `json:"issueCodes,omitempty"`
	Branch                  string              `json:"branch"`
	Base                    string              `json:"base"`
}

// DoctorFixResultView is returned after execution.
type DoctorFixResultView struct {
	OK        bool                `json:"ok"`
	Message   string              `json:"message"`
	Steps     []DoctorFixStepView `json:"steps"`
	Dashboard *Dashboard          `json:"dashboard,omitempty"`
}

// DoctorFixAdvanceView is one step result from AdvanceDoctorFix.
type DoctorFixAdvanceView struct {
	Step      DoctorFixStepView `json:"step"`
	Done      bool              `json:"done"`
	OK        bool              `json:"ok"`
	Message   string            `json:"message,omitempty"`
	Dashboard *Dashboard        `json:"dashboard,omitempty"`
}

// DoctorFixSession holds an in-progress step-by-step doctor fix.
type DoctorFixSession struct {
	Path   string
	Runner *app.DoctorFixRunner
}

// PlanDoctorFix builds the remediation preview for the open project.
func PlanDoctorFix(projectPath, base, newBranch, baseAction, mergedAction string) (*DoctorFixPlanView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	plan, err := app.PlanDoctorFix(app.DoctorFixOptions{
		WorkDir:       projectPath,
		Base:          base,
		NewBranchName: newBranch,
		BaseAction:    baseAction,
		MergedAction:  mergedAction,
	})
	if err != nil {
		return nil, err
	}
	return mapDoctorFixPlan(plan), nil
}

// BeginDoctorFix prepares a step-by-step execution session.
func BeginDoctorFix(
	projectPath, base, newBranch, baseAction, mergedAction string,
	confirmDestructive bool,
) (*DoctorFixPlanView, *DoctorFixSession, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, nil, fmt.Errorf("no project open")
	}
	runner, plan, err := app.NewDoctorFixRunner(app.DoctorFixOptions{
		WorkDir:            projectPath,
		Base:               base,
		NewBranchName:      newBranch,
		BaseAction:         baseAction,
		MergedAction:       mergedAction,
		ConfirmDestructive: confirmDestructive,
	})
	if err != nil {
		return mapDoctorFixPlan(plan), nil, err
	}
	return mapDoctorFixPlan(plan), &DoctorFixSession{Path: projectPath, Runner: runner}, nil
}

// AdvanceDoctorFixSession runs the next step of an active session.
func AdvanceDoctorFixSession(sess *DoctorFixSession) (*DoctorFixAdvanceView, error) {
	if sess == nil || sess.Runner == nil {
		return nil, fmt.Errorf("nenhuma execução do Doctor em andamento — confirme de novo")
	}
	sr, done, err := sess.Runner.Next()
	if err != nil {
		return nil, err
	}
	out := &DoctorFixAdvanceView{
		Step: DoctorFixStepView{
			ID:         sr.ID,
			Kind:       sr.Kind,
			Title:      sr.Title,
			Command:    sr.Command,
			Risk:       sr.Risk,
			Status:     sr.Status,
			Detail:     sr.Detail,
			ManualHint: sr.ManualHint,
		},
		Done: done,
		OK:   sr.Status != "error",
	}
	if sr.Status == "error" {
		out.OK = false
		out.Done = true
		out.Message = fmt.Sprintf("Parou em: %s", sr.Title)
		return out, nil
	}
	if done {
		out.Message = "Ajuste concluído"
		if dash, dashErr := LoadDashboard(sess.Path); dashErr == nil {
			out.Dashboard = dash
		}
	}
	return out, nil
}

func mapDoctorFixPlan(plan *app.DoctorFixPlan) *DoctorFixPlanView {
	if plan == nil {
		return nil
	}
	view := &DoctorFixPlanView{
		SuggestedBranch:         plan.SuggestedBranch,
		NeedsBranchName:         plan.NeedsBranchName,
		NeedsBaseAction:         plan.NeedsBaseAction,
		BaseActionOptions:       append([]string{}, plan.BaseActionOptions...),
		SuggestedBaseAction:     plan.SuggestedBaseAction,
		NeedsMergedAction:       plan.NeedsMergedAction,
		MergedActionOptions:     append([]string{}, plan.MergedActionOptions...),
		SuggestedMergedAction:   plan.SuggestedMergedAction,
		NeedsDestructiveConfirm: plan.NeedsDestructiveConfirm,
		Summary:                 plan.Summary,
		Warnings:                append([]string{}, plan.Warnings...),
		CanAutoFix:              plan.CanAutoFix,
		BlockReason:             plan.BlockReason,
		IssueCodes:              append([]string{}, plan.IssueCodes...),
		Branch:                  plan.Branch,
		Base:                    plan.Base,
		Steps:                   make([]DoctorFixStepView, 0, len(plan.Steps)),
	}
	for _, s := range plan.Steps {
		view.Steps = append(view.Steps, DoctorFixStepView{
			ID:         s.ID,
			Kind:       s.Kind,
			Title:      s.Title,
			Command:    s.Command,
			Risk:       s.Risk,
			Status:     "pending",
			IssueCodes: append([]string{}, s.IssueCodes...),
		})
	}
	return view
}
