package desktop

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// PullResult is returned after a successful one-click pull.
type PullResult struct {
	Message   string     `json:"message"`
	Logs      []string   `json:"logs"`
	Dashboard *Dashboard `json:"dashboard"`
}

// RunPull fetches origin and fast-forwards:
//   - current branch when ↓behind
//   - local base from origin/<base> without leaving the current branch
func RunPull(projectPath, base string) (*PullResult, error) {
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

	clean, err := repo.IsClean()
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, fmt.Errorf("working tree com alterações — commit ou stash antes de puxar")
	}

	base = strings.TrimSpace(base)
	if base == "" {
		if cfg, err := config.Load(); err == nil {
			base = cfg.BaseBranch
		}
	}
	if base == "" {
		base = "main"
	}
	base = strings.TrimPrefix(base, "origin/")

	var logs []string
	logs = append(logs, "Fetching origin")
	if err := repo.FetchPrune(); err != nil {
		return nil, err
	}

	current, err := repo.CurrentBranch()
	if err != nil {
		return nil, err
	}

	var actions []string
	onBase := current == base || current == "HEAD"

	if onBase && current == base {
		logs = append(logs, "Pulling "+base)
		if err := repo.PullFFOnly(); err != nil {
			return nil, fmt.Errorf("pull %s: %w", base, err)
		}
		actions = append(actions, "pulled "+base)
	} else {
		// Feature branch: pull current when behind upstream.
		ov, err := repo.Overview(base)
		if err != nil {
			return nil, err
		}
		if ov.Behind > 0 {
			logs = append(logs, fmt.Sprintf("Pulling %s (↓%d)", current, ov.Behind))
			if err := repo.PullFFOnly(); err != nil {
				return nil, fmt.Errorf("pull %s: %w", current, err)
			}
			actions = append(actions, fmt.Sprintf("pulled %s", current))
		}

		logs = append(logs, "Updating local "+base)
		updated, err := repo.UpdateLocalBranchFromOrigin(base)
		if err != nil {
			return nil, err
		}
		if updated {
			actions = append(actions, "updated "+base)
		} else {
			logs = append(logs, "  "+base+" already up to date")
		}
	}

	dash, err := LoadDashboard(projectPath)
	if err != nil {
		return nil, err
	}

	msg := "Already up to date"
	if len(actions) > 0 {
		msg = strings.Join(actions, " · ")
	}
	return &PullResult{
		Message:   msg,
		Logs:      logs,
		Dashboard: dash,
	}, nil
}
