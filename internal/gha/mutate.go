package gha

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// RerunOptions controls gh run rerun.
type RerunOptions struct {
	JobID      int64 // specific job; 0 = whole run (or failed-only)
	FailedOnly bool
	Debug      bool
}

// Workflow is a repo Actions workflow.
type Workflow struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	State       string `json:"state"`
	CanDispatch bool   `json:"canDispatch"`
}

// RerunPreview describes a pending re-run (no mutation).
type RerunPreview struct {
	RunID       int64        `json:"runId"`
	JobID       int64        `json:"jobId,omitempty"`
	FailedOnly  bool         `json:"failedOnly"`
	RunName     string       `json:"runName"`
	HeadBranch  string       `json:"headBranch"`
	CostWarning string       `json:"costWarning"`
	Usage       ActionsUsage `json:"usage"`
}

// DispatchPreview describes a pending workflow_dispatch (no mutation).
type DispatchPreview struct {
	Workflow    string            `json:"workflow"`
	WorkflowID  int64             `json:"workflowId,omitempty"`
	Ref         string            `json:"ref,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
	Inputs      []DispatchInput   `json:"inputs,omitempty"`
	CostWarning string            `json:"costWarning"`
	Usage       ActionsUsage      `json:"usage"`
	CanDispatch bool              `json:"canDispatch"`
}

// PreviewRerun builds a confirm payload with cost warning.
func (c *Client) PreviewRerun(runID int64, opts RerunOptions) (*RerunPreview, error) {
	if runID <= 0 {
		return nil, fmt.Errorf("run id inválido")
	}
	detail, err := c.ViewRun(runID)
	if err != nil {
		return nil, err
	}
	usage := c.UsageForRun(detail.Run, c.RemoteOwner())
	scope := "o workflow run inteiro"
	if opts.JobID > 0 {
		scope = fmt.Sprintf("o job %d (e dependências)", opts.JobID)
	} else if opts.FailedOnly {
		scope = "somente jobs falhos (e dependências)"
	}
	return &RerunPreview{
		RunID:       runID,
		JobID:       opts.JobID,
		FailedOnly:  opts.FailedOnly,
		RunName:     detail.Run.Name,
		HeadBranch:  detail.Run.HeadBranch,
		CostWarning: fmt.Sprintf("Re-run de %s consome minutos adicionais de GitHub Actions.", scope),
		Usage:       usage,
	}, nil
}

// ConfirmRerun executes gh run rerun after human confirmation.
func (c *Client) ConfirmRerun(runID int64, opts RerunOptions) error {
	if runID <= 0 {
		return fmt.Errorf("run id inválido")
	}
	args := []string{"run", "rerun", strconv.FormatInt(runID, 10)}
	if opts.JobID > 0 {
		args = append(args, "--job", strconv.FormatInt(opts.JobID, 10))
	} else if opts.FailedOnly {
		args = append(args, "--failed")
	}
	if opts.Debug {
		args = append(args, "--debug")
	}
	_, err := c.run(args...)
	return err
}

// ListWorkflows returns workflows; CanDispatch is best-effort via YAML.
func (c *Client) ListWorkflows() ([]Workflow, error) {
	out, err := c.run("workflow", "list", "--json", "id,name,path,state")
	if err != nil {
		return nil, err
	}
	type raw struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Path  string `json:"path"`
		State string `json:"state"`
	}
	var items []raw
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("parse workflow list: %w", err)
	}
	workflows := make([]Workflow, 0, len(items))
	for _, item := range items {
		workflows = append(workflows, Workflow{
			ID:    item.ID,
			Name:  item.Name,
			Path:  item.Path,
			State: item.State,
			// CanDispatch is resolved on PreviewDispatch (avoids N YAML fetches).
		})
	}
	return workflows, nil
}

// PreviewDispatch builds confirm payload for workflow_dispatch.
func (c *Client) PreviewDispatch(workflow, ref string, fields map[string]string) (*DispatchPreview, error) {
	workflow = strings.TrimSpace(workflow)
	if workflow == "" {
		return nil, fmt.Errorf("workflow obrigatório")
	}
	yamlBody, can, inputs := c.loadDispatchSchema(workflow)
	_ = yamlBody
	fields = MergeDispatchDefaults(inputs, fields)
	usage := c.UsageForRuns(nil, c.RemoteOwner())
	preview := &DispatchPreview{
		Workflow:    workflow,
		Ref:         strings.TrimSpace(ref),
		Fields:      fields,
		Inputs:      inputs,
		CanDispatch: can,
		CostWarning: "workflow_dispatch vai disparar um novo run e consumir minutos de GitHub Actions.",
		Usage:       usage,
	}
	if !can {
		preview.CostWarning = "Este workflow pode não ter trigger workflow_dispatch — o GitHub pode rejeitar o disparo."
	}
	return preview, nil
}

// ConfirmDispatch executes gh workflow run after human confirmation.
func (c *Client) ConfirmDispatch(workflow, ref string, fields map[string]string) error {
	workflow = strings.TrimSpace(workflow)
	if workflow == "" {
		return fmt.Errorf("workflow obrigatório")
	}
	_, _, inputs := c.loadDispatchSchema(workflow)
	fields = MergeDispatchDefaults(inputs, fields)
	if missing := MissingRequiredDispatchInputs(inputs, fields); len(missing) > 0 {
		return fmt.Errorf("input obrigatório não informado: %s", strings.Join(missing, ", "))
	}
	args := []string{"workflow", "run", workflow}
	if r := strings.TrimSpace(ref); r != "" {
		args = append(args, "--ref", r)
	}
	for k, v := range fields {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		args = append(args, "--raw-field", k+"="+v)
	}
	_, err := c.run(args...)
	return err
}

func (c *Client) loadDispatchSchema(nameOrPath string) (yamlBody string, canDispatch bool, inputs []DispatchInput) {
	nameOrPath = strings.TrimSpace(nameOrPath)
	if nameOrPath == "" {
		return "", false, nil
	}
	out, _, err := c.runAllowExit("workflow", "view", nameOrPath, "--yaml")
	if out == "" {
		_ = err
		return "", false, nil
	}
	can, parsed, parseErr := ParseWorkflowDispatchInputs(out)
	if parseErr != nil {
		// Fallback to cheap string signal if YAML parse fails.
		return out, strings.Contains(out, "workflow_dispatch"), nil
	}
	return out, can, parsed
}

func (c *Client) workflowHasDispatch(nameOrPath string) bool {
	_, can, _ := c.loadDispatchSchema(nameOrPath)
	return can
}
