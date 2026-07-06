package pr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/laerciocrestani/gitai/internal/ai"
)

type PRView struct {
	URL     string
	Title   string
	State   string
	Number  int
	IsDraft bool
}

type Client struct {
	dir string
}

func New() (*Client, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &Client{dir: dir}, nil
}

func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = c.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (c *Client) Exists() (bool, string, error) {
	view, err := c.ViewCurrent()
	if err != nil {
		return false, "", err
	}
	if view == nil {
		return false, "", nil
	}
	return true, view.URL, nil
}

func (c *Client) ViewCurrent() (*PRView, error) {
	out, err := c.run("pr", "view", "--json", "title,url,state,number,isDraft")
	if err != nil {
		if isPRNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var raw struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		State   string `json:"state"`
		Number  int    `json:"number"`
		IsDraft bool   `json:"isDraft"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}

	return &PRView{
		URL:     raw.URL,
		Title:   raw.Title,
		State:   raw.State,
		Number:  raw.Number,
		IsDraft: raw.IsDraft,
	}, nil
}

func isPRNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "no pull requests") ||
		strings.Contains(msg, "could not find") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no default remote")
}

func (c *Client) Create(suggestion *ai.PRSuggestion, base string, draft bool) (string, error) {
	exists, url, err := c.Exists()
	if err != nil {
		return "", err
	}
	if exists {
		return url, fmt.Errorf("PR já existe: %s", url)
	}

	body := FormatBody(suggestion)

	args := []string{
		"pr", "create",
		"--title", suggestion.Title,
		"--body", body,
		"--base", baseForGH(base),
	}
	if draft {
		args = append(args, "--draft")
	}

	return c.run(args...)
}

func FormatBody(s *ai.PRSuggestion) string {
	var b strings.Builder

	b.WriteString("## Summary\n")
	writeBullets(&b, s.Summary)

	b.WriteString("\n## Changes\n")
	writeBullets(&b, s.Changes)

	b.WriteString("\n## Test plan\n")
	writeChecklist(&b, s.TestPlan)

	if len(s.Notes) > 0 {
		b.WriteString("\n## Notes\n")
		writeBullets(&b, s.Notes)
	}

	return b.String()
}

func writeBullets(b *strings.Builder, items []string) {
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(strings.TrimSpace(item))
		b.WriteString("\n")
	}
}

func writeChecklist(b *strings.Builder, items []string) {
	for _, item := range items {
		b.WriteString("- [ ] ")
		b.WriteString(strings.TrimSpace(item))
		b.WriteString("\n")
	}
}

func baseForGH(base string) string {
	return strings.TrimPrefix(base, "origin/")
}

func (c *Client) PreviewCreate(suggestion *ai.PRSuggestion, base string, draft bool) string {
	body := FormatBody(suggestion)
	draftFlag := ""
	if draft {
		draftFlag = " --draft"
	}
	return fmt.Sprintf("gh pr create --title %q --body %q --base %q%s",
		suggestion.Title, body, baseForGH(base), draftFlag)
}
