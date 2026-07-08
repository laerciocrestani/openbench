package components

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitai/internal/app"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

// RenderNextAction renders the suggested next action panel.
func RenderNextAction(action app.TUINextAction, width int) string {
	keyLabel := theme.S.Key.Render("[" + action.Key + "]")
	msg := action.Message
	if action.Label != "" && !strings.Contains(msg, "[") {
		msg = fmt.Sprintf("Press %s for %s.", keyLabel, action.Label)
	} else {
		msg = strings.ReplaceAll(msg, "["+action.Key+"]", keyLabel)
	}
	return RenderPanel("Suggested Action", msg, width)
}
