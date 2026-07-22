package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/ai"
	"github.com/laerciocrestani/openbench/internal/app"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

// PRPreview is the AI-generated PR awaiting human review.
type PRPreview struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	Base  string   `json:"base"`
	Draft bool     `json:"draft"`
	Notes []string `json:"notes"`
}

// PROutcome is returned after a successful PR create.
type PROutcome struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// PreviewPR generates PR title/body via AI (dry-run; no push/create).
func PreviewPR(ctx context.Context, projectPath string, draft bool) (*PRPreview, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	result, err := app.PreviewPR(ctx, app.Options{
		WorkDir:  projectPath,
		Draft:    draft,
		Progress: app.NopProgress(),
	})
	if err != nil {
		return nil, err
	}
	if result == nil || result.PRSuggestion == nil {
		return nil, fmt.Errorf("IA não retornou sugestão de PR")
	}
	body := result.PRBody
	if strings.TrimSpace(body) == "" {
		body = prpkg.FormatBody(result.PRSuggestion)
	}
	preview := &PRPreview{
		Title: result.PRSuggestion.Title,
		Body:  body,
		Draft: draft,
		Notes: append([]string{}, result.PRSuggestion.Notes...),
	}
	return preview, nil
}

// ConfirmPR pushes and creates the PR with the reviewed title/body.
func ConfirmPR(ctx context.Context, projectPath, title, body string, draft bool) (*PROutcome, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		return nil, fmt.Errorf("título do PR vazio")
	}
	if body == "" {
		return nil, fmt.Errorf("body do PR vazio")
	}

	result, err := app.ConfirmPR(ctx, &app.Result{
		PRSuggestion: &ai.PRSuggestion{Title: title},
		PRBody:       body,
	}, draft, app.Options{
		WorkDir:  projectPath,
		Progress: app.NopProgress(),
	})
	if err != nil {
		return nil, err
	}
	out := &PROutcome{Path: projectPath, Title: title}
	if result != nil {
		out.URL = result.PRURL
		if result.PRSuggestion != nil && result.PRSuggestion.Title != "" {
			out.Title = result.PRSuggestion.Title
		}
	}
	if out.URL == "" {
		return nil, fmt.Errorf("PR criado sem URL retornada")
	}
	return out, nil
}

// MarkPRReady marks the current draft PR as ready for review.
func MarkPRReady(projectPath string) (*PRStatus, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	client, err := prpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	if err := client.Ready(); err != nil {
		return nil, err
	}
	pr, err := client.ViewCurrent()
	if err != nil {
		return nil, err
	}
	return mapPRStatus(pr), nil
}

// MergePR merges the current branch PR. method: squash|merge|rebase.
func MergePR(projectPath, method string) (*PROutcome, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	client, err := prpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	view, err := client.ViewCurrent()
	if err != nil {
		return nil, err
	}
	if view == nil {
		return nil, fmt.Errorf("nenhum PR aberto nesta branch")
	}
	if _, err := client.Merge(method); err != nil {
		return nil, err
	}
	return &PROutcome{
		URL:   view.URL,
		Title: view.Title,
		Path:  projectPath,
	}, nil
}
