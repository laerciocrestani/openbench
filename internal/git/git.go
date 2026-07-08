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

func (r *Repo) HasStagedChanges() (bool, error) {
	diff, err := r.DiffStaged()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(diff) != "", nil
}

func (r *Repo) ResolveBase(preferred string) (string, error) {
	candidates := []string{preferred}
	if preferred != "" {
		candidates = append(candidates, "origin/"+preferred)
	}

	seen := make(map[string]bool)
	for _, ref := range candidates {
		ref = strings.TrimSpace(ref)
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		if _, err := r.run("rev-parse", "--verify", ref); err == nil {
			return ref, nil
		}
	}

	return "", fmt.Errorf("branch base %q não encontrada (tente git fetch)", preferred)
}

func (r *Repo) LogOnBranch(base string) (string, error) {
	return r.run("log", fmt.Sprintf("%s..HEAD", base), "--oneline", "--no-decorate")
}

// RecentLog returns recent commits as oneline output.
func (r *Repo) RecentLog(limit int) (string, error) {
	if limit <= 0 {
		limit = 30
	}
	return r.run("log", fmt.Sprintf("-%d", limit), "--oneline", "--decorate", "--graph")
}

func (r *Repo) IsSameAsBase(base string) (bool, error) {
	count, err := r.run("rev-list", "--count", fmt.Sprintf("%s..HEAD", base))
	if err != nil {
		return false, err
	}
	return count == "0", nil
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

func (r *Repo) ProjectName() string {
	url, err := r.run("remote", "get-url", "origin")
	if err == nil && url != "" {
		return extractRepoName(url)
	}
	parts := strings.Split(r.dir, string(os.PathSeparator))
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

func extractRepoName(remote string) string {
	remote = strings.TrimSpace(remote)
	remote = strings.TrimSuffix(remote, ".git")
	if i := strings.LastIndex(remote, "/"); i >= 0 {
		return remote[i+1:]
	}
	if i := strings.LastIndex(remote, ":"); i >= 0 {
		return remote[i+1:]
	}
	return remote
}

func (r *Repo) Status(args ...string) error {
	cmd := exec.Command("git", append([]string{"status"}, args...)...)
	cmd.Dir = r.dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
