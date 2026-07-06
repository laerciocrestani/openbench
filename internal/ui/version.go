package ui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/laerciocrestani/gitia/internal/version"
)

// Injetados via -ldflags no go install.
var (
	buildVersion string
	buildCommit  string
)

var (
	runtimeOnce sync.Once
	runtimeInfo version.Info
	runtimeOK   bool
)

func Version() string {
	if info, ok := resolveRuntime(); ok {
		return info.Display()
	}
	if v := strings.TrimSpace(buildVersion); v != "" {
		if c := strings.TrimSpace(buildCommit); c != "" {
			return v + " · " + shortHash(c)
		}
		return v
	}
	return "v" + version.DefaultBase
}

func VersionInfo() version.Info {
	if info, ok := resolveRuntime(); ok {
		return info
	}
	if v := strings.TrimSpace(buildVersion); v != "" {
		return version.Info{
			Version: v,
			Commit:  shortHash(buildCommit),
		}
	}
	return version.Info{Version: "v" + version.DefaultBase}
}

func SetBuildVersion(v string) {
	buildVersion = v
}

func SetBuildCommit(c string) {
	buildCommit = c
}

func resolveRuntime() (version.Info, bool) {
	runtimeOnce.Do(func() {
		for _, root := range versionRoots() {
			if info, err := version.Compute(root); err == nil {
				runtimeInfo = info
				runtimeOK = true
				return
			}
		}
	})
	return runtimeInfo, runtimeOK
}

func versionRoots() []string {
	var roots []string
	if saved := version.SavedRepoRoot(); saved != "" {
		roots = append(roots, saved)
	}
	if cwd := findGitiaRootFromCwd(); cwd != "" {
		roots = append(roots, cwd)
	}
	return roots
}

func findGitiaRootFromCwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err == nil && strings.Contains(string(data), "github.com/laerciocrestani/gitia") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func shortHash(rev string) string {
	rev = strings.TrimSpace(rev)
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}
