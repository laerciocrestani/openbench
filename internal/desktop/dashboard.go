package desktop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/laerciocrestani/openbench/internal/app"
	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
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
	BaseBranch         string              `json:"baseBranch"`
	CommitsAheadOfBase int                 `json:"commitsAheadOfBase"`
	HasBranchDiff      bool                `json:"hasBranchDiff"`
	BaseBehind         int                 `json:"baseBehind"`
	HeadHash           string              `json:"headHash"`
	RemoteURL          string              `json:"remoteURL"`
	StatusLabel        string              `json:"statusLabel"`
	HasGH              bool                `json:"hasGH"`
	HasDocker          bool                `json:"hasDocker"`
	Docker             DockerStatus        `json:"docker"`
	OpenPR             *PRStatus           `json:"openPR,omitempty"`
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

// LoadDashboard builds a desktop dashboard for projectPath.
// Docker and open-PR checks are skipped here (slow CLI); refresh them via
// LoadDockerStatus / LoadOpenPR after the UI is shown.
func LoadDashboard(projectPath string) (*Dashboard, error) {
	snap, err := app.LoadWorkspaceSnapshotAtOpts(projectPath, nil, app.SnapshotOpts{
		SkipDocker: true,
		SkipPR:     true,
	})
	if err != nil {
		return nil, err
	}
	d := FromSnapshot(projectPath, snap)
	if d.HasDocker {
		d.Docker = DockerStatus{
			Available: true,
			Visible:   true,
			Summary:   "carregando…",
			Services:  []DockerServiceView{},
		}
	}
	return d, nil
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
	return &PRStatus{
		URL:            pr.URL,
		Title:          pr.Title,
		State:          pr.State,
		Number:         pr.Number,
		IsDraft:        pr.IsDraft,
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
	// Show docker block when CLI exists (even if daemon down) — matches discovery.
	st.Visible = ov.Available
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
