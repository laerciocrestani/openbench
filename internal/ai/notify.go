package ai

import (
	"context"
	"fmt"
	"os"
)

type notifier func(msg string)

type notifierKey struct{}

// WithNotifier registra um callback para avisos da camada de IA (retry, fallback).
// Sem notifier, mensagens vão para stderr (modo CLI).
func WithNotifier(ctx context.Context, fn notifier) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, notifierKey{}, fn)
}

func emitNotice(ctx context.Context, msg string) {
	if fn, ok := ctx.Value(notifierKey{}).(notifier); ok && fn != nil {
		fn(msg)
		return
	}
	fmt.Fprintf(os.Stderr, "  %s\n", msg)
}
