package desktop

import (
	"os/exec"
	"path/filepath"

	"github.com/laerciocrestani/openbench/internal/config"
	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

// ProjectStatus is a lightweight per-project status for tabs / hub.
type ProjectStatus struct {
	Path          string `json:"path"`
	RepoName      string `json:"repoName"`
	Alias         string `json:"alias,omitempty"`
	Branch        string `json:"branch"`
	Dirty         bool   `json:"dirty"`
	ChangedFiles  int    `json:"changedFiles"`
	Insertions    int    `json:"insertions"`
	Deletions     int    `json:"deletions"`
	StatusLabel   string `json:"statusLabel"`
	DockerSummary string `json:"dockerSummary"`
	DockerVisible bool   `json:"dockerVisible"`
	HasOpenPR     bool   `json:"hasOpenPR"`
	PRTitle       string `json:"prTitle,omitempty"`
	Active        bool   `json:"active"`
	Error         string `json:"error,omitempty"`
}

// LoadProjectStatus collects a light status. With includePR, also queries gh (slower).
func LoadProjectStatus(projectPath string, includePR bool) ProjectStatus {
	st := ProjectStatus{
		Path:     projectPath,
		RepoName: filepath.Base(projectPath),
	}
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Path = abs
	st.RepoName = filepath.Base(abs)

	repo, err := gitpkg.Open(abs)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	if err := repo.IsRepo(); err != nil {
		st.Error = "não é um repositório git"
		return st
	}

	base := "main"
	if cfg, cfgErr := config.Load(); cfgErr == nil && cfg.BaseBranch != "" {
		base = cfg.BaseBranch
	}

	overview, err := repo.Overview(base)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.RepoName = filepath.Base(overview.Root)
	if overview.Root != "" {
		st.Path = overview.Root
	}
	st.Branch = overview.Branch
	if overview.Detached {
		st.Branch = "detached HEAD"
	}
	st.Dirty = overview.IsDirty()
	st.StatusLabel = statusLabel(overview.IsDirty(), overview.Staged, overview.Modified, overview.Untracked)
	st.ChangedFiles = len(overview.FileChanges)
	for _, c := range overview.FileChanges {
		st.Insertions += c.Insertions
		st.Deletions += c.Deletions
	}

	// Avoid docker info / compose ps on the hub hot path (can stall for seconds).
	if dockerpkg.HasDocker() && dockerpkg.FindComposeFile(abs) != "" {
		st.DockerVisible = true
		st.DockerSummary = "…"
	}

	if includePR {
		if _, err := exec.LookPath("gh"); err == nil {
			client, err := prpkg.Open(abs)
			if err == nil {
				if pr, _ := client.ViewCurrent(); pr != nil {
					st.HasOpenPR = true
					st.PRTitle = pr.Title
				}
			}
		}
	}

	return st
}
