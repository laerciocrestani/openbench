package app

import (
	"fmt"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
)

const defaultLogLimit = 30

// LoadLog returns recent git log output for the TUI logs view.
func LoadLog(snap *WorkspaceSnapshot) (string, error) {
	if snap == nil || snap.Overview == nil {
		return "", fmt.Errorf("snapshot inválido")
	}

	repo, err := gitpkg.New()
	if err != nil {
		return "", err
	}

	log, err := repo.RecentLog(defaultLogLimit)
	if err != nil {
		return "", err
	}
	if log == "" {
		return "", fmt.Errorf("nenhum commit no histórico")
	}
	return log, nil
}
