package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/laerciocrestani/gitai/internal/uiprefs"
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
	if os.Getenv("CI") != "" {
		return false
	}
	if !uiprefs.ColorsEnabled() {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func (s *Session) Header() {
	writeBanner(s.out, s.dryRun, nil, s.paint)
}

func (s *Session) HeaderWithContext(ctx BannerContext) {
	writeBanner(s.out, s.dryRun, &ctx, s.paint)
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
	fmt.Fprintf(s.out, "\n%s %s\n\n", s.paint("✓", green), s.paint(message, green))
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

func (s *Session) CommandHintWithNote(cmd, note string) {
	fmt.Fprintf(s.out, "  %s %s %s\n",
		s.paint("→", cyan),
		s.paint(cmd, bold+magenta),
		s.paint(note, dim),
	)
}

func (s *Session) CommandHintMuted(cmd string) {
	fmt.Fprintf(s.out, "  %s %s\n", s.paint("→", dim), s.paint(cmd, dim))
}

func (s *Session) CommandHintMutedWithNote(cmd, note string) {
	fmt.Fprintf(s.out, "  %s %s %s\n",
		s.paint("→", dim),
		s.paint(cmd, dim),
		s.paint(note, dim),
	)
}

// Choose exibe opções numeradas e retorna o índice escolhido.
func (s *Session) Choose(label string, options []string, recommended int) (int, error) {
	if len(options) == 0 {
		return 0, fmt.Errorf("nenhuma opção disponível")
	}
	if recommended < 0 || recommended >= len(options) {
		recommended = 0
	}

	fmt.Fprintln(s.out)
	fmt.Fprintf(s.out, "%s\n", s.paint(label, bold+cyan))
	for i, opt := range options {
		marker := " "
		line := opt
		if i == recommended {
			marker = "●"
			line += s.paint(" (recomendado)", green)
		}
		fmt.Fprintf(s.out, "  (%d) %s %s\n", i+1, marker, line)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(s.out, "%s", s.paint("Escolha [número ou Enter para padrão]: ", dim))
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		input = strings.TrimSpace(input)
		if input == "" {
			return recommended, nil
		}

		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(options) {
			fmt.Fprintln(s.out, s.paint("  Opção inválida — informe um número da lista.", yellow))
			continue
		}
		return choice - 1, nil
	}
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
	cleared := make(chan struct{})
	var once sync.Once
	stop := func() {
		once.Do(func() {
			close(done)
			<-cleared
		})
	}

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		i := 0
		ticker := time.NewTicker(90 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				fmt.Fprint(s.out, "\r\033[K")
				close(cleared)
				return
			case <-ticker.C:
				frame := s.paint(frames[i%len(frames)], cyan)
				fmt.Fprintf(s.out, "\r  %s %s", frame, s.paint(label+"...", yellow))
				i++
			}
		}
	}()

	return stop
}

func (s *Session) doneLine(label string) {
	fmt.Fprintf(s.out, "  %s %s\n", s.paint("✓", green), s.paint(label, dim))
}

func (s *Session) failLine(label string) {
	fmt.Fprintf(s.out, "  %s %s\n", s.paint("✗", red), s.paint(label, red))
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
