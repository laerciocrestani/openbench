package app

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/config"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	"github.com/laerciocrestani/openbench/internal/ui"
)

// Hygiene mode identifiers (English CLI/API values).
const (
	HygieneModeFull  = "full"  // delete local + remote merged/absorbed branches
	HygieneModeLocal = "local" // delete local only; keep remote
)

// HygieneOptions configures branch cleanup (fetch refs + prune). Does not
// fast-forward the local base tip — use RunSync for that.
type HygieneOptions struct {
	Mode     string // full | local
	Base     string
	DryRun   bool
	WorkDir  string
	Progress Progress
}

// RunHygiene fetches origin refs and prunes merged/absorbed branches.
func RunHygiene(opts HygieneOptions) error {
	prog := opts.Progress
	if prog == nil {
		sess := ui.New("hygiene", opts.DryRun)
		sess.Header()
		prog = sess
	}

	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		mode = HygieneModeFull
	}
	localOnly := false
	switch mode {
	case HygieneModeFull:
	case HygieneModeLocal:
		localOnly = true
	default:
		return fmt.Errorf("modo de hygiene inválido: %s (use full ou local)", mode)
	}

	repo, err := openRepo(opts.WorkDir)
	if err != nil {
		return err
	}
	if err := repo.IsRepo(); err != nil {
		return fmt.Errorf("diretório atual não é um repositório git")
	}

	// Dirty working tree is usually OK for remote deletes, but local prune
	// may need checkout to base — that can fail if the tree blocks the switch.

	base := strings.TrimSpace(opts.Base)
	if base == "" {
		if cfg, err := config.Load(); err == nil {
			base = cfg.BaseBranch
		}
	}
	if base == "" {
		base = "main"
	}

	fmt.Println()
	if sess, ok := prog.(*ui.Session); ok {
		sess.MetaRow("Base", base)
		sess.MetaRow("Mode", mode)
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

	pruneOpts := SyncOptions{
		PruneLocalOnly: localOnly,
		Prune:          !localOnly,
		Base:           base,
		DryRun:         opts.DryRun,
		WorkDir:        opts.WorkDir,
		Progress:       prog,
	}

	// Local `git branch -d` checks merge into HEAD, and the current branch is
	// never a prune candidate. Switch to base first so hygiene can delete the
	// branch you were on (and so -d is evaluated against main/master).
	if pruneOpts.pruneLocal() {
		if err := ensureHygieneOnBase(prog, repo, base, opts.DryRun); err != nil {
			return err
		}
	}

	local, remote, err := discoverPruneCandidates(prog, repo, pruneOpts, base)
	if err != nil {
		return err
	}

	if len(local) == 0 && len(remote) == 0 {
		prog.Info("No branches to prune")
		prog.Success("Hygiene complete")
		return nil
	}

	if sess, ok := prog.(*ui.Session); ok {
		sess.Section("Prune")
	}

	remoteRemoved := 0
	if pruneOpts.pruneRemote() {
		remoteRemoved, err = pruneRemoteBranches(prog, repo, remote, opts.DryRun)
		if err != nil {
			return err
		}
		if remoteRemoved > 0 || (opts.DryRun && len(remote) > 0) {
			if err := refreshOriginAfterRemotePrune(prog, repo, opts.DryRun); err != nil {
				return err
			}
		}
		if pruneOpts.pruneLocal() && remoteRemoved > 0 {
			local, err = repo.LocalPruneCandidates(base)
			if err != nil {
				return err
			}
		}
	}

	localRemoved := 0
	if pruneOpts.pruneLocal() {
		localRemoved, err = pruneLocalBranches(prog, repo, local, base, opts.DryRun)
		if err != nil {
			return err
		}
	}

	msg := "Hygiene"
	if localRemoved > 0 {
		msg += fmt.Sprintf(" · %d local removed", localRemoved)
	}
	if remoteRemoved > 0 {
		msg += fmt.Sprintf(" · %d remote removed", remoteRemoved)
	}
	if localRemoved == 0 && remoteRemoved == 0 {
		msg += " · nothing removed"
	}
	prog.Success(msg)
	return nil
}

// ensureHygieneOnBase checks out the base branch before local prune.
func ensureHygieneOnBase(prog Progress, repo *gitpkg.Repo, base string, dryRun bool) error {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "main"
	}
	current, err := repo.CurrentBranch()
	if err != nil {
		return err
	}
	if strings.TrimSpace(current) == "" || current == base {
		return nil
	}

	return prog.Step("Switching to "+base+" before prune", func() error {
		if dryRun {
			prog.Detail("git checkout " + base)
			prog.Detail("(necessário: git branch -d valida merge no HEAD; branch atual não pode ser apagada)")
			return nil
		}
		if err := repo.Checkout(base); err != nil {
			return fmt.Errorf(
				"hygiene precisa fazer checkout em %s para apagar branches (atual: %s): %w — faça commit/stash se a working tree bloquear o checkout",
				base, current, err,
			)
		}
		prog.Detail("git checkout " + base)
		return nil
	})
}

// CountHygieneCandidates returns local/remote prune candidate counts for UI pulse.
// Includes the current branch when it would be pruned after switching to base
// (LocalPruneCandidates intentionally skips HEAD so git branch -d is safe).
func CountHygieneCandidates(workDir, base string) (local, remote int, err error) {
	repo, err := openRepo(workDir)
	if err != nil {
		return 0, 0, err
	}
	if err := repo.IsRepo(); err != nil {
		return 0, 0, err
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = "main"
	}
	locs, err := repo.LocalPruneCandidates(base)
	if err != nil {
		return 0, 0, err
	}
	if cur, err := repo.CurrentBranch(); err == nil {
		if cur != "" && cur != base && !containsString(locs, cur) {
			if ok, _ := branchIsHygieneCandidate(repo, cur, base); ok {
				locs = append(locs, cur)
			}
		}
	}
	rems, err := repo.RemotePruneCandidates(base)
	if err != nil {
		return 0, 0, err
	}
	return len(locs), len(rems), nil
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func branchIsHygieneCandidate(repo *gitpkg.Repo, name, base string) (bool, error) {
	merged, err := repo.BranchTipInBase(name, base)
	if err != nil {
		return false, err
	}
	if merged {
		return true, nil
	}
	return repo.BranchAbsorbedIntoBase(name, base)
}
