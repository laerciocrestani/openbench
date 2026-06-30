package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Repo struct {
	dir string
}

func New() (*Repo, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &Repo{dir: dir}, nil
}

func (r *Repo) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r *Repo) AddAll() error {
	_, err := r.run("add", ".")
	return err
}

func (r *Repo) DiffStaged() (string, error) {
	return r.run("diff", "--cached")
}

func (r *Repo) DiffUnstaged() (string, error) {
	return r.run("diff")
}

func (r *Repo) DiffForCommit() (string, error) {
	diff, err := r.DiffStaged()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) != "" {
		return diff, nil
	}
	return r.DiffUnstaged()
}

func (r *Repo) DiffBranch(base string) (string, error) {
	return r.run("diff", fmt.Sprintf("%s...HEAD", base))
}

func (r *Repo) CurrentBranch() (string, error) {
	return r.run("rev-parse", "--abbrev-ref", "HEAD")
}

func (r *Repo) Commit(message string) error {
	_, err := r.run("commit", "-m", message)
	return err
}

func (r *Repo) Push() error {
	branch, err := r.CurrentBranch()
	if err != nil {
		return err
	}
	_, err = r.run("push", "-u", "origin", branch)
	return err
}

func (r *Repo) IsRepo() error {
	_, err := r.run("rev-parse", "--git-dir")
	return err
}
