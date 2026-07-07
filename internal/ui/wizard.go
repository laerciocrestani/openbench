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

// Wizard conduz um setup interativo com redraw completo — nada se perde na tela.
type Wizard struct {
	sess           *Session
	title          string
	intro          string
	entries        []wizardEntry
	currentSection string
	interactive    bool
}

type wizardEntry struct {
	section string
	label   string
	value   string
}

type selectState struct {
	section string
	label   string
	options []string
	cursor  int
}

// NewWizard inicia um wizard com título e texto introdutório.
func NewWizard(sess *Session, title, intro string) *Wizard {
	return &Wizard{
		sess:        sess,
		title:       title,
		intro:       intro,
		interactive: sess.enabled && term.IsTerminal(int(os.Stdin.Fd())),
	}
}

// AddSection marca a próxima etapa como pertencente a uma seção (ex.: Preferências).
func (w *Wizard) AddSection(title string) {
	w.currentSection = title
}

// UndoLast remove a última entrada registrada (útil em fluxos com passo intermediário).
func (w *Wizard) UndoLast() {
	if len(w.entries) > 0 {
		w.entries = w.entries[:len(w.entries)-1]
	}
}

// Record registra uma escolha concluída e redesenha a tela.
func (w *Wizard) Record(label, value string) {
	w.entries = append(w.entries, wizardEntry{
		section: w.currentSection,
		label:   label,
		value:   value,
	})
	if w.interactive {
		w.redraw(nil, false)
	}
}

// SelectConfig configura um seletor com navegação por setas.
type SelectConfig struct {
	Label      string
	Options    []string
	Default    string
	AllowOther bool
}

// Select exibe opções; ao confirmar, grava em Record (exceto Outro).
func (w *Wizard) Select(reader *bufio.Reader, cfg SelectConfig) (string, error) {
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

	if !w.interactive {
		return w.selectFallback(reader, cfg, options, defaultIdx)
	}

	idx, err := w.selectInteractive(cfg.Label, options, defaultIdx)
	if err != nil {
		return "", err
	}

	selected := options[idx]
	if cfg.AllowOther && selected == otherOption {
		custom, err := w.Ask(reader, cfg.Label, "")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(custom) == "" {
			return "", fmt.Errorf("%s: informe um valor", cfg.Label)
		}
		value := strings.TrimSpace(custom)
		w.Record(cfg.Label, value)
		return value, nil
	}

	w.Record(cfg.Label, selected)
	return selected, nil
}

