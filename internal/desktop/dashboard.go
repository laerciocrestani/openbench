package desktop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/laerciocrestani/openbench/internal/app"
	"github.com/laerciocrestani/openbench/internal/config"
	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

// Dashboard is a JSON-friendly view model for the desktop UI.
type Dashboard struct {
	Path               string              `json:"path"`
	RepoName           string              `json:"repoName"`
	Branch             string              `json:"branch"`
	Detached           bool                `json:"detached"`
	Dirty              bool                `json:"dirty"`
	Staged             int                 `json:"staged"`
	Modified           int                 `json:"modified"`
	Untracked          int                 `json:"untracked"`
	Ahead              int                 `json:"ahead"`
	Behind             int                 `json:"behind"`
	HasUpstream        bool                `json:"hasUpstream"`
	BaseBranch         string              `json:"baseBranch"`
	CommitsAheadOfBase int                 `json:"commitsAheadOfBase"`
	HasBranchDiff      bool                `json:"hasBranchDiff"`
	BaseBehind         int                 `json:"baseBehind"`
	HygieneLocal       int                 `json:"hygieneLocal"`
	HygieneRemote      int                 `json:"hygieneRemote"`
	HeadHash           string              `json:"headHash"`
	RemoteURL          string              `json:"remoteURL"`
	StatusLabel        string              `json:"statusLabel"`
	HasGH              bool                `json:"hasGH"`
	HasDocker          bool                `json:"hasDocker"`
	Docker             DockerStatus        `json:"docker"`
	OpenPR             *PRStatus           `json:"openPR,omitempty"`
	CIState            string              `json:"ciState,omitempty"`
	CILabel            string              `json:"ciLabel,omitempty"`
	CIFromCache        bool                `json:"ciFromCache,omitempty"`
	CIHost             string              `json:"ciHost,omitempty"`
	AIReady            bool                `json:"aiReady"`
	Provider           string              `json:"provider"`
	Model              string              `json:"model"`
	NextSteps          []NextStepView      `json:"nextSteps"`
	ChangedFiles       []ChangedFileView   `json:"changedFiles"`
	ContextIndex       *CommitContextIndex `json:"contextIndex,omitempty"`
}

// CommitContextIndex is the desktop DTO for commit-context health.
type CommitContextIndex struct {
	Score              int    `json:"score"`
	Level              string `json:"level"`
	Label              string `json:"label"`
	RecommendCommit    bool   `json:"recommendCommit"`
	FileCount          int    `json:"fileCount"`
	Insertions         int    `json:"insertions"`
	Deletions          int    `json:"deletions"`
	AreaCount          int    `json:"areaCount"`
	EstimatedBytes     int    `json:"estimatedBytes"`
	MaxDiffBytes       int    `json:"maxDiffBytes"`
	NearTruncate       bool   `json:"nearTruncate"`
	Model              string `json:"model,omitempty"`
	ModelContextWindow string `json:"modelContextWindow,omitempty"`
}

// DockerStatus summarizes compose/daemon state for the dashboard.
type DockerStatus struct {
	Available      bool                `json:"available"`
	DaemonRunning  bool                `json:"daemonRunning"`
	ComposeFile    string              `json:"composeFile"`
	Summary        string              `json:"summary"`
	Running        int                 `json:"running"`
	Total          int                 `json:"total"`
	Visible        bool                `json:"visible"`
	DefaultService string              `json:"defaultService"`
	Services       []DockerServiceView `json:"services"`
}

// DockerServiceView is one compose service for UI selectors.
type DockerServiceView struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	Container string `json:"container,omitempty"`
	Ports     string `json:"ports,omitempty"`
	Health    string `json:"health,omitempty"`
}

// PRStatus is the open pull request, if any.
type PRStatus struct {
	URL            string `json:"url"`
	Title          string `json:"title"`
	State          string `json:"state"`
	Number         int    `json:"number"`
	IsDraft        bool   `json:"isDraft"`
	HeadRefName    string `json:"headRefName,omitempty"`
	Mergeable      string `json:"mergeable,omitempty"`
	ReviewDecision string `json:"reviewDecision,omitempty"`
	ChecksPass     int    `json:"checksPass"`
	ChecksFail     int    `json:"checksFail"`
	ChecksPending  int    `json:"checksPending"`
	ChecksTotal    int    `json:"checksTotal"`
	ChecksSummary  string `json:"checksSummary,omitempty"`
}

