package formatter

import (
	"fmt"
	"strings"

	"github.com/laerciocrestani/gitia/internal/ai"
)

func FormatCommit(s *ai.CommitSuggestion, coAuthor string) string {
	title := s.Type
	if s.Scope != "" {
		title += fmt.Sprintf("(%s)", s.Scope)
	}
	title += ": " + s.Title

	if len(s.Body) == 0 {
		if coAuthor != "" {
			return title + "\n\n" + coAuthor
		}
		return title
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	for i, line := range s.Body {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(line)
	}

	if coAuthor != "" {
		b.WriteString("\n\n")
		b.WriteString(coAuthor)
	}

	return b.String()
}

func TitleLine(message string) string {
	if idx := strings.Index(message, "\n"); idx >= 0 {
		return message[:idx]
	}
	return message
}

func BodyBullets(message string) []string {
	lines := strings.Split(message, "\n")
	var bullets []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			bullets = append(bullets, strings.TrimPrefix(line, "- "))
		}
	}
	return bullets
}
