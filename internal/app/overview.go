package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/laerciocrestani/gitia/internal/config"
	gitpkg "github.com/laerciocrestani/gitia/internal/git"
	prpkg "github.com/laerciocrestani/gitia/internal/pr"
	"github.com/laerciocrestani/gitia/internal/ui"
)

func RunOverview() error {
	sess := ui.New("overview", false)
	sess.Header()

	repo, err := gitpkg.New()
	if err != nil {
		return err
	}
	if err := repo.IsRepo(); err != nil {
		return fmt.Errorf("diretório atual não é um repositório git")
	}

	baseBranch := "main"
	if cfg, err := config.Load(); err == nil {
		baseBranch = cfg.BaseBranch
	}

	var overview *gitpkg.Overview
	var openPR *prpkg.PRView

	if err := sess.StepQuiet(func() error {
		var err error
		overview, err = repo.Overview(baseBranch)
		return err
	}); err != nil {
		return err
	}

	if hasGH() {
		_ = sess.StepQuiet(func() error {
			client, err := prpkg.New()
			if err != nil {
				return nil
			}
			openPR, err = client.ViewCurrent()
			return nil
		})
	}

	fmt.Println()
	printGitiaConfig(sess)
	printRecentCommits(sess, overview)
	printBranches(sess, overview)
	printChangedFiles(sess, overview)
	printStash(sess, overview)

	sess.Divider()
	printRepoMeta(sess, overview, openPR)
	sess.Divider()
	printSuggestions(sess, overview, openPR)
	sess.Footer()
	return nil
}

func printRepoMeta(sess *ui.Session, o *gitpkg.Overview, pr *prpkg.PRView) {
	sess.MetaRow("Repository", repoDisplayName(o))
	if o.Detached {
		sess.MetaRow("Branch", "detached HEAD")
	} else {
		sess.MetaRow("Branch", o.Branch)
	}
	sess.MetaRow("Status", sess.StatusValue(o.IsDirty(), o.Staged, o.Modified, o.Untracked))
	if o.Upstream != "" {
		sess.MetaRow("Sync", syncLabel(o.Ahead, o.Behind))
	}
	if pr != nil {
		state := strings.ToLower(pr.State)
		if pr.IsDraft {
			state = "draft"
		}
		sess.MetaRow("Pull request", fmt.Sprintf("#%d %s (%s)", pr.Number, truncate(pr.Title, 40), state))
	}
}

func printRecentCommits(sess *ui.Session, o *gitpkg.Overview) {
	if len(o.RecentCommits) == 0 {
		return
	}
	sess.Section("Recent commits")
	for _, line := range o.RecentCommits {
		sess.Bullet(line)
	}
}

func printBranches(sess *ui.Session, o *gitpkg.Overview) {
	if len(o.Branches) == 0 {
		return
	}
	sess.Section("Branches")
	limit := len(o.Branches)
	if limit > 8 {
		limit = 8
	}
	for _, b := range o.Branches[:limit] {
		sess.BranchLine(b.Name, b.Current, b.Upstream, b.Ahead, b.Behind)
	}
	if len(o.Branches) > 8 {
		sess.Detail(fmt.Sprintf("… +%d more", len(o.Branches)-8))
	}
	if o.CommitsAheadOfBase > 0 && !o.Detached && o.Branch != o.BaseBranch {
		sess.Detail(fmt.Sprintf("%d commit(s) ahead of %s", o.CommitsAheadOfBase, o.BaseBranch))
	}
}

func printChangedFiles(sess *ui.Session, o *gitpkg.Overview) {
	if len(o.FileChanges) == 0 {
		return
	}
	sess.Section("Changed files")
	limit := len(o.FileChanges)
	if limit > 12 {
		limit = 12
	}
	for _, f := range o.FileChanges[:limit] {
		sess.FileChange(f.Path, f.Status, f.StatsLabel())
	}
	if len(o.FileChanges) > 12 {
		sess.Detail(fmt.Sprintf("… +%d more file(s)", len(o.FileChanges)-12))
	}
}

func printStash(sess *ui.Session, o *gitpkg.Overview) {
	if len(o.Stashes) == 0 {
		return
	}
	sess.Section("Stash")
	sess.KV("Entries", fmt.Sprintf("%d saved", len(o.Stashes)))
	limit := len(o.Stashes)
	if limit > 5 {
		limit = 5
	}
	for _, stash := range o.Stashes[:limit] {
		label := stash.Ref
		if stash.Branch != "" {
			label += " on " + stash.Branch
		}
		if stash.Message != "" {
			label += ": " + stash.Message
		}
		sess.Bullet(label)
	}
	if len(o.Stashes) > 5 {
		sess.Detail(fmt.Sprintf("… +%d more stash(es)", len(o.Stashes)-5))
	}
}

func printGitiaConfig(sess *ui.Session) {
	sess.Section("Gitia config")
	cfg, err := config.Load()
	if err != nil {
		sess.KV("Status", "not configured — run: gitia config")
		return
	}
	sess.KV("Provider", string(cfg.Provider))
	sess.KV("Model", cfg.Model)
	sess.KV("API key", config.MaskAPIKey(cfg.APIKey))
	if cfg.ClearScreen {
		sess.KV("Terminal", "limpa antes de cada comando")
	}
}

func printSuggestions(sess *ui.Session, o *gitpkg.Overview, pr *prpkg.PRView) {
	var tips []string

	if _, err := config.Load(); err != nil {
		tips = append(tips, "gitia config")
	}
	if o.IsDirty() {
		tips = append(tips, "gitia commit")
	}
	if o.Ahead > 0 || (o.IsDirty() && o.Upstream != "") {
		tips = append(tips, "gitia push")
	}
	if pr == nil && o.CommitsAheadOfBase > 0 && !o.IsDirty() {
		tips = append(tips, "gitia pr")
	}
	if pr != nil {
		tips = append(tips, "gh pr view --web")
	}
	if len(o.Stashes) > 0 {
		tips = append(tips, "git stash pop")
	}
	if o.Behind > 0 {
		tips = append(tips, "gitia sync")
	}
	if len(tips) == 0 && !o.IsDirty() {
		tips = append(tips, "working tree clean")
	}

	sess.Section("Next steps")
	for _, tip := range tips {
		if strings.Contains(tip, " ") && !strings.HasPrefix(tip, "gitia") && !strings.HasPrefix(tip, "git ") && !strings.HasPrefix(tip, "gh ") {
			sess.Bullet(tip)
		} else {
			sess.CommandHint(tip)
		}
	}

	if hasGH() {
		return
	}
	sess.Detail("install gh for PR info — https://cli.github.com/")
}

func repoDisplayName(o *gitpkg.Overview) string {
	if o.RemoteURL != "" {
		name := o.RemoteURL
		name = strings.TrimSuffix(name, ".git")
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		if i := strings.LastIndex(name, ":"); i >= 0 {
			name = name[i+1:]
		}
		if name != "" {
			return name
		}
	}
	return filepath.Base(o.Root)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func syncLabel(ahead, behind int) string {
	switch {
	case ahead > 0 && behind > 0:
		return fmt.Sprintf("↑%d ahead · ↓%d behind", ahead, behind)
	case ahead > 0:
		return fmt.Sprintf("↑%d ahead of remote", ahead)
	case behind > 0:
		return fmt.Sprintf("↓%d behind remote", behind)
	default:
		return "in sync with remote"
	}
}

func hasGH() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}