// NextStepView is a suggested next action.
type NextStepView struct {
	Command string `json:"command"`
	Note    string `json:"note"`
}

// LoadDashboard returns an instant shell dashboard (path/branch/HEAD only).
// Enrich via LoadGitStatus / LoadDockerStatus / LoadOpenPR / LoadHygieneCounts /
// LoadCIBadge / LoadChangedFiles after the UI is shown.
func LoadDashboard(projectPath string) (*Dashboard, error) {
	return LoadDashboardShell(projectPath)
}

// LoadDashboardShell validates the git repo and returns identity fields only.
// Target: a handful of fast git calls — no status, no diff, no network.
func LoadDashboardShell(projectPath string) (*Dashboard, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	repo, err := gitpkg.Open(abs)
	if err != nil {
		return nil, err
	}
	// Shell's rev-parse --show-toplevel fails if not a git repo (avoids extra IsRepo call).
	shell, err := repo.Shell()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", abs, err)
	}

	d := &Dashboard{
		Path:         shell.Root,
		RepoName:     filepath.Base(shell.Root),
		Branch:       shell.Branch,
		Detached:     shell.Detached,
		HeadHash:     shell.HeadHash,
		StatusLabel:  "…",
		BaseBranch:   "main",
		ChangedFiles: []ChangedFileView{},
		NextSteps:    []NextStepView{},
		HasGH:        false,
		HasDocker:    false,
	}
	if shell.Detached {
		d.Branch = "detached HEAD"
	}

	if cfg, cfgErr := config.Load(); cfgErr == nil && cfg != nil {
		if strings.TrimSpace(cfg.BaseBranch) != "" {
			d.BaseBranch = cfg.BaseBranch
		}
		d.Provider = string(cfg.Provider)
		d.Model = cfg.Model
		d.AIReady = strings.TrimSpace(cfg.APIKey) != ""
	}

	// Root-only compose detect (no upward walk) — cheap enough for the shell.
	if compose := dockerpkg.DetectComposeFile(shell.Root); compose != "" {
		d.HasDocker = true
		d.Docker = DockerStatus{
			Visible:     true,
			ComposeFile: compose,
			Summary:     "carregando…",
			Services:    []DockerServiceView{},
		}
	}

	return d, nil
}

