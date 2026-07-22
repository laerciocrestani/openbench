package git

import (
	"fmt"
	"strconv"
	"strings"
)

// PullFFOnly fast-forwards the current branch from its upstream.
func (r *Repo) PullFFOnly() error {
	_, err := r.run("pull", "--ff-only")
	return err
}

// BaseBehindOrigin returns how many commits local base is behind origin/<base>.
func (r *Repo) BaseBehindOrigin(base string) (int, error) {
	local := strings.TrimPrefix(strings.TrimSpace(base), "origin/")
	if local == "" {
		return 0, fmt.Errorf("base branch vazia")
	}
	remote := "origin/" + local
	if _, err := r.run("rev-parse", "--verify", local); err != nil {
		return 0, nil
	}
	if _, err := r.run("rev-parse", "--verify", remote); err != nil {
		return 0, nil
	}
	out, err := r.run("rev-list", "--count", fmt.Sprintf("%s..%s", local, remote))
	if err != nil {
		return 0, err
	}
	n, _ := strconv.Atoi(out)
	return n, nil
}

// UpdateLocalBranchFromOrigin fast-forwards local branch to match origin/<branch>
// without checking it out. Returns whether the ref moved.
func (r *Repo) UpdateLocalBranchFromOrigin(branch string) (updated bool, err error) {
	local := strings.TrimPrefix(strings.TrimSpace(branch), "origin/")
	if local == "" {
		return false, fmt.Errorf("branch vazia")
	}
	remote := "origin/" + local

	if _, err := r.run("rev-parse", "--verify", remote); err != nil {
		return false, nil
	}
	if _, err := r.run("rev-parse", "--verify", local); err != nil {
		// Local branch missing — create it pointing at origin (no checkout).
		if _, err := r.run("branch", local, remote); err != nil {
			return false, fmt.Errorf("criar %s a partir de %s: %w", local, remote, err)
		}
		return true, nil
	}

	ahead, behind, err := r.BranchAheadBehind(local, remote)
	if err != nil {
		return false, err
	}
	if behind == 0 {
		return false, nil
	}
	if ahead > 0 {
		return false, fmt.Errorf("%s divergiu de %s (↑%d ↓%d) — resolva manualmente", local, remote, ahead, behind)
	}

	spec := fmt.Sprintf("%s:%s", local, local)
	if _, err := r.run("fetch", "origin", spec); err != nil {
		return false, fmt.Errorf("atualizar %s: %w", local, err)
	}
	return true, nil
}
