package version

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DefaultBase é a versão do primeiro commit (v0.1.0).
const DefaultBase = "0.1.0"

// BuildVersion is injected at link time via -ldflags, e.g.
//
//	-X github.com/laerciocrestani/openbench/internal/version.BuildVersion=0.2.1
//
// Prefer this over git Compute for packaged desktop builds (no .git nearby).
var BuildVersion string

type Info struct {
	Version string
	Commit  string
	Commits int
	Dirty   bool
}

func (i Info) Display() string {
	if i.Commit != "" {
		return i.Version + " · " + i.Commit
	}
	return i.Version
}

func (i Info) LDFlags() string {
	flags := fmt.Sprintf("-X github.com/laerciocrestani/openbench/internal/ui.buildVersion=%s", i.Version)
	flags += fmt.Sprintf(" -X github.com/laerciocrestani/openbench/internal/version.BuildVersion=%s", strings.TrimPrefix(i.Version, "v"))
	if i.Commit != "" {
		flags += fmt.Sprintf(" -X github.com/laerciocrestani/openbench/internal/ui.buildCommit=%s", i.Commit)
	}
	return flags
}

// Semver returns the running app version without a "v" prefix (for the updater).
func Semver() string {
	if v := strings.TrimSpace(BuildVersion); v != "" {
		return strings.TrimPrefix(v, "v")
	}
	info, err := Compute(".")
	if err != nil {
		return DefaultBase
	}
	return strings.TrimPrefix(info.Version, "v")
}

// DisplayCurrent returns a UI-facing version string (with optional commit).
func DisplayCurrent() string {
	if v := strings.TrimSpace(BuildVersion); v != "" {
		return "v" + strings.TrimPrefix(v, "v")
	}
	info, err := Compute(".")
	if err != nil {
		return "v" + DefaultBase
	}
	return info.Display()
}

// Compute calcula a versão a partir do número de commits (sem tags git).
// Primeiro commit = v0.1.0; cada commit adicional incrementa o patch.
func Compute(repoDir string) (Info, error) {
	commit, err := gitOutput(repoDir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return Info{}, err
	}

	dirty := gitDirty(repoDir)

	total, err := gitOutput(repoDir, "rev-list", "--count", "HEAD")
	if err != nil {
		return Info{}, err
	}
	count, _ := strconv.Atoi(strings.TrimSpace(total))
	if count < 1 {
		count = 1
	}

	ver, err := bumpPatch("v"+DefaultBase, count-1)
	if err != nil {
		return Info{}, err
	}

	return Info{
		Version: ver,
		Commit:  shortHash(commit),
		Commits: count,
		Dirty:   dirty,
	}, nil
}

func bumpPatch(base string, extra int) (string, error) {
	major, minor, patch, err := parseSemver(base)
	if err != nil {
		return "", err
	}
	patch += extra
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), nil
}

func parseSemver(tag string) (major, minor, patch int, err error) {
	tag = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	parts := strings.Split(tag, ".")
	if len(parts) < 3 {
		return 0, 0, 0, fmt.Errorf("semver inválida: %q", tag)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, err
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, err
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, err
	}
	return major, minor, patch, nil
}

func shortHash(rev string) string {
	rev = strings.TrimSpace(rev)
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitDirty(dir string) bool {
	cmd := exec.Command("git", "diff", "--quiet")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return true
	}
	cmd = exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	return cmd.Run() != nil
}
