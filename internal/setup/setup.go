package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/laerciocrestani/openbench/internal/ui"
	"github.com/laerciocrestani/openbench/internal/version"
)

const moduleID = "github.com/laerciocrestani/openbench"

func Install() error {
	sess := ui.New("install", false)
	sess.Header()

	if err := sess.Step("Checking Go toolchain", func() error {
		return requireGo(sess)
	}); err != nil {
		return err
	}

	sess.Step("Checking optional tools", func() error {
		checkOptionalTools(sess)
		return nil
	})

	var root string
	if err := sess.Step("Locating repository", func() error {
		var err error
		root, err = FindRepoRoot()
		if err != nil {
			return err
		}
		_ = saveSourceRoot(root)
		return nil
	}); err != nil {
		return err
	}

	if err := sess.Step("Building binary", func() error {
		return goInstall(root)
	}); err != nil {
		return err
	}

	var bin string
	if err := sess.Step("Verifying installation", func() error {
		var err error
		bin, err = ObBin()
		if err != nil {
			return fmt.Errorf("instalação falhou — binário não encontrado em %s", GoBinDir())
		}
		sess.Detail(bin)
		if ob, err := exec.LookPath(obBinaryName()); err == nil {
			sess.Detail(ob)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := sess.Step("Configuring PATH", func() error {
		if err := ensurePath(sess); err != nil {
			return err
		}
		applySessionPath(sess)
		return nil
	}); err != nil {
		return err
	}

	if err := sess.Step("Configuring ob alias", func() error {
		if err := ensureObAlias(sess, bin); err != nil {
			return err
		}
		applySessionPath(sess)
		return nil
	}); err != nil {
		return err
	}

	sess.Detail("Próximo passo: ob config")
	sess.Success("Installation complete 🚀")
	return nil
}

func Update() error {
	sess := ui.New("update", false)
	sess.Header()

	if err := sess.Step("Checking Go toolchain", func() error {
		return requireGo(sess)
	}); err != nil {
		return err
	}

	root, err := FindRepoRoot()
	if err != nil {
		sess.Info("Clone local não encontrado — buscando última versão no GitHub")
		return updateFromRemote(sess)
	}

	_ = saveSourceRoot(root)

	if err := ensureFullClone(sess, root); err != nil {
		return err
	}

	before, err := gitShortHash(root)
	if err != nil {
		return err
	}

	branch, err := gitOutput(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}

	if err := sess.Step("Fetching updates", func() error {
		if err := gitRun(root, "fetch", "origin", branch); err != nil {
			_ = gitRun(root, "fetch", "origin")
		}
		return nil
	}); err != nil {
		return err
	}

	if err := sess.Step("Pulling changes", func() error {
		if err := gitRun(root, "pull", "--ff-only", "origin", branch); err != nil {
			return gitRun(root, "pull", "--ff-only")
		}
		return nil
	}); err != nil {
		return err
	}

	after, err := gitShortHash(root)
	if err != nil {
		return err
	}

	if err := sess.Step("Rebuilding binary", func() error {
		return goInstall(root)
	}); err != nil {
		return err
	}

	bin, err := ObBin()
	if err != nil {
		return fmt.Errorf("reinstalação falhou")
	}

	if before == after {
		sess.Info(fmt.Sprintf("Already on latest commit (%s)", after))
	} else {
		sess.Detail(fmt.Sprintf("%s → %s", before, after))
		if line, err := gitOutput(root, "log", "-1", "--oneline"); err == nil {
			sess.Detail(line)
		}
	}
	showInstalledVersion(sess, root, bin)
	sess.Success("Update complete 🚀")
	return nil
}

func updateFromRemote(sess *ui.Session) error {
	root := readSavedSourceRoot()
	if root == "" {
		root = defaultSourceCloneDir()
	}

	if !isValidRepoRoot(root) {
		if err := sess.Step("Cloning repository", func() error {
			if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
				return err
			}
			_ = os.RemoveAll(root)
			cmd := exec.Command("git", "clone", defaultRepoURL, root)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("clone falhou: %w", err)
			}
			return nil
		}); err != nil {
			return err
		}
		_ = saveSourceRoot(root)
	} else {
		_ = saveSourceRoot(root)
		if err := ensureFullClone(sess, root); err != nil {
			return err
		}
		branch, err := gitOutput(root, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}
		if err := sess.Step("Fetching updates", func() error {
			if err := gitRun(root, "fetch", "origin", branch); err != nil {
				_ = gitRun(root, "fetch", "origin")
			}
			return nil
		}); err != nil {
			return err
		}
		if err := sess.Step("Pulling changes", func() error {
			if err := gitRun(root, "pull", "--ff-only", "origin", branch); err != nil {
				return gitRun(root, "pull", "--ff-only")
			}
			return nil
		}); err != nil {
			return err
		}
	}

	if err := sess.Step("Rebuilding binary", func() error {
		return goInstall(root)
	}); err != nil {
		return err
	}

	bin, err := ObBin()
	if err != nil {
		return fmt.Errorf("reinstalação falhou")
	}

	showInstalledVersion(sess, root, bin)
	sess.Success("Update complete 🚀")
	return nil
}

func defaultSourceCloneDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "openbench", "repository")
}

func ensureFullClone(sess *ui.Session, root string) error {
	if _, err := os.Stat(filepath.Join(root, ".git", "shallow")); err != nil {
		return nil
	}
	return sess.Step("Fetching full history", func() error {
		if err := gitRun(root, "fetch", "--unshallow", "origin"); err != nil {
			return gitRun(root, "fetch", "--unshallow")
		}
		return nil
	})
}

func showInstalledVersion(sess *ui.Session, root, bin string) {
	if ver, err := version.Compute(root); err == nil {
		sess.Detail(fmt.Sprintf("Installed: %s", ver.Display()))
	}
	sess.Detail(bin)
}

