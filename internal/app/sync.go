package app

import (
	"fmt"

	"github.com/laerciocrestani/gitai/internal/config"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/ui"
)

type SyncOptions struct {
	Prune       bool // local + remote
	PruneRemote bool // só remoto (GitHub)
	Base        string
	DryRun      bool
}

func RunSync(opts SyncOptions) error {
	sess := ui.New("sync", opts.DryRun)
	sess.Header()

	repo, err := gitpkg.New()
	if err != nil {
		return err
	}
	if err := repo.IsRepo(); err != nil {
		return fmt.Errorf("diretório atual não é um repositório git")
	}

	clean, err := repo.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("working tree com alterações — commit ou stash antes de sincronizar")
	}

	base := opts.Base
	if base == "" {
		if cfg, err := config.Load(); err == nil {
			base = cfg.BaseBranch
		}
	}
	if base == "" {
		base = "main"
	}

	previous, err := repo.CurrentBranch()
	if err != nil {
		return err
	}

	fmt.Println()
	sess.MetaRow("Base", base)
	if previous != base && !opts.shouldPrune() {
		sess.MetaRow("Branch", previous)
	}
	sess.Divider()

	if err := sess.Step("Fetching origin", func() error {
		if opts.DryRun {
			sess.Detail("git fetch origin --prune")
			return nil
		}
		return repo.FetchPrune()
	}); err != nil {
		return err
	}

	if err := sess.Step("Pulling "+base, func() error {
		if opts.DryRun {
			sess.Detail(fmt.Sprintf("git checkout %s && git pull --ff-only origin %s", base, base))
			return nil
		}
		return repo.PullBase(base)
	}); err != nil {
		return err
	}

	if !opts.shouldPrune() {
		sess.Success("Synced with origin/" + base)
		return nil
	}

	var local, remote []string
	if err := sess.Step("Finding merged branches", func() error {
		var err error
		if opts.pruneLocal() {
			local, err = repo.MergedLocalBranches(base)
			if err != nil {
				return err
			}
		}
		if opts.pruneRemote() {
			remote, err = repo.MergedRemoteBranches(base)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if len(local) == 0 && len(remote) == 0 {
		sess.Info("No merged branches to prune")
		sess.Success("Synced with origin/" + base)
		return nil
	}

	sess.Section("Prune")
	for _, name := range local {
		if err := pruneLocal(sess, repo, name, opts.DryRun); err != nil {
			return err
		}
	}
	for _, name := range remote {
		if err := pruneRemote(sess, repo, name, opts.DryRun); err != nil {
			return err
		}
	}

	msg := "Synced"
	if len(local) > 0 {
		msg += fmt.Sprintf(" · %d local removed", len(local))
	}
	if len(remote) > 0 {
		msg += fmt.Sprintf(" · %d remote removed", len(remote))
	}
	sess.Success(msg)
	return nil
}

func (o SyncOptions) shouldPrune() bool {
	return o.Prune || o.PruneRemote
}

func (o SyncOptions) pruneLocal() bool {
	return o.Prune
}

func (o SyncOptions) pruneRemote() bool {
	return o.Prune || o.PruneRemote
}

func pruneLocal(sess *ui.Session, repo *gitpkg.Repo, name string, dryRun bool) error {
	return sess.Step("Removing local "+name, func() error {
		if dryRun {
			sess.Detail("git branch -d " + name)
			return nil
		}
		return repo.DeleteLocalBranch(name)
	})
}

func pruneRemote(sess *ui.Session, repo *gitpkg.Repo, name string, dryRun bool) error {
	return sess.Step("Removing remote "+name, func() error {
		if dryRun {
			sess.Detail("git push origin --delete " + name)
			return nil
		}
		return repo.DeleteRemoteBranch(name)
	})
}
