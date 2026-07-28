package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// ProjectOpenResult is returned when probing/opening a local folder.
// When NeedsGitInit is true, Dashboard is nil and the UI should offer git init.
type ProjectOpenResult struct {
	Dashboard    *Dashboard `json:"dashboard,omitempty"`
	Path         string     `json:"path"`
	NeedsGitInit bool       `json:"needsGitInit"`
}

// TryOpenProject opens path if it is a git repo; otherwise reports NeedsGitInit.
func TryOpenProject(projectPath string) (*ProjectOpenResult, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("caminho do projeto é obrigatório")
	}
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("não é um diretório: %s", abs)
	}

	repo, err := gitpkg.Open(abs)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return &ProjectOpenResult{Path: abs, NeedsGitInit: true}, nil
	}

	dash, err := LoadDashboard(abs)
	if err != nil {
		return nil, err
	}
	return &ProjectOpenResult{Dashboard: dash, Path: dash.Path}, nil
}

// InitGitAndOpen runs git init in path (if needed) and loads the dashboard.
// When addAll is true, also runs git add . after init (or if the repo was already present).
func InitGitAndOpen(projectPath string, addAll bool) (*Dashboard, error) {
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
		if err := repo.Init(); err != nil {
			return nil, fmt.Errorf("git init: %w", err)
		}
	}
	if addAll {
		if err := repo.AddAll(); err != nil {
			return nil, fmt.Errorf("git add .: %w", err)
		}
	}
	return LoadDashboard(abs)
}

// CreateProject creates parentDir/name, runs git init, and opens the project.
func CreateProject(parentDir, name string) (*Dashboard, error) {
	name = strings.TrimSpace(name)
	if err := validateProjectName(name); err != nil {
		return nil, err
	}
	parent := strings.TrimSpace(parentDir)
	if parent == "" {
		return nil, fmt.Errorf("diretório de destino é obrigatório")
	}
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(parentAbs); err != nil {
		return nil, err
	} else if !st.IsDir() {
		return nil, fmt.Errorf("destino não é um diretório: %s", parentAbs)
	}

	dest := filepath.Join(parentAbs, name)
	if st, err := os.Stat(dest); err == nil {
		if !st.IsDir() {
			return nil, fmt.Errorf("já existe um arquivo: %s", dest)
		}
		repo, err := gitpkg.Open(dest)
		if err != nil {
			return nil, err
		}
		if err := repo.IsRepo(); err == nil {
			return LoadDashboard(dest)
		}
		entries, readErr := os.ReadDir(dest)
		if readErr != nil {
			return nil, readErr
		}
		if len(entries) > 0 {
			return nil, fmt.Errorf("a pasta já existe e não está vazia: %s", dest)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	} else if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}

	repo, err := gitpkg.Open(dest)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		if err := repo.Init(); err != nil {
			return nil, fmt.Errorf("git init: %w", err)
		}
	}
	return LoadDashboard(dest)
}

// CloneProject clones url into parentDir/name (name optional; derived from URL).
func CloneProject(url, parentDir, name string) (*Dashboard, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("URL do repositório é obrigatória")
	}
	parent := strings.TrimSpace(parentDir)
	if parent == "" {
		return nil, fmt.Errorf("diretório de destino é obrigatório")
	}
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(parentAbs); err != nil {
		return nil, err
	} else if !st.IsDir() {
		return nil, fmt.Errorf("destino não é um diretório: %s", parentAbs)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = gitpkg.NameFromRemote(url)
	}
	if err := validateProjectName(name); err != nil {
		return nil, err
	}

	dest := filepath.Join(parentAbs, name)
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("já existe: %s", dest)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := gitpkg.Clone(url, dest); err != nil {
		return nil, err
	}
	return LoadDashboard(dest)
}

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("nome do projeto é obrigatório")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("nome de projeto inválido")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, string(os.PathSeparator)) {
		return fmt.Errorf("nome de projeto não pode conter separadores de caminho")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("nome de projeto inválido")
	}
	return nil
}
