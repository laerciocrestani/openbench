package git

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// BranchDetail holds contextual information about a branch vs the configured base.
type BranchDetail struct {
	Info               BranchInfo
	HeadHash           string
	CommitsAheadOfBase int
	RecentCommits      []string
	FilesChanged       int
	Insertions         int
	Deletions          int
}

const maxBranchLogPreview = 5

// ListBranches returns local branches with upstream tracking info.
func (r *Repo) ListBranches() ([]BranchInfo, error) {
	return r.listBranches()
}

// Checkout switches to the given branch.
func (r *Repo) Checkout(branch string) error {
	current, err := r.CurrentBranch()
	if err != nil {
		return err
	}
	if current == branch {
		return nil
	}
	_, err = r.run("checkout", branch)
	return err
}

// BranchDetail loads summary information for a branch relative to base.
func (r *Repo) BranchDetail(name, base string) (*BranchDetail, error) {
	resolved, err := r.ResolveBase(base)
	if err != nil {
		return nil, err
	}

	info := BranchInfo{Name: name}
	branches, err := r.ListBranches()
	if err != nil {
		return nil, err
	}
	for _, b := range branches {
		if b.Name == name {
			info = b
			break
		}
	}

	head, err := r.run("rev-parse", "--short", name)
	if err != nil {
		return nil, fmt.Errorf("branch %q: %w", name, err)
	}

	aheadOfBase := 0
	if count, err := r.run("rev-list", "--count", fmt.Sprintf("%s..%s", resolved, name)); err == nil {
		aheadOfBase, _ = strconv.Atoi(count)
	}

	var recent []string
	if aheadOfBase > 0 {
		recent, err = r.logOnelineRange(resolved, name, maxBranchLogPreview)
		if err != nil {
			return nil, err
		}
	}

	files, ins, del, err := r.branchDiffSummary(resolved, name)
	if err != nil {
		return nil, err
	}

	return &BranchDetail{
		Info:               info,
		HeadHash:           head,
		CommitsAheadOfBase: aheadOfBase,
		RecentCommits:      recent,
		FilesChanged:       files,
		Insertions:         ins,
		Deletions:          del,
	}, nil
}

var shortStatRe = regexp.MustCompile(`(\d+) files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`)

func (r *Repo) branchDiffSummary(base, branch string) (files, insertions, deletions int, err error) {
	out, err := r.run("diff", "--shortstat", fmt.Sprintf("%s...%s", base, branch))
	if err != nil {
		return 0, 0, 0, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return 0, 0, 0, nil
	}
	m := shortStatRe.FindStringSubmatch(out)
	if m == nil {
		return 0, 0, 0, nil
	}
	files, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		insertions, _ = strconv.Atoi(m[2])
	}
	if m[3] != "" {
		deletions, _ = strconv.Atoi(m[3])
	}
	return files, insertions, deletions, nil
}