// LoadGitStatus enriches the dashboard with dirty/ahead/behind/base/files (no numstat).
// Safe to call after the shell is on screen; still skips Docker/PR/hygiene/CI/numstat.
func LoadGitStatus(projectPath string) (*Dashboard, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
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
		return nil, fmt.Errorf("%s: %w", abs, err)
	}

	base := "main"
	var cfg *config.Config
	if c, err := config.Load(); err == nil && c != nil {
		cfg = c
		if strings.TrimSpace(c.BaseBranch) != "" {
			base = c.BaseBranch
		}
	}

	st, err := repo.LoadStatus(base, true)
	if err != nil {
		return nil, err
	}

	d := &Dashboard{
		Path:               st.Root,
		RepoName:           filepath.Base(st.Root),
		Branch:             st.Branch,
		Detached:           st.Detached,
		Dirty:              st.Staged > 0 || st.Modified > 0 || st.Untracked > 0,
		Staged:             st.Staged,
		Modified:           st.Modified,
		Untracked:          st.Untracked,
		Ahead:              st.Ahead,
		Behind:             st.Behind,
		HasUpstream:        strings.TrimSpace(st.Upstream) != "",
		BaseBranch:         base,
		CommitsAheadOfBase: st.CommitsAheadOfBase,
		HasBranchDiff:      st.HasBranchDiff,
		BaseBehind:         st.BaseBehind,
		HeadHash:           st.HeadHash,
		RemoteURL:          st.RemoteURL,
		StatusLabel:        statusLabel(st.Staged > 0 || st.Modified > 0 || st.Untracked > 0, st.Staged, st.Modified, st.Untracked),
		ChangedFiles:       mapChangedFiles(st.FileChanges),
		NextSteps:          []NextStepView{},
		AIReady:            cfg != nil && strings.TrimSpace(cfg.APIKey) != "",
	}
	if st.Detached {
		d.Branch = "detached HEAD"
	}
	if cfg != nil {
		d.Provider = string(cfg.Provider)
		d.Model = cfg.Model
	}
	if d.ChangedFiles == nil {
		d.ChangedFiles = []ChangedFileView{}
	}

	ov := &gitpkg.Overview{
		Staged:      st.Staged,
		Modified:    st.Modified,
		Untracked:   st.Untracked,
		FileChanges: st.FileChanges,
	}
	if idx := app.BuildCommitContextIndex(ov, cfg); idx != nil {
		ci := &CommitContextIndex{
			Score:           idx.Score,
			Level:           idx.Level,
			Label:           idx.Label,
			RecommendCommit: idx.RecommendCommit,
			FileCount:       idx.FileCount,
			Insertions:      idx.Insertions,
			Deletions:       idx.Deletions,
			AreaCount:       idx.AreaCount,
			EstimatedBytes:  idx.EstimatedBytes,
			MaxDiffBytes:    idx.MaxDiffBytes,
			NearTruncate:    idx.NearTruncate,
		}
		if cfg != nil {
			ci.Model = cfg.Model
			ci.ModelContextWindow = app.ModelContextWindow(cfg.Model)
		}
		d.ContextIndex = ci
	}

	applyCICache(d)

	if compose := dockerpkg.DetectComposeFile(d.Path); compose != "" {
		d.HasDocker = true
		d.Docker = DockerStatus{
			Visible:     true,
			ComposeFile: compose,
			Summary:     "carregando…",
			Services:    []DockerServiceView{},
		}
	}

	return d, nil
}

// HygieneCountsView is local/remote prune candidate counts for the hygiene badge.
type HygieneCountsView struct {
	Local  int `json:"hygieneLocal"`
	Remote int `json:"hygieneRemote"`
}

// CIBadgeView is the CI status chip for the dashboard.
type CIBadgeView struct {
	CIState     string `json:"ciState,omitempty"`
	CILabel     string `json:"ciLabel,omitempty"`
	CIFromCache bool   `json:"ciFromCache,omitempty"`
	CIHost      string `json:"ciHost,omitempty"`
}

// ChangedFilesView is the enriched file list (+/- and context index).
type ChangedFilesView struct {
	ChangedFiles []ChangedFileView   `json:"changedFiles"`
	ContextIndex *CommitContextIndex `json:"contextIndex,omitempty"`
}

// LoadHygieneCounts runs the (potentially slow) prune-candidate scan.
func LoadHygieneCounts(projectPath, baseBranch string) (*HygieneCountsView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	baseBranch = strings.TrimSpace(baseBranch)
	if baseBranch == "" {
		baseBranch = "main"
	}
	local, remote, err := app.CountHygieneCandidates(projectPath, baseBranch)
	if err != nil {
		return nil, err
	}
	return &HygieneCountsView{Local: local, Remote: remote}, nil
}

// LoadCIBadge fetches live CI summary (falls back to disk cache on error).
func LoadCIBadge(projectPath, branch string) (*CIBadgeView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	sum := LoadCISummaryForProject(projectPath, branch)
	if sum == nil {
		return &CIBadgeView{}, nil
	}
	return &CIBadgeView{
		CIState:     sum.State,
		CILabel:     sum.Label,
		CIFromCache: sum.FromCache,
		CIHost:      sum.Host,
	}, nil
}