func FindRepoRoot() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		if root := findRepoFromDir(cwd); root != "" {
			return root, nil
		}
	}

	if env := strings.TrimSpace(os.Getenv("OPENBENCH_ROOT")); env != "" {
		if isValidRepoRoot(env) {
			return filepath.Clean(env), nil
		}
	}

	if root := readSavedSourceRoot(); root != "" {
		return root, nil
	}

	if root := findRepoFromExecutable(); root != "" {
		return root, nil
	}

	return "", fmt.Errorf("repositório openbench não encontrado")
}

func GoBinDir() string {
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		return filepath.Join(strings.TrimSpace(string(out)), "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}

func ObBin() (string, error) {
	candidate := filepath.Join(GoBinDir(), binaryName())
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		if runtime.GOOS != "windows" {
			if st.Mode()&0111 != 0 {
				return candidate, nil
			}
		} else {
			return candidate, nil
		}
	}

	if path, err := exec.LookPath(binaryName()); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("binário não encontrado")
}

func ensurePath(sess *ui.Session) error {
	goBin := GoBinDir()
	if pathContains(goBin) {
		sess.Detail("PATH already includes " + goBin)
		return nil
	}

	shellRC := shellRCFile()
	if shellRC != "" {
		if data, err := os.ReadFile(shellRC); err == nil && strings.Contains(string(data), goBin) {
			sess.Detail("PATH entry already exists in " + shellRC)
			return nil
		}
	}

	if shellRC == "" {
		sess.Warn("Could not detect ~/.zshrc or ~/.bashrc")
		sess.Detail(`Add manually: export PATH="$PATH:` + goBin + `"`)
		return nil
	}

	f, err := os.OpenFile(shellRC, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	block := fmt.Sprintf("\n# openbench (Go bin)\nexport PATH=\"$PATH:%s\"\n", goBin)
	if _, err := f.WriteString(block); err != nil {
		return err
	}

	sess.Detail("Added to " + shellRC)
	applySessionPath(sess)
	return nil
}

func applySessionPath(sess *ui.Session) {
	goBin := GoBinDir()
	pathEnv := os.Getenv("PATH")
	if pathContainsIn(pathEnv, goBin) {
		return
	}
	_ = os.Setenv("PATH", goBin+string(os.PathListSeparator)+pathEnv)
	sess.Detail("Session PATH updated")
}

func pathContainsIn(pathEnv, dir string) bool {
	for _, part := range filepath.SplitList(pathEnv) {
		if part == dir {
			return true
		}
	}
	return false
}

func goInstall(root string) error {
	info, err := version.Compute(root)
	if err != nil {
		return err
	}

	binDir := GoBinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	out := filepath.Join(binDir, binaryName())
	args := []string{"build", "-ldflags", info.LDFlags(), "-o", out, "./cmd/ob"}

	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return linkObBinary(out)
}

func linkObBinary(openbenchPath string) error {
	obName := obBinaryName()
	if obName == binaryName() {
		return nil
	}

	obPath := filepath.Join(filepath.Dir(openbenchPath), obName)
	_ = os.Remove(obPath)

	if err := os.Symlink(filepath.Base(openbenchPath), obPath); err == nil {
		return nil
	}

	data, err := os.ReadFile(openbenchPath)
	if err != nil {
		return fmt.Errorf("criar atalho ob: %w", err)
	}
	return os.WriteFile(obPath, data, 0o755)
}

func ensureObAlias(sess *ui.Session, bin string) error {
	if runtime.GOOS == "windows" {
		sess.Detail("Alias ob não necessário no Windows (use o binário ob.exe)")
		return nil
	}

	shellRC := shellRCFile()
	if shellRC == "" {
		sess.Warn("Could not detect ~/.zshrc or ~/.bashrc")
		sess.Detail("ob já está em " + filepath.Join(GoBinDir(), obBinaryName()))
		return nil
	}

	data, err := os.ReadFile(shellRC)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	const marker = "# openbench alias (ob)"
	if strings.Contains(string(data), marker) {
		sess.Detail("Alias ob already configured in " + shellRC)
		return nil
	}

	f, err := os.OpenFile(shellRC, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	block := fmt.Sprintf("\n%s\nalias ob='%s'\n", marker, bin)
	if _, err := f.WriteString(block); err != nil {
		return err
	}

	sess.Detail("Alias ob added to " + shellRC)
	return nil
}

func requireGo(sess *ui.Session) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go não encontrado — instale em https://go.dev/dl/")
	}
	if out, err := exec.Command("go", "env", "GOVERSION").Output(); err == nil {
		sess.Detail(strings.TrimPrefix(strings.TrimSpace(string(out)), "go"))
	}
	return nil
}

func checkOptionalTools(sess *ui.Session) {
	if _, err := exec.LookPath("git"); err != nil {
		sess.Warn("git not found — required for ob")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		sess.Warn("gh not found — required only for ob pr")
		return
	}
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		sess.Warn("gh not authenticated — run: gh auth login")
	}
}

func gitShortHash(root string) (string, error) {
	return gitOutput(root, "rev-parse", "--short", "HEAD")
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitRun(root string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellRCFile() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if shell := os.Getenv("SHELL"); strings.Contains(shell, "zsh") {
		return filepath.Join(home, ".zshrc")
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); err == nil {
		return filepath.Join(home, ".zshrc")
	}
	return filepath.Join(home, ".bashrc")
}

func pathContains(dir string) bool {
	for _, part := range filepath.SplitList(os.Getenv("PATH")) {
		if part == dir {
			return true
		}
	}
	return false
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "openbench.exe"
	}
	return "openbench"
}

func obBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ob.exe"
	}
	return "ob"
}

