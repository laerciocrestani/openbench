package components

import (
	"fmt"

	gitpkg "github.com/laerciocrestani/gitai/internal/git"
	"github.com/laerciocrestani/gitai/internal/tui/theme"
)

// RenderAddFileLine renders one selectable row in the add-files screen.
func RenderAddFileLine(selected, current bool, f gitpkg.FileChange) string {
	check := "[ ]"
	if selected {
		check = "[x]"
	}
	prefix := "  " + theme.S.Hint.Render(check) + " "
	tag := statusTag(f.Status)
	line := fmt.Sprintf("%s%-*s %s", prefix, tagWidth, tag, f.Path)
	if current {
		return theme.S.Current.Render(line)
	}
	return fileRowStyle(f.Status).Render(line)
}