// LoadChangedFiles reloads porcelain + numstat and rebuilds the context index.
func LoadChangedFiles(projectPath string) (*ChangedFilesView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	repo, err := gitpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, err
	}
	changes, err := repo.FileChanges(true)
	if err != nil {
		return nil, err
	}
	staged, modified, untracked, err := repo.WorktreeCounts()
	if err != nil {
		// Fall back to deriving dirty from file list length.
		staged, modified, untracked = 0, 0, 0
		for _, c := range changes {
			switch strings.ToLower(c.Status) {
			case "untracked":
				untracked++
			case "staged", "new", "renamed":
				staged++
			default:
				modified++
			}
		}
	}
	cfg, _ := config.Load()
	ov := &gitpkg.Overview{
		Staged:      staged,
		Modified:    modified,
		Untracked:   untracked,
		FileChanges: changes,
	}
	view := &ChangedFilesView{
		ChangedFiles: mapChangedFiles(changes),
	}
	if idx := app.BuildCommitContextIndex(ov, cfg); idx != nil {
		ci := &CommitContextIndex{
			Score:           idx.Score,
			Level:           idx.Level,
			Label:           idx.Label,
			RecommendCommit: idx.RecommendCommit,
			FileCount:       idx.FileCount,
			Insertions:      idx.Insertions,
			Deletions:       idx.Deletions,
			AreaCount:       idx.AreaCount,
			EstimatedBytes:  idx.EstimatedBytes,
			MaxDiffBytes:    idx.MaxDiffBytes,
			NearTruncate:    idx.NearTruncate,
		}
		if cfg != nil {
			ci.Model = cfg.Model
			ci.ModelContextWindow = app.ModelContextWindow(cfg.Model)
		}
		view.ContextIndex = ci
	}
	if view.ChangedFiles == nil {
		view.ChangedFiles = []ChangedFileView{}
	}
	return view, nil
}

func applyCICache(d *Dashboard) {
	if d == nil {
		return
	}
	sum := loadCISummaryCache(d.Path)
	if sum == nil {
		return
	}
	d.CIState = sum.State
	d.CILabel = sum.Label
	d.CIFromCache = sum.FromCache
	d.CIHost = sum.Host
}

// LoadOpenPR returns the open PR for the current branch, if any.
func LoadOpenPR(projectPath string) (*PRStatus, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	client, err := prpkg.Open(projectPath)
	if err != nil {
		return nil, nil
	}
	pr, err := client.ViewCurrent()
	if err != nil || pr == nil {
		return nil, nil
	}
	return mapPRStatus(pr), nil
}

func mapPRStatus(pr *prpkg.PRView) *PRStatus {
	if pr == nil {
		return nil
	}
	// Only surface open PRs in the dashboard toolbar (merged/closed must not pulse Merge).
	if !strings.EqualFold(strings.TrimSpace(pr.State), "OPEN") {
		return nil
	}
	return &PRStatus{
		URL:            pr.URL,
		Title:          pr.Title,
		State:          pr.State,
		Number:         pr.Number,
		IsDraft:        pr.IsDraft,
		HeadRefName:    pr.HeadRefName,
		Mergeable:      pr.Mergeable,
		ReviewDecision: pr.ReviewDecision,
		ChecksPass:     pr.ChecksPass,
		ChecksFail:     pr.ChecksFail,
		ChecksPending:  pr.ChecksPending,
		ChecksTotal:    pr.ChecksTotal,
		ChecksSummary:  pr.ChecksSummary,
	}
}

