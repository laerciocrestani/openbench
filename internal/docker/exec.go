package docker

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// UpOptions configures docker compose up.
type UpOptions struct {
	ComposeFile   string
	Build         bool
	Profile       string
	Services      []string
	ForceRecreate bool
	NoDeps        bool
	DryRun        bool
	OnLine        func(string) // optional line callback (stdout/stderr)
}

// DownOptions configures docker compose down.
type DownOptions struct {
	ComposeFile string
	DryRun      bool
	OnLine      func(string)
}

// ServiceOptions configures docker compose service actions (stop/start).
type ServiceOptions struct {
	ComposeFile string
	Services    []string
	DryRun      bool
	OnLine      func(string)
}

// LogsOptions configures docker compose logs.
type LogsOptions struct {
	ComposeFile string
	Service     string
	Tail        int
	Follow      bool
}

// ExecOptions configures docker compose exec.
type ExecOptions struct {
	ComposeFile string
	Service     string
	Command     []string
	Interactive bool
}

// RunResult is the captured output of a compose mutation.
type RunResult struct {
	Output string
}

func upArgs(opts UpOptions) []string {
	args := []string{"up", "-d"}
	if opts.Build {
		args = append(args, "--build")
	}
	if opts.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	if opts.NoDeps {
		args = append(args, "--no-deps")
	}
	if opts.Profile != "" {
		args = append(args, "--profile", opts.Profile)
	}
	return append(args, opts.Services...)
}

func serviceArgs(subcommand string, services []string) []string {
	args := []string{subcommand}
	return append(args, services...)
}

func execArgs(service string, interactive bool, command []string) []string {
	args := []string{"exec"}
	if interactive {
		args = append(args, "-it")
	} else {
		// Disable TTY allocation for capture / non-interactive runs.
		args = append(args, "-T")
	}
	args = append(args, service)
	return append(args, command...)
}

func buildDockerComposeCmd(composeFile string, args ...string) *exec.Cmd {
	dir := composeDir(composeFile)
	full := append([]string{"compose", "-f", filepath.Base(composeFile)}, args...)
	cmd := exec.Command("docker", full...)
	cmd.Dir = dir
	return cmd
}

// BuildExecCommand builds docker compose exec for tea.ExecProcess.
func BuildExecCommand(composeFile, service string, interactive bool, command ...string) (*exec.Cmd, error) {
	if composeFile == "" {
		return nil, fmt.Errorf("compose file não encontrado")
	}
	if service == "" {
		return nil, fmt.Errorf("serviço não informado")
	}
	if len(command) == 0 {
		return nil, fmt.Errorf("comando não informado")
	}
	return buildDockerComposeCmd(composeFile, execArgs(service, interactive, command)...), nil
}

// BuildShellCommand builds an interactive shell exec command (sh).
func BuildShellCommand(composeFile, service string) (*exec.Cmd, error) {
	return BuildExecCommand(composeFile, service, true, "sh")
}

