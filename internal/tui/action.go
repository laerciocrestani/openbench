package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/config"
	prpkg "github.com/laerciocrestani/gitai/internal/pr"
)

type ActionKind int

const (
	ActionCommit ActionKind = iota
	ActionPush
	ActionPR
	ActionSync
	ActionOpenPR
)

type ActionPhase int

const (
	PhaseRunning ActionPhase = iota
	PhaseConfirm
	PhaseConfirming
	PhaseDone
	PhaseError
)

type actionState struct {
	kind     ActionKind
	phase    ActionPhase
	progress *ActionProgress
	preview  *app.Result
	result   *app.Result
	err      error
	draft    bool
	opts     app.Options
}

type actionPreviewMsg struct {
	kind    ActionKind
	preview *app.Result
	err     error
}

type actionConfirmMsg struct {
	kind   ActionKind
	result *app.Result
	err    error
}

type actionSimpleMsg struct {
	kind ActionKind
	err  error
}

func newActionState(kind ActionKind) *actionState {
	a := &actionState{kind: kind}
	return a.start()
}

func (a *actionState) start() *actionState {
	a.progress = NewActionProgress()
	a.progress.Reset()
	a.opts = app.Options{Progress: a.progress}
	a.phase = PhaseRunning
	a.preview = nil
	a.result = nil
	a.err = nil
	return a
}

func (a *actionState) previewCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		opts := app.Options{Progress: a.progress}

		switch a.kind {
		case ActionCommit:
			preview, err := app.PreviewCommit(ctx, opts)
			return actionPreviewMsg{kind: a.kind, preview: preview, err: err}
		case ActionPR:
			preview, err := app.PreviewPR(ctx, opts)
			return actionPreviewMsg{kind: a.kind, preview: preview, err: err}
		default:
			return actionPreviewMsg{kind: a.kind, err: fmt.Errorf("ação sem preview")}
		}
	}
}

func (a *actionState) confirmCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		opts := a.opts
		opts.Progress = a.progress

		switch a.kind {
		case ActionCommit:
			result, err := app.ConfirmCommit(ctx, a.preview, opts)
			return actionConfirmMsg{kind: ActionCommit, result: result, err: err}
		case ActionPR:
			result, err := app.ConfirmPR(ctx, a.preview, a.draft, opts)
			return actionConfirmMsg{kind: ActionPR, result: result, err: err}
		default:
			return actionConfirmMsg{err: fmt.Errorf("confirmação não suportada")}
		}
	}
}

func (a *actionState) directCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		opts := app.Options{Progress: a.progress}

		switch a.kind {
		case ActionPush:
			_, err := app.RunPush(ctx, opts)
			return actionSimpleMsg{kind: a.kind, err: err}
		case ActionSync:
			err := app.RunSync(app.SyncOptions{Progress: a.progress})
			return actionSimpleMsg{kind: a.kind, err: err}
		case ActionOpenPR:
			client, err := prpkg.New()
			if err != nil {
				return actionSimpleMsg{kind: a.kind, err: err}
			}
			_, err = client.OpenInBrowser()
			return actionSimpleMsg{kind: a.kind, err: err}
		default:
			return actionSimpleMsg{kind: a.kind, err: fmt.Errorf("ação inválida")}
		}
	}
}

func (a *actionState) handlePreview(msg actionPreviewMsg) {
	a.preview = msg.preview
	a.err = msg.err
	if msg.err != nil {
		a.phase = PhaseError
		return
	}
	a.phase = PhaseConfirm
}

func (a *actionState) handleConfirm(msg actionConfirmMsg) {
	a.result = msg.result
	a.err = msg.err
	if msg.err != nil {
		a.phase = PhaseError
		return
	}
	a.phase = PhaseDone
}

func (a *actionState) handleSimple(msg actionSimpleMsg) {
	a.err = msg.err
	if msg.err != nil {
		a.phase = PhaseError
		return
	}
	a.phase = PhaseDone
}

