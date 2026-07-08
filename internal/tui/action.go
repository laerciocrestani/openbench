package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/components"
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
	kind         ActionKind
	phase        ActionPhase
	progress     *ActionProgress
	preview      *app.Result
	result       *app.Result
	err          error
	draft        bool
	opts         app.Options
	syncOpts     app.SyncOptions
	editing      bool
	editFocus    editFocus
	editorsReady bool
	commitArea   textarea.Model
	prTitle      textinput.Model
	prBody       textarea.Model
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

func newSyncActionState(opts app.SyncOptions) *actionState {
	a := &actionState{kind: ActionSync, syncOpts: opts}
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
	a.editing = false
	a.editorsReady = false
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
		case ActionPush:
			preview, err := app.PreviewPush(ctx, opts)
			return actionPreviewMsg{kind: a.kind, preview: preview, err: err}
		case ActionPR:
			preview, err := app.PreviewPR(ctx, opts)
			return actionPreviewMsg{kind: a.kind, preview: preview, err: err}
		default:
			return actionPreviewMsg{kind: a.kind, err: fmt.Errorf("action has no preview")}
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
		case ActionPush:
			result, err := app.ConfirmPush(ctx, a.preview, opts)
			return actionConfirmMsg{kind: ActionPush, result: result, err: err}
		case ActionPR:
			result, err := app.ConfirmPR(ctx, a.preview, a.draft, opts)
			return actionConfirmMsg{kind: ActionPR, result: result, err: err}
		default:
			return actionConfirmMsg{err: fmt.Errorf("confirmation not supported")}
		}
	}
}

func (a *actionState) directCmd() tea.Cmd {
	return func() tea.Msg {
		switch a.kind {
		case ActionPush:
			return actionSimpleMsg{kind: a.kind, err: fmt.Errorf("push requires preview confirmation")}
		case ActionSync:
			opts := a.syncOpts
			opts.Progress = a.progress
			err := app.RunSync(opts)
			return actionSimpleMsg{kind: a.kind, err: err}
		case ActionOpenPR:
			client, err := prpkg.New()
			if err != nil {
				return actionSimpleMsg{kind: a.kind, err: err}
			}
			_, err = client.OpenInBrowser()
			return actionSimpleMsg{kind: a.kind, err: err}
		default:
			return actionSimpleMsg{kind: a.kind, err: fmt.Errorf("invalid action")}
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
	a.refreshPRPreview()
}

func (a *actionState) title() string {
	switch a.kind {
	case ActionCommit:
		return "AI Commit"
	case ActionPush:
		return "Push"
	case ActionPR:
		return "Pull Request"
	case ActionSync:
		return "Sync"
	case ActionOpenPR:
		return "Open PR"
	default:
		return "Action"
	}
}

func (a *actionState) View(width, height int) string {
	if a == nil {
		return ""
	}

	if a.phase == PhaseDone {
		status, logs := a.progress.Snapshot()
		var out strings.Builder
		out.WriteString(components.RenderActionDone(a.title(), status, logs, width))
		if a.result != nil {
			if a.result.Message != "" {
				out.WriteString("\n")
				out.WriteString(styleHint.Render(wrapPreview(a.result.Message, width-4)))
			}
			if a.result.PRURL != "" {
				out.WriteString("\n")
				out.WriteString(styleHint.Render(a.result.PRURL))
			}
		}
		return out.String()
	}

	var b strings.Builder
	b.WriteString(styleSection.Render(a.title()))
	b.WriteString("\n\n")

	switch a.phase {
	case PhaseRunning, PhaseConfirming:
		status, logs := a.progress.Snapshot()
		if status == "" {
			status = "Working…"
		}
		b.WriteString(styleHint.Render("  " + status))
		b.WriteString("\n")
		for _, line := range logs {
			b.WriteString(styleHint.Render("  " + line))
			b.WriteString("\n")
		}
		if a.phase == PhaseConfirming {
			b.WriteString("\n")
			b.WriteString(styleHint.Render("  Confirming…"))
		}

	case PhaseConfirm:
		if a.editing {
			b.WriteString(a.renderEditView())
		} else {
			b.WriteString(renderPreview(a))
		}
		b.WriteString("\n")
		b.WriteString(actionConfirmHelp(a))

	case PhaseError:
		if a.err != nil {
			b.WriteString(styleError.Render("  ✗ " + a.err.Error()))
		}
		b.WriteString("\n\n")
		b.WriteString(styleHint.Render("  Press Enter or Esc to go back"))
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
		b.WriteString(styleHint.Render("Commit preview:\n\n"))
		b.WriteString(wrapPreview(a.preview.Message, 76))

	case ActionPush:
		b.WriteString(styleHint.Render("Confirm push:\n\n"))
		if a.preview.Message != "" {
			b.WriteString(styleHint.Render("Commit (AI):\n\n"))
			b.WriteString(wrapPreview(a.preview.Message, 76))
			b.WriteString("\n\n")
		} else {
			b.WriteString(styleHint.Render("  No pending commit — push existing commits only.\n\n"))
		}
		b.WriteString(styleHint.Render("Command:"))
		b.WriteString("\n")
		b.WriteString(styleHint.Render("  git push -u origin HEAD"))

	case ActionPR:
		b.WriteString(styleHint.Render("Confirm Pull Request:\n\n"))
		if a.preview.PRSuggestion != nil {
			s := a.preview.PRSuggestion
			b.WriteString(styleTitle.Render(s.Title))
			b.WriteString("\n\n")
			if a.draft {
				b.WriteString(styleYellow.Render("  [draft]"))
				b.WriteString("\n\n")
			}
		}
		body := prBodyForPreview(a.preview)
		if body != "" {
			b.WriteString(wrapPreview(body, 76))
		}
		if a.preview.PRPreview != "" {
			b.WriteString("\n\n")
			b.WriteString(styleHint.Render("Command:"))
			b.WriteString("\n")
			b.WriteString(styleHint.Render("  " + a.preview.PRPreview))
		}
	}

	return b.String()
}

func actionConfirmHelp(a *actionState) string {
	if a.editing {
		parts := styleKey.Render("esc") + " or " + styleKey.Render("e") + " back to preview"
		if a.kind == ActionPR {
			parts += "  " + styleKey.Render("tab") + " title/body"
		}
		return styleHint.Render("  ") + parts
	}

	parts := styleKey.Render("Enter") + " confirm  " +
		styleKey.Render("esc") + " cancel"
	if a.canEditPreview() {
		parts = styleKey.Render("Enter") + " confirm  " +
			styleKey.Render("e") + " edit  " +
			styleKey.Render("esc") + " cancel"
	}
	if a.kind == ActionPR {
		parts += "  " + styleKey.Render("d") + " draft"
		if a.draft {
			parts += " " + styleHint.Render("(on)")
		}
	}
	return styleHint.Render("  ") + parts
}

func (a *actionState) canEditPreview() bool {
	switch a.kind {
	case ActionCommit, ActionPR:
		return true
	case ActionPush:
		return a.preview != nil && a.preview.Message != ""
	default:
		return false
	}
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
	return styleKey.Render("esc") + " cancel"
}
