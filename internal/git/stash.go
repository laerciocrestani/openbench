package git

import (
	"fmt"
	"strings"
)

// StashPushAll saves tracked and untracked changes (does not include ignored files).
func (r *Repo) StashPushAll(message string) error {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "openbench-doctor-wip"
	}
	_, err := r.run("stash", "push", "-u", "-m", msg)
	return err
}

// StashPop applies the most recent stash.
func (r *Repo) StashPop() error {
	_, err := r.run("stash", "pop")
	return err
}

// LocalBranchExists reports whether a local branch ref exists.
func (r *Repo) LocalBranchExists(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	_, err := r.run("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// RebaseUpstream rebases the current branch onto its upstream.
func (r *Repo) RebaseUpstream() error {
	_, err := r.run("rebase", "@{u}")
	return err
}

// ResetBranchToOrigin moves local branch to match origin/<branch>.
// If the branch is checked out, uses reset --hard; otherwise branch -f.
func (r *Repo) ResetBranchToOrigin(branch string) error {
	branch = strings.TrimPrefix(strings.TrimSpace(branch), "origin/")
	if branch == "" {
		return fmt.Errorf("branch vazia")
	}
	remote := "origin/" + branch
	if _, err := r.run("rev-parse", "--verify", remote); err != nil {
		return fmt.Errorf("ref %s não encontrada — rode fetch", remote)
	}
	current, err := r.CurrentBranch()
	if err != nil {
		return err
	}
	if current == branch {
		_, err = r.run("reset", "--hard", remote)
		return err
	}
	_, err = r.run("branch", "-f", branch, remote)
	return err
}

// PushBranch pushes a local branch to origin.
func (r *Repo) PushBranch(branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("branch vazia")
	}
	_, err := r.run("push", "-u", "origin", branch)
	return err
}

// RebaseOnto rebases the current branch onto ref (e.g. origin/main).
func (r *Repo) RebaseOnto(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("ref vazia")
	}
	_, err := r.run("rebase", ref)
	return err
}
