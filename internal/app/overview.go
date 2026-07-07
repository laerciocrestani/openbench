package app

import (
	"fmt"
	"path/filepath"
	"strings"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	prpkg "github.com/laerciocrestani/gitai/internal/pr"
	"github.com/laerciocrestani/gitai/internal/ui"
)

func RunOverview() error {
	sess := ui.New("overview", false)

	var snap *WorkspaceSnapshot
	if err := sess.StepQuiet(func() error {
		var err error
		snap, err = LoadWorkspaceSnapshot()
		return err
	}); err != nil {
		return err
	}

	sess.HeaderWithContext(bannerContext(snap))
	printRecentCommits(sess, snap.Overview)
	printBranches(sess, snap.Overview)
	printChangedFiles(sess, snap.Overview)
	printStash(sess, snap.Overview)

	sess.Divider()
	printRepoMeta(sess, snap.Overview, snap.OpenPR)
	sess.Divider()
	printSuggestions(sess, snap)
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
	sess.SectionFirst("Recent commits")
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

func bannerContext(snap *WorkspaceSnapshot) ui.BannerContext {
	ctx := ui.BannerContext{}
	if snap.Overview != nil {
		o := snap.Overview
		ctx.Repo = repoDisplayName(o)
		if o.Detached {
			ctx.Branch = "detached HEAD"
		} else {
			ctx.Branch = o.Branch
		}
		ctx.Sync = bannerSyncLabel(o)
	}
	if snap.ConfigErr == nil && snap.Config != nil {
		ctx.Provider = string(snap.Config.Provider)
		ctx.Model = snap.Config.Model
	}
	return ctx
}

func bannerSyncLabel(o *gitpkg.Overview) string {
	if o.IsDirty() {
		n := o.Staged + o.Modified + o.Untracked
		if n == 1 {
			return "1 change"
		}
		return fmt.Sprintf("%d changes", n)
	}
	switch {
	case o.Ahead > 0 && o.Behind > 0:
		return fmt.Sprintf("↑%d ↓%d", o.Ahead, o.Behind)
	case o.Ahead > 0:
		return fmt.Sprintf("↑%d ahead", o.Ahead)
	case o.Behind > 0:
		return fmt.Sprintf("↓%d behind", o.Behind)
	default:
		return "in sync"
	}
}

func printSuggestions(sess *ui.Session, snap *WorkspaceSnapshot) {
	sess.Section("Next steps")
	for _, step := range snap.NextSteps {
		switch {
		case step.Plain:
			sess.Bullet(step.Command)
		case step.Muted && step.Note != "":
			sess.CommandHintMutedWithNote(step.Command, step.Note)
		case step.Muted:
			sess.CommandHintMuted(step.Command)
		case step.Note != "":
			sess.CommandHintWithNote(step.Command, step.Note)
		default:
			sess.CommandHint(step.Command)
		}
	}

	if snap.HasGH {
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