// Ask solicita texto livre mantendo o histórico visível.
func (w *Wizard) Ask(reader *bufio.Reader, label, hint string) (string, error) {
	if hint != "" {
		w.redrawWithInput(label, hint, "")
	} else {
		w.redrawWithInput(label, "", "")
	}
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// Finish redesenha o estado final sem prompt ativo.
func (w *Wizard) Finish() {
	if w.interactive {
		w.redraw(nil, false)
	}
}

func (w *Wizard) selectInteractive(label string, options []string, start int) (int, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer term.Restore(fd, oldState)

	cursor := start
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(options) {
		cursor = len(options) - 1
	}

	state := &selectState{
		section: w.currentSection,
		label:   label,
		options: options,
		cursor:  cursor,
	}

	w.redraw(state, true)
	defer fmt.Fprint(w.sess.out, "\033[?25h")

	for {
		key, err := readKey(os.Stdin)
		if err != nil {
			return 0, err
		}

		switch key {
		case keyEnter:
			return cursor, nil
		case keyCancel:
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
		state.cursor = cursor
		w.redraw(state, true)
	}
}

func (w *Wizard) selectFallback(reader *bufio.Reader, cfg SelectConfig, options []string, defaultIdx int) (string, error) {
	w.redraw(&selectState{
		section: w.currentSection,
		label:   cfg.Label,
		options: options,
		cursor:  defaultIdx,
	}, false)
	fmt.Fprintf(w.sess.out, "\n%s", w.sess.paint("Escolha [número ou Enter para padrão]: ", dim))
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	idx := defaultIdx
	if input != "" {
		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(options) {
			return "", fmt.Errorf("opção inválida: %q", input)
		}
		idx = choice - 1
	}

	selected := options[idx]
	if cfg.AllowOther && selected == otherOption {
		custom, err := w.Ask(reader, cfg.Label, "")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(custom) == "" {
			return "", fmt.Errorf("%s: informe um valor", cfg.Label)
		}
		value := strings.TrimSpace(custom)
		w.Record(cfg.Label, value)
		return value, nil
	}

	w.Record(cfg.Label, selected)
	return selected, nil
}

func (w *Wizard) redraw(active *selectState, raw bool) {
	if w.interactive {
		fmt.Fprint(w.sess.out, "\033[H\033[2J\033[?25l")
	}
	frame := w.buildFrame(active, "", "")
	if raw {
		frame = strings.ReplaceAll(frame, "\n", "\r\n")
	}
	fmt.Fprint(w.sess.out, frame)
}

func (w *Wizard) redrawWithInput(label, hint, prompt string) {
	if w.interactive {
		fmt.Fprint(w.sess.out, "\033[H\033[2J\033[?25l")
	}
	fmt.Fprint(w.sess.out, w.buildFrame(nil, label, hint))
	if prompt != "" {
		fmt.Fprint(w.sess.out, prompt)
	} else if label != "" {
		fmt.Fprint(w.sess.out, label+": ")
	}
}

func (w *Wizard) buildFrame(active *selectState, inputLabel, inputHint string) string {
	var b strings.Builder

	fmt.Fprint(&b, FormatDashboardHeader(nil, defaultHeaderWidth, w.sess.dryRun, w.sess.enabled))

	b.WriteString(w.sess.paint(w.title, bold+cyan))
	b.WriteString("\n")
	if w.intro != "" {
		b.WriteString(w.sess.paint("  • "+w.intro, dim))
		b.WriteString("\n")
	}

	printedSection := ""
	for _, e := range w.entries {
		if e.section != "" && e.section != printedSection {
			b.WriteString("\n")
			b.WriteString(w.sess.paint(e.section, bold+cyan))
			b.WriteString("\n")
			printedSection = e.section
		}
		b.WriteString(fmt.Sprintf("  %s %s\n",
			w.sess.paint("✓", green),
			w.sess.paint(e.label+": "+e.value, dim)))
	}

	if active != nil {
		if active.section != "" && active.section != printedSection {
			b.WriteString("\n")
			b.WriteString(w.sess.paint(active.section, bold+cyan))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(w.sess.paint(active.label, bold+cyan))
		b.WriteString("\n")
		b.WriteString(w.sess.paint("  ↑↓ navegar · Enter confirmar", dim))
		b.WriteString("\n")
		for i, opt := range active.options {
			if i == active.cursor {
				b.WriteString("  ")
				b.WriteString(w.sess.paint("▸ ", green))
				b.WriteString(w.sess.paint(opt, bold+cyan))
				b.WriteString("\n")
			} else {
				b.WriteString(w.sess.paint("    "+opt, dim))
				b.WriteString("\n")
			}
		}
	}

	if inputLabel != "" || inputHint != "" {
		if inputHint != "" {
			b.WriteString("\n")
			b.WriteString(w.sess.paint("  • "+inputHint, dim))
			b.WriteString("\n")
		}
		if inputLabel != "" {
			b.WriteString("\n")
			b.WriteString(inputLabel)
			b.WriteString(": ")
		}
	}

	return b.String()
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

func indexOf(options []string, value string) int {
	for i, opt := range options {
		if opt == value {
			return i
		}
	}
	return -1
}

// StdinReader retorna um reader para prompts de texto.
func StdinReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}
