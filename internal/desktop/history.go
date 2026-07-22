package desktop

import (
	"fmt"
	"strings"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// HistoryActionResult is returned after a timeline history mutation.
type HistoryActionResult struct {
	Message   string     `json:"message"`
	Dashboard *Dashboard `json:"dashboard,omitempty"`
}

// RevertTimelineCommit runs git revert for a commit (or merge with -m 1).
func RevertTimelineCommit(projectPath, hash string, isMerge bool) (*HistoryActionResult, error) {
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
	if err := repo.RevertCommit(hash, isMerge); err != nil {
		return nil, err
	}
	dash, err := LoadDashboard(projectPath)
	if err != nil {
		return &HistoryActionResult{Message: "revert ok"}, nil
	}
	kind := "commit"
	if isMerge {
		kind = "merge"
	}
	return &HistoryActionResult{
		Message:   fmt.Sprintf("revert do %s %s criado", kind, short(hash)),
		Dashboard: dash,
	}, nil
}

// ResetTimelineCommit runs git reset --soft|mixed|hard to hash.
func ResetTimelineCommit(projectPath, hash, mode string) (*HistoryActionResult, error) {
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

	m := gitpkg.ResetSoft
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "soft":
		m = gitpkg.ResetSoft
	case "mixed":
		m = gitpkg.ResetMixed
	case "hard":
		m = gitpkg.ResetHard
	default:
		return nil, fmt.Errorf("modo inválido: %s (use soft, mixed ou hard)", mode)
	}

	ok, err := repo.IsAncestor(hash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("commit %s não está no histórico de HEAD — reset abortado", short(hash))
	}

	if err := repo.ResetTo(hash, m); err != nil {
		return nil, err
	}
	dash, err := LoadDashboard(projectPath)
	if err != nil {
		return &HistoryActionResult{Message: "reset ok"}, nil
	}
	return &HistoryActionResult{
		Message:   fmt.Sprintf("reset --%s para %s", m, short(hash)),
		Dashboard: dash,
	}, nil
}

// DeleteTimelineBranch deletes a local branch (not the current one).
func DeleteTimelineBranch(projectPath, name string, force bool) (*HistoryActionResult, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "origin/")
	if name == "" {
		return nil, fmt.Errorf("branch vazia")
	}
	repo, err := gitpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	current, err := repo.CurrentBranch()
	if err != nil {
		return nil, err
	}
	if current == name {
		return nil, fmt.Errorf("não é possível apagar a branch atual (%s)", name)
	}
	if force {
		if err := repo.DeleteLocalBranchForce(name); err != nil {
			return nil, err
		}
	} else {
		if err := repo.DeleteLocalBranch(name); err != nil {
			return nil, err
		}
	}
	dash, _ := LoadDashboard(projectPath)
	return &HistoryActionResult{
		Message:   fmt.Sprintf("branch %s removida", name),
		Dashboard: dash,
	}, nil
}

func short(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
