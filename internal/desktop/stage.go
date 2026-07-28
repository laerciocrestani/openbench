package desktop

import (
	"fmt"
	"path/filepath"
	"strings"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// StageAll runs git add . in the project and returns an updated shell dashboard.
func StageAll(projectPath string) (*Dashboard, error) {
	repo, err := openProjectRepo(projectPath)
	if err != nil {
		return nil, err
	}
	if err := repo.AddAll(); err != nil {
		return nil, err
	}
	return LoadDashboard(repo.Dir())
}

// StageFiles runs git add on the given paths (relative to the repo root).
// With no paths, stages everything (git add .).
func StageFiles(projectPath string, paths []string) (*Dashboard, error) {
	repo, err := openProjectRepo(projectPath)
	if err != nil {
		return nil, err
	}
	clean := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		clean = append(clean, p)
	}
	if err := repo.Add(clean...); err != nil {
		return nil, err
	}
	return LoadDashboard(repo.Dir())
}

func openProjectRepo(projectPath string) (*gitpkg.Repo, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("caminho do projeto é obrigatório")
	}
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	repo, err := gitpkg.Open(abs)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("%s: não é um repositório git", abs)
	}
	return repo, nil
}
