package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/config"
	prpkg "github.com/laerciocrestani/gitai/internal/pr"
)

type editFocus int

const (
	focusCommitMessage editFocus = iota
	focusPRTitle
	focusPRBody
)

func (a *actionState) resizeEditors(width, height int) {
	if a == nil || !a.editorsReady {
		return
	}
	innerW := editorWidth(width)
	editH := editorHeight(height)

	switch a.kind {
	case ActionCommit:
		a.commitArea.SetWidth(innerW)
		a.commitArea.SetHeight(editH)
	case ActionPR:
		a.prTitle.Width = innerW
		a.prBody.SetWidth(innerW)
		bodyH := editH - 2
		if bodyH < 4 {
			bodyH = 4
		}
		a.prBody.SetHeight(bodyH)
	}
}

func editorWidth(width int) int {
	innerW := width - 4
	if innerW < 40 {
		return 40
	}
	return innerW
}

func editorHeight(height int) int {
	editH := height - 14
	if editH < 6 {
		return 6
	}
	return editH
}

func (a *actionState) initEditors(width, height int) {
	if a == nil || a.preview == nil {
		return
	}

	innerW := editorWidth(width)
	editH := editorHeight(height)

	switch a.kind {
	case ActionCommit:
		a.commitArea = textarea.New()
		a.commitArea.SetWidth(innerW)
		a.commitArea.SetHeight(editH)
		a.commitArea.SetValue(a.preview.Message)
		a.commitArea.ShowLineNumbers = false
		a.editFocus = focusCommitMessage
	case ActionPR:
		a.prTitle = textinput.New()
		a.prTitle.Width = innerW
		a.prTitle.Placeholder = "Título do PR"
		if a.preview.PRSuggestion != nil {
			a.prTitle.SetValue(a.preview.PRSuggestion.Title)
		}

		body := prBodyForPreview(a.preview)
		a.prBody = textarea.New()
		a.prBody.SetWidth(innerW)
		bodyH := editH - 2
		if bodyH < 4 {
			bodyH = 4
		}
		a.prBody.SetHeight(bodyH)
		a.prBody.SetValue(body)
		a.prBody.ShowLineNumbers = false
		a.editFocus = focusPRTitle
	}

	a.editorsReady = true
}

func (a *actionState) enterEdit(width, height int) tea.Cmd {
	if !a.editorsReady {
		a.initEditors(width, height)
	} else {
		a.syncEditorsFromPreview()
		a.resizeEditors(width, height)
	}
	a.editing = true
	return a.focusEditor()
}

func (a *actionState) exitEdit() {
	a.syncPreviewFromEditors()
	a.editing = false
}

func (a *actionState) focusEditor() tea.Cmd {
	switch a.kind {
	case ActionCommit:
		return a.commitArea.Focus()
	case ActionPR:
		switch a.editFocus {
		case focusPRTitle:
			a.prBody.Blur()
			return a.prTitle.Focus()
		default:
			a.prTitle.Blur()
			return a.prBody.Focus()
		}
	}
	return nil
}

func (a *actionState) cyclePRFocus() tea.Cmd {
	if a.kind != ActionPR {
		return nil
	}
	if a.editFocus == focusPRTitle {
		a.editFocus = focusPRBody
	} else {
		a.editFocus = focusPRTitle
	}
	return a.focusEditor()
}

func (a *actionState) syncEditorsFromPreview() {
	if a == nil || a.preview == nil || !a.editorsReady {
		return
	}

	switch a.kind {
	case ActionCommit:
		a.commitArea.SetValue(a.preview.Message)
	case ActionPR:
		if a.preview.PRSuggestion != nil {
			a.prTitle.SetValue(a.preview.PRSuggestion.Title)
		}
		a.prBody.SetValue(prBodyForPreview(a.preview))
	}
}

func (a *actionState) syncPreviewFromEditors() {
	if a == nil || a.preview == nil {
		return
	}
	if !a.editorsReady {
		return
	}

	switch a.kind {
	case ActionCommit:
		a.preview.Message = a.commitArea.Value()
	case ActionPR:
		if a.preview.PRSuggestion != nil {
			a.preview.PRSuggestion.Title = strings.TrimSpace(a.prTitle.Value())
		}
		a.preview.PRBody = a.prBody.Value()
		a.refreshPRPreview()
	}
}

func prBodyForPreview(preview *app.Result) string {
	if preview == nil {
		return ""
	}
	if strings.TrimSpace(preview.PRBody) != "" {
		return preview.PRBody
	}
	if preview.PRSuggestion != nil {
		return prpkg.FormatBody(preview.PRSuggestion)
	}
	return ""
}

func (a *actionState) refreshPRPreview() {
	if a.preview == nil || a.preview.PRSuggestion == nil {
		return
	}
	base := "main"
	if cfg, err := config.Load(); err == nil && cfg.BaseBranch != "" {
		base = cfg.BaseBranch
	}
	client, err := prpkg.New()
	if err == nil {
		a.preview.PRPreview = client.PreviewCreate(a.preview.PRSuggestion, base, a.draft, a.preview.PRBody)
	}
}

func (a *actionState) updateEditors(msg tea.Msg) (*actionState, tea.Cmd) {
	switch a.kind {
	case ActionCommit:
		var cmd tea.Cmd
		a.commitArea, cmd = a.commitArea.Update(msg)
		return a, cmd
	case ActionPR:
		var cmd tea.Cmd
		switch a.editFocus {
		case focusPRTitle:
			a.prTitle, cmd = a.prTitle.Update(msg)
		default:
			a.prBody, cmd = a.prBody.Update(msg)
		}
		return a, cmd
	default:
		return a, nil
	}
}

func (a *actionState) renderEditView() string {
	var b strings.Builder

	switch a.kind {
	case ActionCommit:
		b.WriteString(styleHint.Render("Edite a mensagem do commit:"))
		b.WriteString("\n\n")
		b.WriteString(a.commitArea.View())
	case ActionPR:
		b.WriteString(styleHint.Render("Título:"))
		b.WriteString("\n")
		b.WriteString(a.prTitle.View())
		b.WriteString("\n\n")
		b.WriteString(styleHint.Render("Corpo do PR:"))
		b.WriteString("\n")
		b.WriteString(a.prBody.View())
	}

	return b.String()
}
