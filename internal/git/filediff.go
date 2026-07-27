package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FileDiff is a before/after snapshot for one changed path (working tree vs HEAD).
type FileDiff struct {
	Path       string
	Status     string
	Before     string
	After      string
	Unified    string
	Binary     bool
	Insertions int
	Deletions  int
}

const maxDiffBytes = 2 << 20 // 2 MiB per side

// DiffFile builds a side-by-side view for path relative to the repo root.
// Before = last committed content (HEAD); After = current working tree (or empty if deleted).
func (r *Repo) DiffFile(path string) (*FileDiff, error) {
	rel, err := r.safeRelPath(path)
	if err != nil {
		return nil, err
	}

	status := "modified"
	var insertions, deletions int
	if changes, err := r.fileChanges(true); err == nil {
		for _, c := range changes {
			if c.Path == rel {
				status = c.Status
				insertions = c.Insertions
				deletions = c.Deletions
				break
			}
		}
	}

	before, beforeOK := r.blobAtHEAD(rel)
	after, afterOK, afterErr := r.worktreeFile(rel)
	if afterErr != nil {
		return nil, afterErr
	}

	out := &FileDiff{
		Path:       rel,
		Status:     status,
		Insertions: insertions,
		Deletions:  deletions,
	}

	if (beforeOK && isBinaryContent(before)) || (afterOK && isBinaryContent(after)) {
		out.Binary = true
		out.Before = ""
		out.After = ""
		return out, nil
	}

	if beforeOK {
		out.Before = truncateDiffText(before)
	}
	if afterOK {
		out.After = truncateDiffText(after)
	}

	unified, err := r.unifiedDiffFor(rel, status)
	if err == nil {
		out.Unified = unified
	}
	return out, nil
}

func (r *Repo) safeRelPath(path string) (string, error) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" || path == "." || strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("caminho inválido")
	}
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("caminho inválido")
	}
	abs := filepath.Join(r.dir, filepath.FromSlash(path))
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(r.dir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("caminho fora do repositório")
	}
	return filepath.ToSlash(rel), nil
}

func (r *Repo) blobAtHEAD(rel string) (string, bool) {
	out, err := r.runRaw("show", "HEAD:"+rel)
	if err != nil {
		return "", false
	}
	return out, true
}

func (r *Repo) worktreeFile(rel string) (content string, ok bool, err error) {
	abs := filepath.Join(r.dir, filepath.FromSlash(rel))
	data, readErr := os.ReadFile(abs)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", false, nil
		}
		return "", false, readErr
	}
	return string(data), true, nil
}

func (r *Repo) unifiedDiffFor(rel, status string) (string, error) {
	switch status {
	case "untracked":
		// Show entire file as additions when possible.
		data, ok, err := r.worktreeFile(rel)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", nil
		}
		return formatAsAddedDiff(rel, data), nil
	case "new", "staged":
		if out, err := r.run("diff", "--cached", "--", rel); err == nil && strings.TrimSpace(out) != "" {
			return out, nil
		}
	case "deleted":
		if out, err := r.run("diff", "--", rel); err == nil && strings.TrimSpace(out) != "" {
			return out, nil
		}
		if out, err := r.run("diff", "--cached", "--", rel); err == nil {
			return out, nil
		}
	}

	unstaged, err := r.run("diff", "--", rel)
	if err != nil {
		return "", err
	}
	staged, err := r.run("diff", "--cached", "--", rel)
	if err != nil {
		return unstaged, nil
	}
	parts := make([]string, 0, 2)
	if strings.TrimSpace(staged) != "" {
		parts = append(parts, staged)
	}
	if strings.TrimSpace(unstaged) != "" {
		parts = append(parts, unstaged)
	}
	return strings.Join(parts, "\n"), nil
}

func formatAsAddedDiff(path, content string) string {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- /dev/null\n+++ b/%s\n", path))
	for _, line := range lines {
		b.WriteByte('+')
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func isBinaryContent(s string) bool {
	if strings.ContainsRune(s, 0) {
		return true
	}
	if !utf8.ValidString(s) {
		return true
	}
	return false
}

func truncateDiffText(s string) string {
	if len(s) <= maxDiffBytes {
		return s
	}
	return s[:maxDiffBytes] + "\n\n… (arquivo truncado)"
}
