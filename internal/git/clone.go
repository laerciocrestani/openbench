package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Init creates a new git repository in the repo directory (git init).
func (r *Repo) Init() error {
	_, err := r.run("init")
	return err
}

// NameFromRemote derives a folder name from a git remote URL.
func NameFromRemote(remote string) string {
	return extractRepoName(remote)
}

// Clone clones url into dest (dest must not already exist as a non-empty path).
func Clone(url, dest string) error {
	url = strings.TrimSpace(url)
	dest = strings.TrimSpace(dest)
	if url == "" {
		return fmt.Errorf("URL do repositório é obrigatória")
	}
	if dest == "" {
		return fmt.Errorf("destino do clone é obrigatório")
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	if parent := filepath.Dir(abs); parent != "" {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	if err := EnsureGitHubCredentials(); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", url, abs)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return wrapGitAuthError([]string{"clone", url, abs}, stderr.String(), err)
	}
	return nil
}
