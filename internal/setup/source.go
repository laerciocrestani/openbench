package setup

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultRepoURL = "https://github.com/laerciocrestani/gitia.git"

func sourceRootFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gitia", "source"), nil
}

func readSavedSourceRoot() string {
	path, err := sourceRootFile()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(data))
	if isValidRepoRoot(root) {
		return root
	}
	return ""
}

func saveSourceRoot(root string) error {
	root = filepath.Clean(root)
	if !isValidRepoRoot(root) {
		return nil
	}
	path, err := sourceRootFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(root+"\n"), 0o644)
}

func isValidRepoRoot(dir string) bool {
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

func findRepoFromDir(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}

	for {
		if isValidRepoRoot(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func findRepoFromExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	return findRepoFromDir(filepath.Dir(exe))
}
