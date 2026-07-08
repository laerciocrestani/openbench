package app

import (
	"os/exec"

	"github.com/laerciocrestani/gitai/internal/config"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	prpkg "github.com/laerciocrestani/gitai/internal/pr"
)

// WorkspaceSnapshot agrega o estado read-only do repositório para o dashboard TUI.
type WorkspaceSnapshot struct {
	Overview  *gitpkg.Overview
	OpenPR    *prpkg.PRView
	Config    *config.Config
	ConfigErr error
	NextSteps []NextStep
	HasGH     bool
}

// LoadWorkspaceSnapshot coleta overview, PR aberto, config e próximos passos.
func LoadWorkspaceSnapshot() (*WorkspaceSnapshot, error) {
	return LoadWorkspaceSnapshotWithProgress(nil)
}

// LoadWorkspaceSnapshotWithProgress coleta o snapshot reportando etapas ao Progress.
func LoadWorkspaceSnapshotWithProgress(prog Progress) (*WorkspaceSnapshot, error) {
	var repo *gitpkg.Repo

	step := func(label string, fn func() error) error {
		if prog == nil {
			return fn()
		}
		return prog.Step(label, fn)
	}

	if err := step("Opening repository", func() error {
		r, err := gitpkg.New()
		if err != nil {
			return err
		}
		repo = r
		return repo.IsRepo()
	}); err != nil {
		return nil, err
	}

	baseBranch := "main"
	var cfg *config.Config
	var cfgErr error

	if err := step("Loading configuration", func() error {
		cfg, cfgErr = config.Load()
		if cfgErr == nil {
			baseBranch = cfg.BaseBranch
		}
		return nil
	}); err != nil {
		return nil, err
	}

	var overview *gitpkg.Overview
	if err := step("Reading workspace", func() error {
		var err error
		overview, err = repo.Overview(baseBranch)
		return err
	}); err != nil {
		return nil, err
	}

	snap := &WorkspaceSnapshot{
		Overview:  overview,
		Config:    cfg,
		ConfigErr: cfgErr,
		HasGH:     hasGH(),
	}

	if snap.HasGH {
		if err := step("Checking pull request", func() error {
			client, err := prpkg.New()
			if err != nil {
				return nil
			}
			snap.OpenPR, _ = client.ViewCurrent()
			return nil
		}); err != nil {
			return nil, err
		}
	}

	snap.NextSteps = buildNextSteps(overview, snap.OpenPR, cfgErr == nil)

	if prog != nil {
		prog.Success("Ready")
	}
	return snap, nil
}

func hasGH() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}
