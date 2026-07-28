package desktop

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// SetOriginRemote configures git remote origin and returns an updated dashboard.
func SetOriginRemote(projectPath, url string) (*Dashboard, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("URL do remote é obrigatória")
	}
	repo, err := openProjectRepo(projectPath)
	if err != nil {
		return nil, err
	}
	if err := repo.SetOrigin(url); err != nil {
		return nil, err
	}
	return dashboardWithRemote(repo)
}

// CreateGitHubRepo creates a GitHub repository with gh and sets origin (no push).
// visibility must be "public" or "private".
func CreateGitHubRepo(projectPath, name, visibility, description string) (*Dashboard, error) {
	repo, err := openProjectRepo(projectPath)
	if err != nil {
		return nil, err
	}
	if repo.HasOrigin() {
		return nil, fmt.Errorf("origin já existe — use \"Usar URL\" para alterar o remote")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = filepath.Base(repo.Dir())
	}
	if err := validateGitHubRepoName(name); err != nil {
		return nil, err
	}

	visibility = strings.ToLower(strings.TrimSpace(visibility))
	if visibility != "public" && visibility != "private" {
		return nil, fmt.Errorf("visibilidade deve ser public ou private")
	}

	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("GitHub CLI (gh) não encontrado — instale: https://cli.github.com/")
	}
	status := exec.Command("gh", "auth", "status")
	status.Stdout = nil
	status.Stderr = nil
	if err := status.Run(); err != nil {
		return nil, fmt.Errorf("gh sem autenticação — rode: gh auth login")
	}

	args := []string{
		"repo", "create", name,
		"--source=" + repo.Dir(),
		"--remote=origin",
		"--" + visibility,
	}
	if desc := strings.TrimSpace(description); desc != "" {
		args = append(args, "--description", desc)
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = repo.Dir()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh repo create: %s", msg)
	}

	return dashboardWithRemote(repo)
}

func dashboardWithRemote(repo *gitpkg.Repo) (*Dashboard, error) {
	dash, err := LoadDashboard(repo.Dir())
	if err != nil {
		return nil, err
	}
	if url, err := repo.RemoteOriginURL(); err == nil {
		dash.RemoteURL = url
	}
	if _, err := exec.LookPath("gh"); err == nil {
		dash.HasGH = true
	}
	return dash, nil
}

func validateGitHubRepoName(name string) error {
	if name == "" {
		return fmt.Errorf("nome do repositório é obrigatório")
	}
	if strings.ContainsAny(name, " \t") {
		return fmt.Errorf("nome do repositório não pode conter espaços")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("nome do repositório inválido")
	}
	// Allow owner/repo for gh create under an org.
	if strings.Count(name, "/") > 1 {
		return fmt.Errorf("nome do repositório inválido (use nome ou org/nome)")
	}
	return nil
}
