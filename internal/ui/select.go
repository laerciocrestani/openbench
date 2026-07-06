package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const otherOption = "Outro"

// SelectConfig configura um seletor interativo com navegação por setas.
type SelectConfig struct {
	Label      string
	Options    []string
	Default    string
	AllowOther bool
}

// Select exibe opções com ( ) / (●). Em TTY usa ↑↓; fora disso, lista numerada.
func (s *Session) Select(reader *bufio.Reader, cfg SelectConfig) (string, error) {
	options := append([]string{}, cfg.Options...)
	if cfg.AllowOther {
		options = append(options, otherOption)
	}
	if len(options) == 0 {
		return "", fmt.Errorf("%s: nenhuma opção disponível", cfg.Label)
	}

	defaultIdx := indexOf(options, cfg.Default)
	if defaultIdx < 0 {
		defaultIdx = 0
	}

	interactive := s.enabled && term.IsTerminal(int(os.Stdin.Fd()))
	var (
		idx int
		err error
	)
	if interactive {
		idx, err = s.selectInteractive(cfg.Label, options, defaultIdx)
	} else {
		idx, err = s.selectFallback(reader, cfg.Label, options, defaultIdx)
	}
	if err != nil {
		return "", err
	}

	selected := options[idx]
	if cfg.AllowOther && selected == otherOption {
		custom, err := s.promptCustom(reader, cfg.Label)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(custom) == "" {
			return "", fmt.Errorf("%s: informe um valor", cfg.Label)
		}
		return strings.TrimSpace(custom), nil
	}
	return selected, nil
}

func (s *Session) selectInteractive(label string, options []string, start int) (int, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer term.Restore(fd, oldState)

	out := os.Stderr
	const eol = "\r\n"

	cursor := start
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(options) {
		cursor = len(options) - 1
	}

	lines := len(options) + 3

	render := func() {
		fmt.Fprint(out, "\033[?25l")
		var b strings.Builder
		b.WriteString(s.paint(label, bold+cyan))
		b.WriteString(eol)
		b.WriteString(s.paint("  ↑↓ navegar · Enter confirmar", dim))
		b.WriteString(eol)
		b.WriteString(eol)
		for i, opt := range options {
			marker := " "
			style := dim
			if i == cursor {
				marker = "●"
				style = green
			}
			line := fmt.Sprintf("  (%s) %s", marker, opt)
			b.WriteString(s.paint(line, style))
			b.WriteString(eol)
		}
		fmt.Fprint(out, b.String())
	}

	clear := func() {
		fmt.Fprintf(out, "\033[%dA\033[J", lines)
	}

	fmt.Fprint(out, eol)
	render()
	defer fmt.Fprint(out, "\033[?25h")

	for {
		key, err := readKey(os.Stdin)
		if err != nil {
			clear()
			return 0, err
		}

		switch key {
		case keyEnter:
			clear()
			fmt.Fprintf(out, "  %s %s%s%s", s.paint("✓", green), s.paint(label+": "+options[cursor], dim), eol, eol)
			return cursor, nil
		case keyCancel:
			clear()
			return 0, fmt.Errorf("cancelado")
		case keyUp:
			cursor--
		case keyDown:
			cursor++
		default:
			continue
		}

		if cursor < 0 {
			cursor = 0
		}
		if cursor >= len(options) {
			cursor = len(options) - 1
		}
		clear()
		render()
	}
}

type keyKind int

const (
	keyNone keyKind = iota
	keyUp
	keyDown
	keyEnter
	keyCancel
)

func readKey(r io.Reader) (keyKind, error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return keyNone, err
	}
	switch buf[0] {
	case '\r', '\n':
		return keyEnter, nil
	case 3:
		return keyCancel, nil
	case 'k':
		return keyUp, nil
	case 'j':
		return keyDown, nil
	case 27:
		seq := make([]byte, 2)
		if _, err := io.ReadFull(r, seq[:]); err != nil {
			return keyNone, nil
		}
		switch {
		case seq[0] == '[' && seq[1] == 'A':
			return keyUp, nil
		case seq[0] == '[' && seq[1] == 'B':
			return keyDown, nil
		case seq[0] == 'O' && seq[1] == 'A':
			return keyUp, nil
		case seq[0] == 'O' && seq[1] == 'B':
			return keyDown, nil
		}
	}
	return keyNone, nil
}

func (s *Session) selectFallback(reader *bufio.Reader, label string, options []string, defaultIdx int) (int, error) {
	fmt.Fprintf(s.out, "\n%s\n", s.paint(label, bold+cyan))
	for i, opt := range options {
		marker := " "
		if i == defaultIdx {
			marker = "●"
		}
		fmt.Fprintf(s.out, "  (%s) %d) %s\n", marker, i+1, opt)
	}
	fmt.Fprintf(s.out, "\n%s", s.paint("? Escolha [número ou Enter para padrão]: ", magenta))
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultIdx, nil
	}
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(options) {
		return 0, fmt.Errorf("opção inválida: %q", input)
	}
	return choice - 1, nil
}

func (s *Session) promptCustom(reader *bufio.Reader, label string) (string, error) {
	fmt.Fprintln(os.Stderr)
	s.Prompt(label + ": ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func indexOf(options []string, value string) int {
	for i, opt := range options {
		if opt == value {
			return i
		}
	}
	return -1
}

// StdinReader retorna um reader para prompts de texto após o seletor.
func StdinReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}
