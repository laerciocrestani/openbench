package desktop

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/laerciocrestani/openbench/internal/gha"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// PushPreview is shown before pushing (default-branch gate).
type PushPreview struct {
	Path                 string `json:"path"`
	Branch               string `json:"branch"`
	DefaultBranch        string `json:"defaultBranch"`
	DefaultBranchWarning string `json:"defaultBranchWarning,omitempty"`
	Ahead                int    `json:"ahead"`
	WillTriggerCI        bool   `json:"willTriggerCI"`
}

// PushOutcome is returned after pushing the current branch.
type PushOutcome struct {
	Path                 string       `json:"path"`
	Branch               string       `json:"branch"`
	HeadSHA              string       `json:"headSha,omitempty"`
	Message              string       `json:"message"`
	DefaultBranchWarning string       `json:"defaultBranchWarning,omitempty"`
	CI                   *CIWatchView `json:"ci,omitempty"`
}

// CIWatchView summarizes post-push CI observation.
type CIWatchView struct {
	Branch   string      `json:"branch"`
	HeadSHA  string      `json:"headSha,omitempty"`
	TimedOut bool        `json:"timedOut"`
	Message  string      `json:"message"`
	Usage    CIUsageView `json:"usage"`
	Runs     []CIRunView `json:"runs"`
}

// CIWatchUpdate is emitted on event ci:watch during observation.
type CIWatchUpdate struct {
	Path    string      `json:"path"`
	Phase   string      `json:"phase"`
	Message string      `json:"message"`
	Branch  string      `json:"branch"`
	HeadSHA string      `json:"headSha,omitempty"`
	Usage   CIUsageView `json:"usage"`
	Runs    []CIRunView `json:"runs"`
}

// PreviewPush returns gate info without pushing.
func PreviewPush(projectPath string) (*PushPreview, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	repo, err := gitpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}
	branch, err := repo.CurrentBranch()
	if err != nil {
		return nil, err
	}
	if branch == "" || branch == "HEAD" {
		return nil, fmt.Errorf("checkout em detached HEAD — faça checkout de uma branch antes do push")
	}
	overview, err := repo.Overview("")
	if err != nil {
		return nil, err
	}
	ahead := overview.Ahead
	hasUpstream := strings.TrimSpace(overview.Upstream) != ""
	if !hasUpstream {
		ahead = overview.CommitsAheadOfBase
		if ahead <= 0 {
			ahead = overview.Unpublished
		}
	}
	def := gha.ResolveDefaultBranch(projectPath)
	warn := gha.DefaultBranchWarning(branch, def)
	return &PushPreview{
		Path:                 projectPath,
		Branch:               branch,
		DefaultBranch:        def,
		DefaultBranchWarning: warn,
		Ahead:                ahead,
		WillTriggerCI:        true,
	}, nil
}

// PushCurrentBranch pushes existing commits on HEAD to origin (no commit/AI).
// When watchCI is true, observes Actions for the pushed SHA.
func PushCurrentBranch(projectPath string, watchCI bool, onWatch func(CIWatchUpdate)) (*PushOutcome, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	repo, err := gitpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	branch, err := repo.CurrentBranch()
	if err != nil {
		return nil, err
	}
	if branch == "" || branch == "HEAD" {
		return nil, fmt.Errorf("checkout em detached HEAD — faça checkout de uma branch antes do push")
	}

	overview, err := repo.Overview("")
	if err != nil {
		return nil, err
	}
	hasUpstream := strings.TrimSpace(overview.Upstream) != ""
	if hasUpstream && overview.Ahead <= 0 {
		return nil, fmt.Errorf("nada para enviar — branch já está sincronizada com o remote (↑0)")
	}

	def := gha.ResolveDefaultBranch(projectPath)
	warn := gha.DefaultBranchWarning(branch, def)

	if err := repo.Push(); err != nil {
		return nil, err
	}
	sha := gitHeadSHA(projectPath)
	msg := fmt.Sprintf("push de %s ok", branch)
	if hasUpstream && overview.Ahead > 0 {
		msg = fmt.Sprintf("push de %s (↑%d) ok", branch, overview.Ahead)
	}
	if warn != "" {
		msg += " · CI será disparada na branch default"
	}

	out := &PushOutcome{
		Path:                 projectPath,
		Branch:               branch,
		HeadSHA:              sha,
		Message:              msg,
		DefaultBranchWarning: warn,
	}
	if watchCI {
		out.CI = watchCIAfterPush(projectPath, branch, sha, onWatch)
		if out.CI != nil && out.CI.Message != "" {
			out.Message += " · " + out.CI.Message
		}
	}
	return out, nil
}

func watchCIAfterPush(projectPath, branch, sha string, onWatch func(CIWatchUpdate)) *CIWatchView {
	client, err := gha.Open(projectPath)
	if err != nil {
		return &CIWatchView{
			Branch:  branch,
			HeadSHA: sha,
			Message: "CI: " + err.Error(),
			Usage: CIUsageView{
				State:   gha.UsageStateUnavailable,
				Message: err.Error(),
			},
			Runs: []CIRunView{},
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	result, err := client.WatchAfterPush(ctx, gha.WatchOptions{
		Branch:  branch,
		HeadSHA: sha,
		OnUpdate: func(snap gha.WatchSnapshot) {
			if onWatch == nil {
				return
			}
			onWatch(CIWatchUpdate{
				Path:    projectPath,
				Phase:   snap.Phase,
				Message: snap.Message,
				Branch:  snap.Branch,
				HeadSHA: snap.HeadSHA,
				Usage:   usageToView(snap.Usage),
				Runs:    runsToViews(snap.Runs),
			})
		},
	})
	if err != nil {
		return &CIWatchView{
			Branch:  branch,
			HeadSHA: sha,
			Message: "CI: " + err.Error(),
			Usage: CIUsageView{
				State:   gha.UsageStateUnavailable,
				Message: err.Error(),
			},
			Runs: []CIRunView{},
		}
	}
	return watchResultToView(result)
}

func watchResultToView(r *gha.WatchResult) *CIWatchView {
	if r == nil {
		return nil
	}
	return &CIWatchView{
		Branch:   r.Branch,
		HeadSHA:  r.HeadSHA,
		TimedOut: r.TimedOut,
		Message:  r.Message,
		Usage:    usageToView(r.Usage),
		Runs:     runsToViews(r.Runs),
	}
}

func runsToViews(runs []gha.WorkflowRun) []CIRunView {
	out := make([]CIRunView, 0, len(runs))
	for _, r := range runs {
		out = append(out, runToView(r))
	}
	return out
}

func gitHeadSHA(dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// WatchCIAfterPush observes Actions for branch/SHA and invokes onWatch each poll.
func WatchCIAfterPush(projectPath, branch, headSHA string, onWatch func(CIWatchUpdate)) *CIWatchView {
	if strings.TrimSpace(headSHA) == "" {
		headSHA = gitHeadSHA(projectPath)
	}
	if strings.TrimSpace(branch) == "" {
		if repo, err := gitpkg.Open(projectPath); err == nil {
			if b, err := repo.CurrentBranch(); err == nil {
				branch = b
			}
		}
	}
	return watchCIAfterPush(projectPath, branch, headSHA, onWatch)
}
