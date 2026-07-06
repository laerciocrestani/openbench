package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const dividerWidth = 42

type Session struct {
	command string
	dryRun  bool
	enabled bool
	out     io.Writer
}

func New(command string, dryRun bool) *Session {
	return &Session{
		command: command,
		dryRun:  dryRun,
		enabled: colorsEnabled(),
		out:     os.Stdout,
	}
}

func colorsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("GITAI_NO_UI") != "" {
		return false
	}
	if os.Getenv("CI") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func (s *Session) Header() {
	title := s.paint("GitAi", bold+cyan)
	ver := s.paint(Version(), dim)
	sep := s.paint("|", dim)
	fmt.Fprintf(s.out, "🤖 %s %s %s\n\n", title, sep, ver)
}

func (s *Session) Divider() {
	line := strings.Repeat("─", dividerWidth)
	fmt.Fprintln(s.out, s.paint(line, dim))
}

func (s *Session) MetaRow(label, value string) {
	padded := fmt.Sprintf("%-12s", label)
	fmt.Fprintf(s.out, "%s  %s\n", s.paint(padded, dim), value)
}

func (s *Session) StatusValue(dirty bool, staged, modified, untracked int) string {
	if !dirty {
		return s.paint("✓ clean", green)
	}
	return s.paint(fmt.Sprintf("%d staged · %d modified · %d untracked", staged, modified, untracked), yellow)
}

func (s *Session) Step(label string, fn func() error) error {
	if !s.enabled {
		return fn()
	}

	stop := s.spinner(label)
	err := fn()
	stop()

	if err != nil {
		s.failLine(label)
		return err
	}
	s.doneLine(label)
	return nil
}

func (s *Session) StepQuiet(fn func() error) error {
	return fn()
}

func (s *Session) Info(label string) {
	fmt.Fprintln(s.out, s.paint("  • "+label, dim))
}

func (s *Session) Success(message string) {
	fmt.Fprintf(s.out, "\n%s %s\n", s.paint("✓", green), s.paint(message, green))
}

func (s *Session) Detail(message string) {
	fmt.Fprintln(s.out, s.paint("  "+message, dim))
}

func (s *Session) Warn(message string) {
	fmt.Fprintln(os.Stderr, s.paint("! "+message, yellow))
}

func (s *Session) Prompt(label string) {
	fmt.Fprint(os.Stderr, s.paint("? "+label, magenta))
}

// Input exibe um prompt de texto simples, sem "?".
func (s *Session) Input(label string) {
	fmt.Fprint(s.out, label)
}

func (s *Session) UsageBlock(lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintln(s.out)
	fmt.Fprintf(s.out, "%s\n", s.paint("Uso de IA", bold+cyan))
	for _, line := range lines {
		s.Bullet(line)
	}
}

func (s *Session) Section(title string) {
	fmt.Fprintf(s.out, "\n%s\n", s.paint(title, bold+cyan))
}

// SectionFirst é a primeira seção após o header — sem linha em branco antes.
func (s *Session) SectionFirst(title string) {
	fmt.Fprintf(s.out, "%s\n", s.paint(title, bold+cyan))
}

// Footer adiciona uma linha em branco antes do prompt do shell.
func (s *Session) Footer() {
	fmt.Fprintln(s.out)
}

func (s *Session) KV(key, value string) {
	fmt.Fprintf(s.out, "  %s %s\n", s.paint(key+":", dim), value)
}

func (s *Session) Bullet(text string) {
	fmt.Fprintf(s.out, "  %s %s\n", s.paint("•", dim), text)
}

func (s *Session) BranchLine(name string, current bool, upstream string, ahead, behind int) {
	marker := " "
	if current {
		marker = s.paint("*", green)
	}

	line := fmt.Sprintf("%s %s", marker, name)
	if upstream != "" {
		line += s.paint(" → "+upstream, dim)
	}
	if ahead > 0 || behind > 0 {
		line += s.paint(fmt.Sprintf(" (↑%d ↓%d)", ahead, behind), yellow)
	}
	fmt.Fprintf(s.out, "  %s\n", line)
}

func (s *Session) CommandHint(cmd string) {
	fmt.Fprintf(s.out, "  %s %s\n", s.paint("→", cyan), s.paint(cmd, bold+magenta))
}

func (s *Session) FileChange(path, status, stats string) {
	tag := s.paint(status, fileStatusColor(status))
	line := fmt.Sprintf("  %s %s", tag, path)
	if stats != "" {
		line += " " + s.paint(stats, green)
	}
	fmt.Fprintln(s.out, line)
}

func fileStatusColor(status string) string {
	switch status {
	case "untracked":
		return yellow
	case "deleted":
		return red
	case "new", "staged":
		return green
	case "modified", "staged+modified":
		return magenta
	default:
		return cyan
	}
}

func (s *Session) spinner(label string) func() {
	if !s.enabled {
		return func() {}
	}

	done := make(chan struct{})
	var once sync.Once
	stop := func() {
		once.Do(func() { close(done) })
	}

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		i := 0
		ticker := time.NewTicker(90 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				fmt.Fprint(os.Stderr, "\r\033[K")
				return
			case <-ticker.C:
				frame := s.paint(frames[i%len(frames)], cyan)
				fmt.Fprintf(os.Stderr, "\r  %s %s", frame, s.paint(label+"...", yellow))
				i++
			}
		}
	}()

	return stop
}

func (s *Session) doneLine(label string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", s.paint("✓", green), s.paint(label, dim))
}

func (s *Session) failLine(label string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", s.paint("✗", red), s.paint(label, red))
}

func (s *Session) paint(text, code string) string {
	if !s.enabled {
		return text
	}
	return code + text + reset
}

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	cyan    = "\033[36m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	magenta = "\033[35m"
	red     = "\033[31m"
)
