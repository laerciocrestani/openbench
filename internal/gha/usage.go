package gha

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

type orgBillingActions struct {
	TotalMinutesUsed int `json:"total_minutes_used"`
	IncludedMinutes  int `json:"included_minutes"`
}

// UsageForRuns builds ActionsUsage from a window of runs and best-effort org billing.
func (c *Client) UsageForRuns(runs []WorkflowRun, owner string) ActionsUsage {
	window := windowMinutes(runs)
	usage := ActionsUsage{
		State:   UsageStateRepoWindow,
		Message: "estimativa wall-clock dos runs listados (não é billable oficial)",
	}
	if window > 0 {
		usage.WindowMinutes = floatPtr(window)
	} else if len(runs) == 0 {
		usage.Message = "nenhum workflow run listado neste repositório/branch"
	} else {
		usage.Message = "sem duração calculável nos runs listados"
	}

	owner = strings.TrimSpace(owner)
	if owner == "" {
		return usage
	}

	orgUsage, err := c.orgBilling(owner)
	if err != nil || orgUsage == nil {
		// Billing is optional chrome — never surface it in the CI banner.
		return usage
	}

	used := float64(orgUsage.TotalMinutesUsed)
	included := float64(orgUsage.IncludedMinutes)
	usage.State = UsageStateOrg
	usage.OrgUsedMinutes = floatPtr(used)
	usage.OrgIncludedMinutes = floatPtr(included)
	if included > 0 {
		rem := math.Max(0, included-used)
		usage.OrgRemainingMinutes = floatPtr(rem)
	}
	usage.Message = fmt.Sprintf("billing Actions da org/user %s (mês corrente)", owner)
	if usage.WindowMinutes != nil {
		usage.Message += fmt.Sprintf(" · janela listada ~%.1f min", *usage.WindowMinutes)
	}
	return usage
}

// UsageForRun returns per-run wall-clock minutes plus org billing when possible.
func (c *Client) UsageForRun(run WorkflowRun, owner string) ActionsUsage {
	base := c.UsageForRuns([]WorkflowRun{run}, owner)
	mins := runMinutes(run)
	if mins > 0 {
		base.RunMinutes = floatPtr(mins)
		if base.State != UsageStateOrg {
			base.State = UsageStateRun
			base.Message = "duração wall-clock do run (não é billable oficial)"
		}
	}
	return base
}

func (c *Client) orgBilling(owner string) (*orgBillingActions, error) {
	out, err := c.run("api", fmt.Sprintf("orgs/%s/settings/billing/actions", owner))
	if err != nil {
		// Users (not orgs) — try nothing else; mark unavailable via caller.
		return nil, err
	}
	var raw orgBillingActions
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}
	return &raw, nil
}

func windowMinutes(runs []WorkflowRun) float64 {
	var total float64
	for _, r := range runs {
		total += runMinutes(r)
	}
	return round1(total)
}

func runMinutes(r WorkflowRun) float64 {
	start := r.StartedAt
	if start.IsZero() {
		start = r.CreatedAt
	}
	end := r.UpdatedAt
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	// Cap absurd gaps (queued forever) at 6h wall for estimate.
	d := end.Sub(start)
	if d > 6*time.Hour {
		d = 6 * time.Hour
	}
	return round1(d.Minutes())
}

func unavailableReason(err error) string {
	if err == nil {
		return "indisponível"
	}
	low := strings.ToLower(err.Error())
	switch {
	case isOptionalBillingErr(err):
		return "billing da org indisponível (permissão opcional)"
	case strings.Contains(low, "404") || strings.Contains(low, "not found"):
		return "enterprise_unsupported ou owner não é org"
	case strings.Contains(low, "auth"):
		return "no_permission — auth insuficiente"
	default:
		return "api_error — " + err.Error()
	}
}

// isOptionalBillingErr reports org billing failures that are expected without admin:org.
func isOptionalBillingErr(err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "403") ||
		strings.Contains(low, "forbidden") ||
		strings.Contains(low, "sem permissão") ||
		strings.Contains(low, "admin:org") ||
		strings.Contains(low, "auth refresh") ||
		strings.Contains(low, "billing") ||
		strings.Contains(low, "actions policies") ||
		strings.Contains(low, "fine-grained permission")
}

func floatPtr(v float64) *float64 { return &v }

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
