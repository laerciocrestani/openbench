package desktop

import (
	"fmt"
	"strings"
	"time"

	"github.com/laerciocrestani/openbench/internal/gha"
)

// CIStatusView is the Observe-slice payload for the CI panel.
type CIStatusView struct {
	Branch  string       `json:"branch"`
	HeadSHA string       `json:"headSha,omitempty"`
	Runs    []CIRunView  `json:"runs"`
	Usage   CIUsageView  `json:"usage"`
	Filter  CIFilterView `json:"filter"`
}

// CIFilterView mirrors list filters for the UI.
type CIFilterView struct {
	Branch     string `json:"branch"`
	FailedOnly bool   `json:"failedOnly"`
	Limit      int    `json:"limit"`
}

// CIRunView is one workflow run row.
type CIRunView struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	DisplayTitle string `json:"displayTitle"`
	Event        string `json:"event"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	HeadBranch   string `json:"headBranch"`
	HeadSHA      string `json:"headSha"`
	URL          string `json:"url"`
	WorkflowName string `json:"workflowName"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	Failed       bool   `json:"failed"`
}

// CIJobView is a job inside a run.
type CIJobView struct {
	ID         int64        `json:"id"`
	Name       string       `json:"name"`
	Status     string       `json:"status"`
	Conclusion string       `json:"conclusion"`
	Failed     bool         `json:"failed"`
	Steps      []CIStepView `json:"steps"`
}

// CIStepView is a step inside a job.
type CIStepView struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Number     int    `json:"number"`
	Failed     bool   `json:"failed"`
}

// CIRunDetailView is run + jobs for drill-down.
type CIRunDetailView struct {
	Run   CIRunView   `json:"run"`
	Jobs  []CIJobView `json:"jobs"`
	Usage CIUsageView `json:"usage"`
}

// CIUsageView always accompanies CI data (ADR-008).
type CIUsageView struct {
	State               string   `json:"state"`
	RunMinutes          *float64 `json:"runMinutes,omitempty"`
	WindowMinutes       *float64 `json:"windowMinutes,omitempty"`
	OrgUsedMinutes      *float64 `json:"orgUsedMinutes,omitempty"`
	OrgIncludedMinutes  *float64 `json:"orgIncludedMinutes,omitempty"`
	OrgRemainingMinutes *float64 `json:"orgRemainingMinutes,omitempty"`
	Message             string   `json:"message"`
}

