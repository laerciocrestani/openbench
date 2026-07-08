package app

import (
	"context"

	"github.com/laerciocrestani/gitai/internal/ai"
)

func withAINotices(ctx context.Context, prog Progress) context.Context {
	if prog == nil {
		return ctx
	}
	return ai.WithNotifier(ctx, func(msg string) {
		prog.Warn(msg)
	})
}
