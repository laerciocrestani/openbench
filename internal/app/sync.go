package app

import (
	"fmt"

	"github.com/laerciocrestani/gitai/internal/config"
	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/ui"
)

type SyncOptions struct {
	Prune       bool
	PruneRemote bool
	Base        string
	DryRun      bool
	Progress    Progress
}

func RunSync(opts SyncOptions) error {
	prog := opts.Progress
	if prog == nil {
		sess := ui.New("sync", opts.DryRun)
		sess.Header()
		prog = sess
	}

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
	if sess, ok := prog.(*ui.Session); ok {
		sess.MetaRow("Base", base)
		if previous != base && !opts.shouldPrune() {
			sess.MetaRow("Branch", previous)
		}
		sess.Divider()
	}

	if err := prog.Step("Fetching origin", func() error {
		if opts.DryRun {
			prog.Detail("git fetch origin --prune")
			return nil
		}
		return repo.FetchPrune()
	}); err != nil {
		return err
	}

	if err := prog.Step("Pulling "+base, func() error {
		if opts.DryRun {
			prog.Detail(fmt.Sprintf("git checkout %s && git pull --ff-only origin %s", base, base))
			return nil
		}
		return repo.PullBase(base)
	}); err != nil {
		return err
	}

	if !opts.shouldPrune() {
		prog.Success("Synced with origin/" + base)
		return nil
	}

	var local, remote []string
	if err := prog.Step("Finding branches to prune", func() error {
		var err error
		if opts.pruneLocal() {
			local, err = repo.LocalPruneCandidates(base)
			if err != nil {
				return err
			}
		}
		if opts.pruneRemote() {
			remote, err = repo.RemotePruneCandidates(base)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if len(local) == 0 && len(remote) == 0 {
		prog.Info("No branches to prune")
		prog.Success("Synced with origin/" + base)
		return nil
	}

	if sess, ok := prog.(*ui.Session); ok {
		sess.Section("Prune")
	}

	localRemoved := 0
	for _, name := range local {
		removed, err := pruneLocal(prog, repo, name, base, opts.DryRun)
		if err != nil {
			return err
		}
		if removed {
			localRemoved++
		}
	}

	remoteRemoved := 0
	for _, name := range remote {
		if err := pruneRemote(prog, repo, name, opts.DryRun); err != nil {
			return err
		}
		remoteRemoved++
	}

	msg := "Synced"
	if localRemoved > 0 {
		msg += fmt.Sprintf(" · %d local removed", localRemoved)
	}
	if remoteRemoved > 0 {
		msg += fmt.Sprintf(" · %d remote removed", remoteRemoved)
	}
	prog.Success(msg)
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

func pruneLocal(prog Progress, repo *gitpkg.Repo, name, base string, dryRun bool) (bool, error) {
	issue, err := repo.LocalBranchPruneIssue(name)
	if err != nil {
		return false, err
	}

	force := false
	if issue != nil && issue.UpstreamGone {
		if dryRun {
			prog.Info(name + ": upstream removido no remoto — usaria git branch -D")
			return false, nil
		}
		force = true
	} else if issue != nil && issue.LocalAhead > 0 {
		if dryRun {
			prog.Info(fmt.Sprintf(
				"%s: diverge de %s (%d commit(s) local não enviado(s)) — usaria -D após confirmação",
				name, issue.Upstream, issue.LocalAhead,
			))
			for _, commit := range issue.LocalCommits {
				prog.Detail(commit)
			}
			return false, nil
		}

		sess, ok := prog.(*ui.Session)
		if !ok {
			rec := RecommendPruneBranchAction(issue)
			if rec.Action == PruneBranchKeep {
				prog.Info("Mantida: " + name + " — " + rec.Reason)
				return false, nil
			}
			force = true
		} else {
			action, err := promptPruneBranchConflict(sess, issue)
			if err != nil {
				return false, err
			}
			if action == PruneBranchKeep {
				prog.Info("Mantida: " + name)
				return false, nil
			}
			force = true
		}
	}

	if !force {
		absorbed, err := repo.BranchAbsorbedIntoBase(name, base)
		if err != nil {
			return false, err
		}
		if absorbed {
			if dryRun {
				prog.Info(name + ": alterações já estão na base (squash/rebase) — usaria git branch -D")
				return false, nil
			}
			force = true
		}
	}

	label := "Removing local " + name
	if force {
		label += " (forced)"
	}

	err = prog.Step(label, func() error {
		if dryRun {
			if force {
				prog.Detail("git branch -D " + name)
			} else {
				prog.Detail("git branch -d " + name)
			}
			return nil
		}
		if force {
			return repo.DeleteLocalBranchForce(name)
		}
		return repo.DeleteLocalBranch(name)
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func pruneRemote(prog Progress, repo *gitpkg.Repo, name string, dryRun bool) error {
	return prog.Step("Removing remote "+name, func() error {
		if dryRun {
			prog.Detail("git push origin --delete " + name)
			return nil
		}
		return repo.DeleteRemoteBranch(name)
	})
}
