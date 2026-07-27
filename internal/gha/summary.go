package gha

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// SummaryState values for CI badge / cache.
const (
	SummaryPass        = "pass"
	SummaryFail        = "fail"
	SummaryPending     = "pending"
	SummaryUnknown     = "unknown"
	SummaryOffline     = "offline"
	SummaryUnavailable = "unavailable"
)

// Summary is a lightweight Actions status for StatusHub / Doctor / cache.
type Summary struct {
	State      string `json:"state"`
	Label      string `json:"label"`
	Pass       int    `json:"pass"`
	Fail       int    `json:"fail"`
	Pending    int    `json:"pending"`
	Host       string `json:"host"`
	Branch     string `json:"branch,omitempty"`
	FromCache  bool   `json:"fromCache,omitempty"`
	Message    string `json:"message,omitempty"`
	Enterprise bool   `json:"enterprise,omitempty"`
}

// LoadSummary lists recent runs for the current branch and builds a badge summary.
// Only the latest run per workflow counts — an older failure is ignored when a
// newer run of the same workflow succeeded (or is pending).
func (c *Client) LoadSummary(branch string) (*Summary, error) {
	host := ResolveHost(c.dir)
	sum := &Summary{
		Host:       host,
		Branch:     branch,
		Enterprise: IsEnterpriseHost(host),
		State:      SummaryUnknown,
		Label:      "CI ?",
	}
	f := ListFilter{Branch: strings.TrimSpace(branch), Limit: 20}
	runs, err := c.ListRuns(f)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		sum.State = SummaryUnknown
		sum.Label = "CI —"
		sum.Message = "nenhum run recente nesta branch"
		return sum, nil
	}
	return FinalizeSummary(sum, latestRunsPerWorkflow(runs)), nil
}

// FinalizeSummary applies pass/fail/pending counts and state/label to sum.
func FinalizeSummary(sum *Summary, latest []WorkflowRun) *Summary {
	if sum == nil {
		sum = &Summary{State: SummaryUnknown, Label: "CI ?"}
	}
	for _, r := range latest {
		switch {
		case Failed(r.Status, r.Conclusion):
			sum.Fail++
		case normalize(r.Conclusion) == "success":
			sum.Pass++
		case !isTerminal(r.Status, r.Conclusion):
			sum.Pending++
		default:
			// neutral/skipped count as pass-ish for badge
			sum.Pass++
		}
	}
	switch {
	case sum.Fail > 0:
		sum.State = SummaryFail
		sum.Label = fmt.Sprintf("CI %d✗", sum.Fail)
	case sum.Pending > 0:
		sum.State = SummaryPending
		sum.Label = fmt.Sprintf("CI %d…", sum.Pending)
	case sum.Pass > 0:
		sum.State = SummaryPass
		sum.Label = fmt.Sprintf("CI %d✓", sum.Pass)
	default:
		sum.State = SummaryUnknown
		sum.Label = "CI ?"
	}
	if sum.Enterprise && sum.Message == "" {
		sum.Message = "host " + sum.Host
	}
	return sum
}

// latestRunsPerWorkflow keeps only the newest run for each workflow.
func latestRunsPerWorkflow(runs []WorkflowRun) []WorkflowRun {
	if len(runs) == 0 {
		return nil
	}
	ordered := append([]WorkflowRun(nil), runs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		return a.ID > b.ID
	})
	seen := make(map[string]struct{}, len(ordered))
	out := make([]WorkflowRun, 0, len(ordered))
	for _, r := range ordered {
		key := workflowKey(r)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

func workflowKey(r WorkflowRun) string {
	if r.WorkflowID > 0 {
		return fmt.Sprintf("id:%d", r.WorkflowID)
	}
	name := strings.TrimSpace(r.WorkflowName)
	if name == "" {
		name = strings.TrimSpace(r.Name)
	}
	if name == "" {
		return fmt.Sprintf("run:%d", r.ID)
	}
	return "name:" + strings.ToLower(name)
}

// SummaryFromError maps classified errors to an offline/unavailable summary.
func SummaryFromError(dir string, err error, cached *Summary) *Summary {
	host := ResolveHost(dir)
	sum := &Summary{
		Host:       host,
		Enterprise: IsEnterpriseHost(host),
		State:      SummaryUnavailable,
		Label:      "CI !",
		Message:    err.Error(),
	}
	if cached != nil {
		out := *cached
		out.FromCache = true
		out.Host = host
		out.Enterprise = IsEnterpriseHost(host)
		switch {
		case errorsIs(err, ErrNetwork):
			out.State = SummaryOffline
			out.Label = cached.Label + " (off)"
			out.Message = "offline — último status em cache"
		case errorsIs(err, ErrGhAuth), errorsIs(err, ErrGhNotInstalled):
			out.State = SummaryUnavailable
			out.Message = err.Error()
		default:
			out.Message = err.Error() + " · cache"
		}
		return &out
	}
	if errorsIs(err, ErrNetwork) {
		sum.State = SummaryOffline
		sum.Label = "CI off"
		sum.Message = "sem rede — sem cache de CI"
	}
	return sum
}

func errorsIs(err, target error) bool {
	return errors.Is(err, target)
}