func (a *actionState) toggleDraft() {
	a.draft = !a.draft
	if a.preview == nil || a.preview.PRSuggestion == nil {
		return
	}
	base := "main"
	if cfg, err := config.Load(); err == nil && cfg.BaseBranch != "" {
		base = cfg.BaseBranch
	}
	client, err := prpkg.New()
	if err == nil {
		a.preview.PRPreview = client.PreviewCreate(a.preview.PRSuggestion, base, a.draft)
	}
}

func (a *actionState) title() string {
	switch a.kind {
	case ActionCommit:
		return "Commit com IA"
	case ActionPush:
		return "Push"
	case ActionPR:
		return "Pull Request"
	case ActionSync:
		return "Sync"
	case ActionOpenPR:
		return "Abrir PR"
	default:
		return "Ação"
	}
}

func (a *actionState) View(width int) string {
	if a == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(styleSection.Render(a.title()))
	b.WriteString("\n\n")

	switch a.phase {
	case PhaseRunning, PhaseConfirming:
		status, logs := a.progress.Snapshot()
		if status == "" {
			status = "Trabalhando…"
		}
		b.WriteString(styleHint.Render("  " + status))
		b.WriteString("\n")
		for _, line := range logs {
			b.WriteString(styleHint.Render("  " + line))
			b.WriteString("\n")
		}
		if a.phase == PhaseConfirming {
			b.WriteString("\n")
			b.WriteString(styleHint.Render("  Confirmando…"))
		}

	case PhaseConfirm:
		b.WriteString(renderPreview(a))
		b.WriteString("\n")
		b.WriteString(actionConfirmHelp(a))

	case PhaseDone:
		b.WriteString(styleCurrent.Render("  ✓ Concluído"))
		b.WriteString("\n")
		if a.result != nil {
			if a.result.Message != "" {
				b.WriteString("\n")
				b.WriteString(styleHint.Render(wrapPreview(a.result.Message, width-4)))
			}
			if a.result.PRURL != "" {
				b.WriteString("\n")
				b.WriteString(styleHint.Render(a.result.PRURL))
			}
		}
		b.WriteString("\n\n")
		b.WriteString(styleHint.Render("  Enter para voltar"))

	case PhaseError:
		if a.err != nil {
			b.WriteString(styleError.Render("  ✗ " + a.err.Error()))
		}
		b.WriteString("\n\n")
		b.WriteString(styleHint.Render("  Enter ou Esc para voltar"))
	}

	return b.String()
}

func renderPreview(a *actionState) string {
	if a.preview == nil {
		return ""
	}
	var b strings.Builder

	switch a.kind {
	case ActionCommit:
		b.WriteString(styleHint.Render("Preview do commit:\n\n"))
		b.WriteString(wrapPreview(a.preview.Message, 76))

	case ActionPR:
		if a.preview.PRSuggestion != nil {
			s := a.preview.PRSuggestion
			b.WriteString(styleTitle.Render(s.Title))
			b.WriteString("\n\n")
			if a.draft {
				b.WriteString(styleYellow.Render("  [draft]"))
				b.WriteString("\n")
			}
			b.WriteString(styleHint.Render("Summary:"))
			b.WriteString("\n")
			for _, line := range s.Summary {
				b.WriteString("  • " + line + "\n")
			}
		}
		if a.preview.PRPreview != "" {
			b.WriteString("\n")
			b.WriteString(styleHint.Render(truncate(a.preview.PRPreview, 200)))
		}
	}

	return b.String()
}

func actionConfirmHelp(a *actionState) string {
	parts := styleKey.Render("Enter") + " confirmar  " +
		styleKey.Render("esc") + " cancelar"
	if a.kind == ActionPR {
		parts += "  " + styleKey.Render("d") + " draft"
		if a.draft {
			parts += " " + styleHint.Render("(on)")
		}
	}
	return styleHint.Render("  ") + parts
}

func wrapPreview(text string, width int) string {
	if width < 20 {
		width = 76
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func actionHelpLine() string {
	return styleKey.Render("esc") + " cancelar"
}
