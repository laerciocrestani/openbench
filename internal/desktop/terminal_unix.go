//go:build unix

package desktop

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	dockerpkg "github.com/laerciocrestani/openbench/internal/docker"
)

// TerminalSession manages an interactive PTY (host shell or docker exec).
type TerminalSession struct {
	mu     sync.Mutex
	ptmx   *os.File
	cmd    *exec.Cmd
	cwd    string
	label  string
	closed bool
	emit   func(event string, data string)
}

// NewTerminalSession starts $SHELL (or zsh/bash) as an interactive login shell.
func NewTerminalSession(cwd string, cols, rows uint16, emit func(event string, data string)) (*TerminalSession, error) {
	shell, args := resolveShell()
	cmd := exec.Command(shell, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)
	return startTerminalSession(cwd, "host", cmd, cols, rows, emit)
}

// NewDockerShellSession starts an interactive docker compose exec in a PTY.
// command is the argv after the service name (default: sh, with bash fallback on start failure handled by caller).
func NewDockerShellSession(cwd, composeFile, service string, command []string, cols, rows uint16, emit func(event string, data string)) (*TerminalSession, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		return nil, fmt.Errorf("serviço não informado")
	}
	if len(command) == 0 {
		command = []string{"sh"}
	}
	cmd, err := dockerpkg.BuildExecCommand(composeFile, service, true, command...)
	if err != nil {
		return nil, err
	}
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)
	label := "docker:" + service
	if len(command) > 0 && command[0] != "sh" && command[0] != "bash" {
		label = "docker:" + service + ":" + strings.Join(command, " ")
	}
	return startTerminalSession(cwd, label, cmd, cols, rows, emit)
}

func startTerminalSession(cwd, label string, cmd *exec.Cmd, cols, rows uint16, emit func(event string, data string)) (*TerminalSession, error) {
	if cwd == "" {
		return nil, fmt.Errorf("cwd vazio — abra um projeto")
	}
	info, err := os.Stat(cwd)
	if err != nil {
		return nil, fmt.Errorf("cwd inválido: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("cwd não é diretório: %s", cwd)
	}

	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	if cmd.Dir == "" {
		cmd.Dir = cwd
	}

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return nil, fmt.Errorf("iniciar PTY: %w", err)
	}

	s := &TerminalSession{
		ptmx:  ptmx,
		cmd:   cmd,
		cwd:   cwd,
		label: label,
		emit:  emit,
	}
	go s.readLoop()
	go s.waitLoop()
	return s, nil
}

// resolveShell picks the user login shell (macOS: usually /bin/zsh) and
// runs it as an interactive login shell, matching Terminal.app behaviour.
func resolveShell() (string, []string) {
	if sh := strings.TrimSpace(os.Getenv("SHELL")); sh != "" {
		if st, err := os.Stat(sh); err == nil && !st.IsDir() {
			return sh, []string{"-il"}
		}
	}
	candidates := []string{
		"/bin/zsh",
		"/bin/bash",
		"/usr/bin/bash",
		"/usr/local/bin/bash",
		"/opt/homebrew/bin/bash",
		"/bin/sh",
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			base := filepath.Base(c)
			if base == "sh" {
				return c, []string{"-i"}
			}
			return c, []string{"-il"}
		}
	}
	return "/bin/sh", []string{"-i"}
}

const terminalFlushInterval = 33 * time.Millisecond

func (s *TerminalSession) readLoop() {
	buf := make([]byte, 32*1024)
	var pending []byte
	// Coalesce PTY output with a one-shot timer so idle sessions do not wake
	// the CPU at ~60 Hz (a permanent ticker drained battery on macOS).
	var flushTimer *time.Timer
	flushCh := make(chan struct{}, 1)
	armFlush := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if flushTimer != nil {
			return
		}
		flushTimer = time.AfterFunc(terminalFlushInterval, func() {
			select {
			case flushCh <- struct{}{}:
			default:
			}
		})
	}
	emitPending := func() {
		s.mu.Lock()
		chunk := pending
		pending = nil
		closed := s.closed
		if flushTimer != nil {
			flushTimer.Stop()
			flushTimer = nil
		}
		s.mu.Unlock()
		if len(chunk) > 0 && !closed && s.emit != nil {
			s.emit("terminal:data", base64.StdEncoding.EncodeToString(chunk))
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			n, err := s.ptmx.Read(buf)
			if n > 0 {
				s.mu.Lock()
				pending = append(pending, buf[:n]...)
				needArm := flushTimer == nil
				s.mu.Unlock()
				if needArm {
					armFlush()
				}
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			emitPending()
			return
		case <-flushCh:
			emitPending()
		}
	}
}

func (s *TerminalSession) waitLoop() {
	err := s.cmd.Wait()
	s.mu.Lock()
	already := s.closed
	s.closed = true
	s.mu.Unlock()
	if already {
		return
	}
	msg := "exit"
	if err != nil {
		msg = err.Error()
	}
	if s.emit != nil {
		s.emit("terminal:exit", msg)
	}
	_ = s.ptmx.Close()
}

// Write sends input bytes to the PTY.
func (s *TerminalSession) Write(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil {
		return fmt.Errorf("terminal fechado")
	}
	_, err := io.WriteString(s.ptmx, data)
	return err
}

// Resize updates the PTY window size.
func (s *TerminalSession) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil {
		return fmt.Errorf("terminal fechado")
	}
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	return pty.Setsize(s.ptmx, &pty.Winsize{Cols: cols, Rows: rows})
}

// Cwd returns the session working directory.
func (s *TerminalSession) Cwd() string {
	return s.cwd
}

// Label returns a short session descriptor (host / docker:svc).
func (s *TerminalSession) Label() string {
	if s == nil {
		return ""
	}
	return s.label
}

// Close terminates the shell and releases the PTY.
func (s *TerminalSession) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	ptmx := s.ptmx
	cmd := s.cmd
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if ptmx != nil {
		_ = ptmx.Close()
	}
}