// FromSnapshot maps an app snapshot into the desktop DTO.
func FromSnapshot(projectPath string, snap *app.WorkspaceSnapshot) *Dashboard {
	d := &Dashboard{
		Path:      projectPath,
		HasGH:     snap.HasGH,
		HasDocker: snap.HasDocker,
		AIReady:   snap.ConfigErr == nil && snap.Config != nil && snap.Config.APIKey != "",
		NextSteps: make([]NextStepView, 0, len(snap.NextSteps)),
	}

	if snap.ConfigErr == nil && snap.Config != nil {
		d.Provider = string(snap.Config.Provider)
		d.Model = snap.Config.Model
	}

	if snap.Overview != nil {
		o := snap.Overview
		d.RepoName = filepath.Base(o.Root)
		if o.Root != "" {
			d.Path = o.Root
		}
		d.Branch = o.Branch
		d.Detached = o.Detached
		d.Dirty = o.IsDirty()
		d.Staged = o.Staged
		d.Modified = o.Modified
		d.Untracked = o.Untracked
		d.Ahead = o.Ahead
		d.Behind = o.Behind
		d.HasUpstream = strings.TrimSpace(o.Upstream) != ""
		d.BaseBranch = o.BaseBranch
		d.CommitsAheadOfBase = o.CommitsAheadOfBase
		d.HasBranchDiff = o.HasBranchDiff
		d.BaseBehind = o.BaseBehind
		d.HeadHash = o.HeadHash
		d.RemoteURL = o.RemoteURL
		d.StatusLabel = statusLabel(o.IsDirty(), o.Staged, o.Modified, o.Untracked)
		d.ChangedFiles = mapChangedFiles(o.FileChanges)
		if o.Detached {
			d.Branch = "detached HEAD"
		}
	} else {
		d.RepoName = filepath.Base(projectPath)
	}
	if d.ChangedFiles == nil {
		d.ChangedFiles = []ChangedFileView{}
	}

	if idx := app.BuildCommitContextIndex(snap.Overview, snap.Config); idx != nil {
		ci := &CommitContextIndex{
			Score:           idx.Score,
			Level:           idx.Level,
			Label:           idx.Label,
			RecommendCommit: idx.RecommendCommit,
			FileCount:       idx.FileCount,
			Insertions:      idx.Insertions,
			Deletions:       idx.Deletions,
			AreaCount:       idx.AreaCount,
			EstimatedBytes:  idx.EstimatedBytes,
			MaxDiffBytes:    idx.MaxDiffBytes,
			NearTruncate:    idx.NearTruncate,
		}
		if snap.Config != nil {
			ci.Model = snap.Config.Model
			ci.ModelContextWindow = app.ModelContextWindow(snap.Config.Model)
		}
		d.ContextIndex = ci
	}

	d.Docker = mapDocker(snap.Docker, snap.HasDocker)

	if snap.OpenPR != nil {
		d.OpenPR = mapPRStatus(snap.OpenPR)
	}

	for _, step := range snap.NextSteps {
		d.NextSteps = append(d.NextSteps, NextStepView{
			Command: step.Command,
			Note:    step.Note,
		})
	}

	return d
}

func mapDocker(ov *dockerpkg.Overview, hasDocker bool) DockerStatus {
	st := DockerStatus{
		Available: hasDocker,
		Visible:   false,
		Services:  []DockerServiceView{},
	}
	if ov == nil {
		return st
	}
	st.Available = ov.Available
	st.DaemonRunning = ov.DaemonRunning
	st.ComposeFile = ov.ComposeFile
	st.Summary = ov.SummaryLine()
	st.Total = len(ov.Containers)
	st.DefaultService = ov.DefaultService()
	st.Services = make([]DockerServiceView, 0, len(ov.Containers))
	for _, c := range ov.Containers {
		if strings.EqualFold(c.State, "running") {
			st.Running++
		}
		if strings.TrimSpace(c.Service) == "" {
			continue
		}
		st.Services = append(st.Services, DockerServiceView{
			Name:      c.Service,
			State:     c.State,
			Container: c.Name,
			Ports:     c.Ports,
			Health:    c.Health,
		})
	}
	// When no containers yet (Down), still expose compose service names for the UI.
	if len(st.Services) == 0 && strings.TrimSpace(ov.ComposeFile) != "" && ov.DaemonRunning {
		if names, err := dockerpkg.ListComposeServices(ov.ComposeFile); err == nil {
			st.Services = make([]DockerServiceView, 0, len(names))
			for _, name := range names {
				st.Services = append(st.Services, DockerServiceView{Name: name})
			}
			if st.DefaultService == "" && len(names) > 0 {
				st.DefaultService = names[0]
			}
		}
	}
	// Show docker block when CLI exists or the repo has Compose (so UI can warn
	// that the environment is stopped / Docker is missing).
	st.Visible = ov.Available || strings.TrimSpace(ov.ComposeFile) != ""
	return st
}

func statusLabel(dirty bool, staged, modified, untracked int) string {
	if !dirty {
		return "clean"
	}
	parts := make([]string, 0, 3)
	if staged > 0 {
		parts = append(parts, fmt.Sprintf("%d staged", staged))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", untracked))
	}
	if len(parts) == 0 {
		return "dirty"
	}
	return strings.Join(parts, " · ")
}
