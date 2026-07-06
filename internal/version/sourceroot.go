package version

import (
	"os"
	"path/filepath"
	"strings"
)

const moduleID = "github.com/laerciocrestani/gitia"

// SavedRepoRoot retorna o caminho do clone salvo em ~/.config/gitia/source.
func SavedRepoRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".config", "gitia", "source")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(data))
	if isGitiaRepo(root) {
		return root
	}
	return ""
}

func isGitiaRepo(dir string) bool {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false
	}
	modPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil || !strings.Contains(string(data), moduleID) {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "cmd", "gitia", "main.go"))
	return err == nil
}
