package gha

import (
	"errors"
	"fmt"
	"strings"
)

// Classified error kinds for onboarding / UI.
var (
	ErrGhNotInstalled = errors.New("gh não encontrado no PATH")
	ErrGhAuth         = errors.New("gh não autenticado ou token inválido")
	ErrForbidden      = errors.New("sem permissão para Actions neste repositório")
	ErrNotFound       = errors.New("recurso não encontrado")
	ErrNetwork        = errors.New("falha de rede ao falar com o GitHub")
)

type Error struct {
	Kind    error
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "erro gha"
}

func (e *Error) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return e.Kind
}

func (e *Error) Is(target error) bool {
	return errors.Is(e.Kind, target) || errors.Is(e.Cause, target)
}

func classifyGhErr(stderr string, cause error) error {
	msg := strings.TrimSpace(stderr)
	low := strings.ToLower(msg + " " + fmt.Sprint(cause))
	switch {
	case strings.Contains(low, "not found the gh") || strings.Contains(low, "executable file not found"):
		return &Error{Kind: ErrGhNotInstalled, Message: ErrGhNotInstalled.Error(), Cause: cause}
	// Org billing / Actions policy scopes must not be confused with "gh auth login".
	case strings.Contains(low, "admin:org") ||
		strings.Contains(low, "billing") ||
		strings.Contains(low, "actions policies") ||
		strings.Contains(low, "fine-grained permission"):
		return &Error{Kind: ErrForbidden, Message: "sem permissão de billing/policies da org (opcional)", Cause: cause}
	case strings.Contains(low, "403") || strings.Contains(low, "forbidden") || strings.Contains(low, "resource not accessible"):
		return &Error{Kind: ErrForbidden, Message: "sem permissão para listar Actions (Forbidden)", Cause: cause}
	case strings.Contains(low, "auth") && (strings.Contains(low, "login") || strings.Contains(low, "refresh") || strings.Contains(low, "token") || strings.Contains(low, "http 401") || strings.Contains(low, "401")):
		return &Error{Kind: ErrGhAuth, Message: "gh auth necessário — rode `gh auth login`", Cause: cause}
	case strings.Contains(low, "404") || strings.Contains(low, "not found"):
		return &Error{Kind: ErrNotFound, Message: msg, Cause: cause}
	case strings.Contains(low, "timeout") || strings.Contains(low, "connection") || strings.Contains(low, "network") || strings.Contains(low, "dial tcp"):
		return &Error{Kind: ErrNetwork, Message: "falha de rede ao falar com o GitHub", Cause: cause}
	default:
		if msg == "" {
			msg = "falha ao executar gh"
		}
		return &Error{Message: msg, Cause: cause}
	}
}
