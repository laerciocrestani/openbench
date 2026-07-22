package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/app"
)

// DoctorIssueView is one health finding for the desktop UI.
type DoctorIssueView struct {
	Level  string `json:"level"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// DoctorAIView is the optional AI explanation.
type DoctorAIView struct {
	Summary  string   `json:"summary"`
	Cause    string   `json:"cause"`
	Risk     string   `json:"risk"`
	Steps    []string `json:"steps"`
	Warnings []string `json:"warnings"`
}

// DoctorView is the desktop DTO for repository health.
type DoctorView struct {
	Overall         string            `json:"overall"`
	Branch          string            `json:"branch"`
	Base            string            `json:"base"`
	Issues          []DoctorIssueView `json:"issues"`
	Recommendations []string          `json:"recommendations"`
	Lines           []string          `json:"lines"`
	Explained       bool              `json:"explained"`
	AI              *DoctorAIView     `json:"ai,omitempty"`
	ExplainError    string            `json:"explainError,omitempty"`
}

// RunDoctor analyzes repository health for the open project.
// When explain is true, consults AI (requires API key).
func RunDoctor(ctx context.Context, projectPath string, explain bool, base string) (*DoctorView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	report, err := app.RunDoctor(ctx, app.DoctorOptions{
		Explain:  explain,
		Base:     base,
		WorkDir:  projectPath,
		Progress: app.NopProgress(),
	})
	if err != nil {
		if report == nil {
			return nil, err
		}
		// Base analysis succeeded; AI explain (or a late step) failed.
		view := mapDoctorView(report, false)
		view.ExplainError = err.Error()
		return view, nil
	}
	return mapDoctorView(report, explain), nil
}

func mapDoctorView(report *app.DoctorReport, explained bool) *DoctorView {
	if report == nil {
		return nil
	}
	view := &DoctorView{
		Overall:         string(report.Overall),
		Branch:          report.Branch,
		Base:            report.Base,
		Recommendations: append([]string{}, report.Recommendations...),
		Lines:           append([]string{}, report.Lines...),
		Explained:       explained && report.AI != nil,
		Issues:          make([]DoctorIssueView, 0, len(report.Issues)),
	}
	if view.Overall == "" {
		view.Overall = "ok"
	}
	for _, issue := range report.Issues {
		view.Issues = append(view.Issues, DoctorIssueView{
			Level:  issue.Level,
			Code:   issue.Code,
			Title:  issue.Title,
			Detail: issue.Detail,
		})
	}
	if report.AI != nil {
		view.AI = &DoctorAIView{
			Summary:  report.AI.Summary,
			Cause:    report.AI.Cause,
			Risk:     report.AI.Risk,
			Steps:    append([]string{}, report.AI.Steps...),
			Warnings: append([]string{}, report.AI.Warnings...),
		}
	}
	return view
}
