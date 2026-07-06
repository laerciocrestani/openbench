package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const moduleID = "github.com/laerciocrestani/gitia"

func Install() error {
	if err := requireGo(); err != nil {
		return err
	}
	checkOptionalTools()

	root, err := FindRepoRoot()
	if err != nil {
		return err
	}

	info("Instalando gitia...")
	if err := goInstall(root); err != nil {
		return err
	}

	bin, err := GitiaBin()
	if err != nil {
		return fmt.Errorf("instalação falhou — binário não encontrado em %s", GoBinDir())
	}
	ok("gitia instalado em %s", bin)

	if err := EnsurePath(); err != nil {
		return err
	}

	fmt.Println()
	ok("Instalação concluída!")
	info("Próximo passo: gitia config")
	info("Teste: %s --help", bin)
	return nil
}

func Update() error {
	if err := requireGo(); err != nil {
		return err
	}

	root, err := FindRepoRoot()
	if err != nil {
		return fmt.Errorf("rode update dentro do clone do repositório gitia")
	}

	before, err := gitShortHash(root)
	if err != nil {
		return err
	}

	branch, err := gitOutput(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}

	info("Atualizando branch %q...", branch)
	if err := gitRun(root, "fetch", "origin", branch); err != nil {
		_ = gitRun(root, "fetch", "origin")
	}
	if err := gitRun(root, "pull", "--ff-only", "origin", branch); err != nil {
		if err := gitRun(root, "pull", "--ff-only"); err != nil {
			return err
		}
	}

	after, err := gitShortHash(root)
	if err != nil {
		return err
	}

	info("Reinstalando binário...")
	if err := goInstall(root); err != nil {
		return err
	}

	bin, err := GitiaBin()
	if err != nil {
		return fmt.Errorf("reinstalação falhou")
	}
	ok("gitia atualizado em %s", bin)

	if before == after {
		info("Já estava na versão mais recente (%s)", after)
	} else {
		ok("Atualizado: %s → %s", before, after)
		if line, err := gitOutput(root, "log", "-1", "--oneline"); err == nil {
			fmt.Println(line)
		}
	}

	fmt.Println()
	info("Teste: %s --help", bin)
	return nil
}

func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err == nil && strings.Contains(string(data), moduleID) {
			if _, err := os.Stat(filepath.Join(dir, "cmd", "gitia", "main.go")); err == nil {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("repositório gitia não encontrado — rode dentro do clone ou use: go run ./cmd/gitia install")
}

func GoBinDir() string {
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		return filepath.Join(strings.TrimSpace(string(out)), "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}

func GitiaBin() (string, error) {
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

func EnsurePath() error {
	goBin := GoBinDir()
	if pathContains(goBin) {
		ok("PATH já inclui %s", goBin)
		return nil
	}

	shellRC := shellRCFile()
	if shellRC != "" {
		if data, err := os.ReadFile(shellRC); err == nil && strings.Contains(string(data), goBin) {
			ok("Entrada PATH já existe em %s", shellRC)
			return nil
		}
	}

	if shellRC == "" {
		warn("Não foi possível detectar ~/.zshrc ou ~/.bashrc")
		info("Adicione manualmente ao PATH: export PATH=\"$PATH:%s\"", goBin)
		return nil
	}

	info("Adicionando %s ao PATH em %s", goBin, shellRC)
	f, err := os.OpenFile(shellRC, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	block := fmt.Sprintf("\n# gitia (Go bin)\nexport PATH=\"$PATH:%s\"\n", goBin)
	if _, err := f.WriteString(block); err != nil {
		return err
	}

	ok("PATH configurado. Rode: source %s", shellRC)
	warn("Ou abra um novo terminal antes de usar gitia")
	return nil
}

func goInstall(root string) error {
	cmd := exec.Command("go", "install", "./cmd/gitia")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func requireGo() error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go não encontrado — instale em https://go.dev/dl/")
	}
	if out, err := exec.Command("go", "env", "GOVERSION").Output(); err == nil {
		info("Go %s detectado", strings.TrimPrefix(strings.TrimSpace(string(out)), "go"))
	}
	return nil
}

func checkOptionalTools() {
	if _, err := exec.LookPath("git"); err != nil {
		warn("git não encontrado — necessário para usar o gitia")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		warn("gh não encontrado — necessário apenas para gitia pr (https://cli.github.com/)")
		return
	}
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		warn("gh não autenticado — rode: gh auth login")
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
		return "gitia.exe"
	}
	return "gitia"
}

func info(format string, args ...any) {
	fmt.Printf("→ "+format+"\n", args...)
}

func ok(format string, args ...any) {
	fmt.Printf("✓ "+format+"\n", args...)
}

func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "! "+format+"\n", args...)
}