// LoadCIStatus lists Actions runs for the open project.
func LoadCIStatus(projectPath string, failedOnly bool, limit int) (*CIStatusView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	client, err := gha.Open(projectPath)
	if err != nil {
		return nil, err
	}
	snap, err := client.LoadStatus(gha.ListFilter{
		FailedOnly: failedOnly,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	return statusToView(snap), nil
}

// LoadCIRunDetail returns one run with jobs/steps.
func LoadCIRunDetail(projectPath string, runID int64) (*CIRunDetailView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	if runID <= 0 {
		return nil, fmt.Errorf("run id inválido")
	}
	client, err := gha.Open(projectPath)
	if err != nil {
		return nil, err
	}
	detail, err := client.ViewRun(runID)
	if err != nil {
		return nil, err
	}
	usage := client.UsageForRun(detail.Run, client.RemoteOwner())

	view := &CIRunDetailView{
		Run:   runToView(detail.Run),
		Usage: usageToView(usage),
	}
	for _, j := range detail.Jobs {
		jv := CIJobView{
			ID:         j.ID,
			Name:       j.Name,
			Status:     j.Status,
			Conclusion: j.Conclusion,
			Failed:     gha.Failed(j.Status, j.Conclusion),
		}
		for _, s := range j.Steps {
			jv.Steps = append(jv.Steps, CIStepView{
				Name:       s.Name,
				Status:     s.Status,
				Conclusion: s.Conclusion,
				Number:     s.Number,
				Failed:     gha.Failed(s.Status, s.Conclusion),
			})
		}
		view.Jobs = append(view.Jobs, jv)
	}
	return view, nil
}

func statusToView(snap *gha.StatusSnapshot) *CIStatusView {
	v := &CIStatusView{
		Branch:  snap.Branch,
		HeadSHA: snap.HeadSHA,
		Usage:   usageToView(snap.Usage),
		Filter: CIFilterView{
			Branch:     snap.Filter.Branch,
			FailedOnly: snap.Filter.FailedOnly,
			Limit:      snap.Filter.Limit,
		},
		Runs: []CIRunView{},
	}
	for _, r := range snap.Runs {
		v.Runs = append(v.Runs, runToView(r))
	}
	return v
}

func runToView(r gha.WorkflowRun) CIRunView {
	return CIRunView{
		ID:           r.ID,
		Name:         r.Name,
		DisplayTitle: r.DisplayTitle,
		Event:        r.Event,
		Status:       r.Status,
		Conclusion:   r.Conclusion,
		HeadBranch:   r.HeadBranch,
		HeadSHA:      r.HeadSHA,
		URL:          r.URL,
		WorkflowName: r.WorkflowName,
		CreatedAt:    formatCITime(r.CreatedAt),
		UpdatedAt:    formatCITime(r.UpdatedAt),
		Failed:       gha.Failed(r.Status, r.Conclusion),
	}
}

func usageToView(u gha.ActionsUsage) CIUsageView {
	return CIUsageView{
		State:               u.State,
		RunMinutes:          u.RunMinutes,
		WindowMinutes:       u.WindowMinutes,
		OrgUsedMinutes:      u.OrgUsedMinutes,
		OrgIncludedMinutes:  u.OrgIncludedMinutes,
		OrgRemainingMinutes: u.OrgRemainingMinutes,
		Message:             u.Message,
	}
}

func formatCITime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// CILogView is an on-demand redacted Actions log (ADR-008).
type CILogView struct {
	RunID        int64  `json:"runId"`
	JobID        int64  `json:"jobId,omitempty"`
	FailedOnly   bool   `json:"failedOnly"`
	RedactedText string `json:"redactedText"`
	Bytes        int    `json:"bytes"`
	RawBytes     int    `json:"rawBytes"`
	Truncated    bool   `json:"truncated"`
	Useful       bool   `json:"useful"`
	Message      string `json:"message,omitempty"`
}

// LoadCILog fetches workflow logs on demand and redacts secrets.
func LoadCILog(projectPath string, runID, jobID int64, failedOnly bool) (*CILogView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	if runID <= 0 {
		return nil, fmt.Errorf("run id inválido")
	}
	client, err := gha.Open(projectPath)
	if err != nil {
		return nil, err
	}
	payload, err := client.FetchLog(gha.LogOptions{
		RunID:      runID,
		JobID:      jobID,
		FailedOnly: failedOnly,
	})
	if err != nil {
		return nil, err
	}
	return &CILogView{
		RunID:        payload.RunID,
		JobID:        payload.JobID,
		FailedOnly:   payload.FailedOnly,
		RedactedText: payload.RedactedText,
		Bytes:        payload.Bytes,
		RawBytes:     payload.RawBytes,
		Truncated:    payload.Truncated,
		Useful:       payload.Useful,
		Message:      payload.Message,
	}, nil
}

// CIRerunPreviewView is shown before ConfirmCIRerun.
type CIRerunPreviewView struct {
	RunID       int64       `json:"runId"`
	JobID       int64       `json:"jobId,omitempty"`
	FailedOnly  bool        `json:"failedOnly"`
	RunName     string      `json:"runName"`
	HeadBranch  string      `json:"headBranch"`
	CostWarning string      `json:"costWarning"`
	Usage       CIUsageView `json:"usage"`
}

// CIDispatchPreviewView is shown before ConfirmCIDispatch.
type CIDispatchPreviewView struct {
	Workflow    string                 `json:"workflow"`
	Ref         string                 `json:"ref,omitempty"`
	Fields      map[string]string      `json:"fields,omitempty"`
	Inputs      []CIDispatchInputView  `json:"inputs,omitempty"`
	CostWarning string                 `json:"costWarning"`
	CanDispatch bool                   `json:"canDispatch"`
	Usage       CIUsageView            `json:"usage"`
}

// CIDispatchInputView is one workflow_dispatch input for the confirm form.
type CIDispatchInputView struct {
	ID          string   `json:"id"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type,omitempty"`
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// CIWorkflowView is one workflow for dispatch UI.
type CIWorkflowView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	State       string `json:"state"`
	CanDispatch bool   `json:"canDispatch"`
}

// PreviewCIRerun prepares a re-run confirmation (no mutation).
func PreviewCIRerun(projectPath string, runID, jobID int64, failedOnly bool) (*CIRerunPreviewView, error) {
	client, err := openCI(projectPath)
	if err != nil {
		return nil, err
	}
	prev, err := client.PreviewRerun(runID, gha.RerunOptions{JobID: jobID, FailedOnly: failedOnly})
	if err != nil {
		return nil, err
	}
	return &CIRerunPreviewView{
		RunID:       prev.RunID,
		JobID:       prev.JobID,
		FailedOnly:  prev.FailedOnly,
		RunName:     prev.RunName,
		HeadBranch:  prev.HeadBranch,
		CostWarning: prev.CostWarning,
		Usage:       usageToView(prev.Usage),
	}, nil
}

// ConfirmCIRerun executes a re-run after human confirmation.
func ConfirmCIRerun(projectPath string, runID, jobID int64, failedOnly bool) error {
	client, err := openCI(projectPath)
	if err != nil {
		return err
	}
	return client.ConfirmRerun(runID, gha.RerunOptions{JobID: jobID, FailedOnly: failedOnly})
}

// ListCIWorkflows lists Actions workflows in the open project.
func ListCIWorkflows(projectPath string) ([]CIWorkflowView, error) {
	client, err := openCI(projectPath)
	if err != nil {
		return nil, err
	}
	list, err := client.ListWorkflows()
	if err != nil {
		return nil, err
	}
	out := make([]CIWorkflowView, 0, len(list))
	for _, w := range list {
		out = append(out, CIWorkflowView{
			ID:          w.ID,
			Name:        w.Name,
			Path:        w.Path,
			State:       w.State,
			CanDispatch: w.CanDispatch,
		})
	}
	return out, nil
}

// PreviewCIDispatch prepares workflow_dispatch confirmation (no mutation).
// fields are "key=value" pairs.
func PreviewCIDispatch(projectPath, workflow, ref string, fields []string) (*CIDispatchPreviewView, error) {
	client, err := openCI(projectPath)
	if err != nil {
		return nil, err
	}
	prev, err := client.PreviewDispatch(workflow, ref, parseFieldPairs(fields))
	if err != nil {
		return nil, err
	}
	return &CIDispatchPreviewView{
		Workflow:    prev.Workflow,
		Ref:         prev.Ref,
		Fields:      prev.Fields,
		Inputs:      mapDispatchInputs(prev.Inputs),
		CostWarning: prev.CostWarning,
		CanDispatch: prev.CanDispatch,
		Usage:       usageToView(prev.Usage),
	}, nil
}

// ConfirmCIDispatch executes workflow_dispatch after human confirmation.
func ConfirmCIDispatch(projectPath, workflow, ref string, fields []string) error {
	client, err := openCI(projectPath)
	if err != nil {
		return err
	}
	return client.ConfirmDispatch(workflow, ref, parseFieldPairs(fields))
}

func mapDispatchInputs(inputs []gha.DispatchInput) []CIDispatchInputView {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]CIDispatchInputView, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, CIDispatchInputView{
			ID:          in.ID,
			Description: in.Description,
			Type:        in.Type,
			Required:    in.Required,
			Default:     in.Default,
			Options:     append([]string(nil), in.Options...),
		})
	}
	return out
}

func openCI(projectPath string) (*gha.Client, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	return gha.Open(projectPath)
}

func parseFieldPairs(fields []string) map[string]string {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]string, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}
