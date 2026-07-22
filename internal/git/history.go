package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// RevertCommit creates a revert commit for hash.
// For merge commits, mainline must be 1 (usually the first parent / target branch).
func (r *Repo) RevertCommit(hash string, isMerge bool) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return fmt.Errorf("hash vazio")
	}
	clean, err := r.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("working tree com alterações — commit ou stash antes de reverter")
	}

	args := []string{"revert", "--no-edit"}
	if isMerge {
		args = append(args, "-m", "1")
	}
	args = append(args, hash)
	_, err = r.run(args...)
	return err
}

// ResetMode selects git reset mode.
type ResetMode string

const (
	ResetSoft  ResetMode = "soft"
	ResetMixed ResetMode = "mixed"
	ResetHard  ResetMode = "hard"
)

// ResetTo moves HEAD to hash with the given mode.
// Hard reset requires a clean tree check is skipped intentionally — caller must confirm.
func (r *Repo) ResetTo(hash string, mode ResetMode) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return fmt.Errorf("hash vazio")
	}
	flag := "--mixed"
	switch mode {
	case ResetSoft:
		flag = "--soft"
	case ResetMixed:
		flag = "--mixed"
	case ResetHard:
		flag = "--hard"
	default:
		return fmt.Errorf("modo de reset inválido: %s", mode)
	}
	if mode != ResetHard {
		clean, err := r.IsClean()
		if err != nil {
			return err
		}
		if !clean {
			return fmt.Errorf("working tree com alterações — commit ou stash antes do reset")
		}
	}
	_, err := r.run("reset", flag, hash)
	return err
}

// IsAncestor reports whether ancestor is an ancestor of HEAD (inclusive).
func (r *Repo) IsAncestor(ancestor string) (bool, error) {
	ancestor = strings.TrimSpace(ancestor)
	if ancestor == "" {
		return false, fmt.Errorf("hash vazio")
	}
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, "HEAD")
	cmd.Dir = r.dir
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}