// runCompose runs a compose subcommand. When onLine is nil, output goes to
// the process stdout/stderr (CLI). When set, lines are captured and streamed.
func runCompose(composeFile string, args []string, onLine func(string)) (string, error) {
	cmd := buildDockerComposeCmd(composeFile, args...)
	if onLine == nil {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return "", cmd.Run()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var (
		mu  sync.Mutex
		buf strings.Builder
		wg  sync.WaitGroup
	)
	emit := func(line string) {
		mu.Lock()
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
		mu.Unlock()
		onLine(line)
	}
	scan := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			emit(sc.Text())
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	wg.Wait()
	runErr := cmd.Wait()

	mu.Lock()
	out := buf.String()
	mu.Unlock()
	return out, runErr
}

// Up runs docker compose up -d.
func Up(opts UpOptions) error {
	_, err := UpResult(opts)
	return err
}

// UpResult runs docker compose up -d and returns captured output when OnLine is set.
func UpResult(opts UpOptions) (RunResult, error) {
	if opts.ComposeFile == "" {
		return RunResult{}, fmt.Errorf("compose file não encontrado")
	}
	if opts.DryRun {
		return RunResult{}, nil
	}
	out, err := runCompose(opts.ComposeFile, upArgs(opts), opts.OnLine)
	return RunResult{Output: out}, err
}

// Down runs docker compose down.
func Down(opts DownOptions) error {
	_, err := DownResult(opts)
	return err
}

// DownResult runs docker compose down and returns captured output when OnLine is set.
func DownResult(opts DownOptions) (RunResult, error) {
	if opts.ComposeFile == "" {
		return RunResult{}, fmt.Errorf("compose file não encontrado")
	}
	if opts.DryRun {
		return RunResult{}, nil
	}
	out, err := runCompose(opts.ComposeFile, []string{"down"}, opts.OnLine)
	return RunResult{Output: out}, err
}

// Stop runs docker compose stop for one or more services.
func Stop(opts ServiceOptions) error {
	_, err := StopResult(opts)
	return err
}

// StopResult runs docker compose stop and returns captured output when OnLine is set.
func StopResult(opts ServiceOptions) (RunResult, error) {
	if opts.ComposeFile == "" {
		return RunResult{}, fmt.Errorf("compose file não encontrado")
	}
	if len(opts.Services) == 0 {
		return RunResult{}, fmt.Errorf("serviço não informado")
	}
	if opts.DryRun {
		return RunResult{}, nil
	}
	out, err := runCompose(opts.ComposeFile, serviceArgs("stop", opts.Services), opts.OnLine)
	return RunResult{Output: out}, err
}

// Start runs docker compose start for one or more services.
func Start(opts ServiceOptions) error {
	_, err := StartResult(opts)
	return err
}

// StartResult runs docker compose start and returns captured output when OnLine is set.
func StartResult(opts ServiceOptions) (RunResult, error) {
	if opts.ComposeFile == "" {
		return RunResult{}, fmt.Errorf("compose file não encontrado")
	}
	if len(opts.Services) == 0 {
		return RunResult{}, fmt.Errorf("serviço não informado")
	}
	if opts.DryRun {
		return RunResult{}, nil
	}
	out, err := runCompose(opts.ComposeFile, serviceArgs("start", opts.Services), opts.OnLine)
	return RunResult{Output: out}, err
}

// Recreate runs docker compose up -d --force-recreate --no-deps for a service.
func Recreate(composeFile, service string, dryRun bool) error {
	return Up(UpOptions{
		ComposeFile:   composeFile,
		Services:      []string{service},
		ForceRecreate: true,
		NoDeps:        true,
		DryRun:        dryRun,
	})
}

// RecreateResult force-recreates a service with optional line streaming.
func RecreateResult(composeFile, service string, dryRun bool, onLine func(string)) (RunResult, error) {
	return UpResult(UpOptions{
		ComposeFile:   composeFile,
		Services:      []string{service},
		ForceRecreate: true,
		NoDeps:        true,
		DryRun:        dryRun,
		OnLine:        onLine,
	})
}

// Logs runs docker compose logs.
func Logs(opts LogsOptions) error {
	if opts.ComposeFile == "" {
		return fmt.Errorf("compose file não encontrado")
	}
	dir := composeDir(opts.ComposeFile)
	tail := opts.Tail
	if tail <= 0 {
		tail = 100
	}
	args := []string{"compose", "-f", filepath.Base(opts.ComposeFile), "logs", "--tail", fmt.Sprintf("%d", tail)}
	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Service != "" {
		args = append(args, opts.Service)
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// LogsOutput captures docker compose logs without following.
func LogsOutput(opts LogsOptions) (string, error) {
	if opts.ComposeFile == "" {
		return "", fmt.Errorf("compose file não encontrado")
	}
	dir := composeDir(opts.ComposeFile)
	tail := opts.Tail
	if tail <= 0 {
		tail = 200
	}
	args := []string{"compose", "-f", filepath.Base(opts.ComposeFile), "logs", "--tail", fmt.Sprintf("%d", tail), "--no-color"}
	if opts.Service != "" {
		args = append(args, opts.Service)
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Exec runs a command inside a service container.
func Exec(opts ExecOptions) error {
	if opts.ComposeFile == "" {
		return fmt.Errorf("compose file não encontrado")
	}
	if opts.Service == "" {
		return fmt.Errorf("serviço não informado")
	}
	if len(opts.Command) == 0 {
		return fmt.Errorf("comando não informado")
	}
	cmd, err := BuildExecCommand(opts.ComposeFile, opts.Service, opts.Interactive, opts.Command...)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// ExecResult is the captured output of a non-interactive compose exec.
type ExecResult struct {
	Service  string
	Command  []string
	Output   string
	ExitCode int
}

// ExecOutput runs a non-interactive command and captures combined stdout/stderr.
func ExecOutput(opts ExecOptions) (*ExecResult, error) {
	if opts.ComposeFile == "" {
		return nil, fmt.Errorf("compose file não encontrado")
	}
	if opts.Service == "" {
		return nil, fmt.Errorf("serviço não informado")
	}
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("comando não informado")
	}
	cmd, err := BuildExecCommand(opts.ComposeFile, opts.Service, false, opts.Command...)
	if err != nil {
		return nil, err
	}
	out, runErr := cmd.CombinedOutput()
	res := &ExecResult{
		Service: opts.Service,
		Command: append([]string{}, opts.Command...),
		Output:  string(out),
	}
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, runErr
	}
	return res, nil
}

// Shell opens an interactive shell in the service container.
func Shell(composeFile, service string) error {
	if service == "" {
		return fmt.Errorf("serviço não informado")
	}
	err := Exec(ExecOptions{
		ComposeFile: composeFile,
		Service:     service,
		Command:     []string{"sh"},
		Interactive: true,
	})
	if err == nil {
		return nil
	}
	return Exec(ExecOptions{
		ComposeFile: composeFile,
		Service:     service,
		Command:     []string{"bash"},
		Interactive: true,
	})
}
