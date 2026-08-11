package desktop

import (
	"os/exec"
	"path/filepath"

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
	CIState       string `json:"ciState,omitempty"`
	CILabel       string `json:"ciLabel,omitempty"`
	CIFromCache   bool   `json:"ciFromCache,omitempty"`
	CIHost        string `json:"ciHost,omitempty"`
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

	// Hub hot path: porcelain only (no base/rev-list/numstat/branches/stash).
	// Empty baseBranch skips ResolveBase / rev-list / BaseBehindOrigin inside LoadStatus.
	snap, err := repo.LoadStatus("", true)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.RepoName = filepath.Base(snap.Root)
	if snap.Root != "" {
		st.Path = snap.Root
	}
	st.Branch = snap.Branch
	if snap.Detached {
		st.Branch = "detached HEAD"
	}
	dirty := snap.Staged > 0 || snap.Modified > 0 || snap.Untracked > 0
	st.Dirty = dirty
	st.StatusLabel = statusLabel(dirty, snap.Staged, snap.Modified, snap.Untracked)
	st.ChangedFiles = len(snap.FileChanges)
	// Insertions/deletions intentionally omitted (numstat is too expensive for polling).

	// Root compose detect only — FindComposeFile walks parents and is too heavy here.
	if dockerpkg.HasDocker() && dockerpkg.DetectComposeFile(abs) != "" {
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
		if sum := LoadCISummaryForProject(abs, st.Branch); sum != nil {
			st.CIState = sum.State
			st.CILabel = sum.Label
			st.CIFromCache = sum.FromCache
			st.CIHost = sum.Host
		}
	}

	return st
}
