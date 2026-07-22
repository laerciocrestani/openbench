package pr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/laerciocrestani/openbench/internal/ai"
)

type PRView struct {
	URL            string
	Title          string
	State          string
	Number         int
	IsDraft        bool
	Mergeable      string // MERGEABLE, CONFLICTING, UNKNOWN
	ReviewDecision string
	ChecksPass     int
	ChecksFail     int
	ChecksPending  int
	ChecksTotal    int
	ChecksSummary  string
}

// PRCheck is one CI check from gh pr checks.
type PRCheck struct {
	Name   string
	State  string
	Bucket string
	Link   string
}

type Client struct {
	dir string
}

func New() (*Client, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return Open(dir)
}

// Open returns a gh client bound to dir.
func Open(dir string) (*Client, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &Client{dir: abs}, nil
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

func (c *Client) OpenInBrowser() (*PRView, error) {
	view, err := c.ViewCurrent()
	if err != nil {
		return nil, err
	}
	if view == nil {
		return nil, fmt.Errorf("nenhum PR para a branch atual")
	}
	if _, err := c.run("pr", "view", "--web"); err != nil {
		return nil, err
	}
	return view, nil
}

// ListOpen returns all open pull requests keyed by head branch name.
func (c *Client) ListOpen() (map[string]PRView, error) {
	out, err := c.run("pr", "list", "--state", "open", "--json", "title,url,state,number,isDraft,headRefName")
	if err != nil {
		if isPRNotFound(err) {
			return map[string]PRView{}, nil
		}
		return nil, err
	}

	var raw []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		State       string `json:"state"`
		Number      int    `json:"number"`
		IsDraft     bool   `json:"isDraft"`
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}

	byHead := make(map[string]PRView, len(raw))
	for _, item := range raw {
		if item.HeadRefName == "" {
			continue
		}
		byHead[item.HeadRefName] = PRView{
			URL:     item.URL,
			Title:   item.Title,
			State:   item.State,
			Number:  item.Number,
			IsDraft: item.IsDraft,
		}
	}
	return byHead, nil
}

func (c *Client) ViewCurrent() (*PRView, error) {
	out, err := c.run("pr", "view", "--json", "title,url,state,number,isDraft,mergeable,reviewDecision")
	if err != nil {
		if isPRNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	var raw struct {
		Title          string `json:"title"`
		URL            string `json:"url"`
		State          string `json:"state"`
		Number         int    `json:"number"`
		IsDraft        bool   `json:"isDraft"`
		Mergeable      string `json:"mergeable"`
		ReviewDecision string `json:"reviewDecision"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}

	view := &PRView{
		URL:            raw.URL,
		Title:          raw.Title,
		State:          raw.State,
		Number:         raw.Number,
		IsDraft:        raw.IsDraft,
		Mergeable:      raw.Mergeable,
		ReviewDecision: raw.ReviewDecision,
	}
	c.enrichChecks(view)
	return view, nil
}

func (c *Client) enrichChecks(view *PRView) {
	if view == nil {
		return
	}
	checks := c.ChecksBestEffort()
	if len(checks) == 0 {
		view.ChecksSummary = "sem checks"
		return
	}
	view.ChecksTotal = len(checks)
	for _, ch := range checks {
		switch strings.ToLower(ch.Bucket) {
		case "pass":
			view.ChecksPass++
		case "fail":
			view.ChecksFail++
		case "pending":
			view.ChecksPending++
		}
	}
	switch {
	case view.ChecksFail > 0:
		view.ChecksSummary = fmt.Sprintf("%d fail · %d pass", view.ChecksFail, view.ChecksPass)
	case view.ChecksPending > 0:
		view.ChecksSummary = fmt.Sprintf("%d pending · %d pass", view.ChecksPending, view.ChecksPass)
	default:
		view.ChecksSummary = fmt.Sprintf("%d pass", view.ChecksPass)
	}
}

// Checks returns CI checks for the PR on the current branch.
func (c *Client) Checks() ([]PRCheck, error) {
	out, err := c.run("pr", "checks", "--json", "name,state,bucket,link")
	if err != nil {
		// gh exits non-zero when checks are pending/failing; still try to parse stdout via raw.
		return nil, err
	}
	return parseChecksJSON(out)
}

// ChecksBestEffort returns checks even when gh exits non-zero (pending/fail).
func (c *Client) ChecksBestEffort() []PRCheck {
	cmd := exec.Command("gh", "pr", "checks", "--json", "name,state,bucket,link")
	cmd.Dir = c.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	checks, err := parseChecksJSON(strings.TrimSpace(stdout.String()))
	if err != nil {
		return nil
	}
	return checks
}

func parseChecksJSON(out string) ([]PRCheck, error) {
	out = strings.TrimSpace(out)
	if out == "" || out == "null" {
		return nil, nil
	}
	var raw []struct {
		Name   string `json:"name"`
		State  string `json:"state"`
		Bucket string `json:"bucket"`
		Link   string `json:"link"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}
	checks := make([]PRCheck, 0, len(raw))
	for _, item := range raw {
		checks = append(checks, PRCheck{
			Name:   item.Name,
			State:  item.State,
			Bucket: item.Bucket,
			Link:   item.Link,
		})
	}
	return checks, nil
}

// Ready marks the current draft PR as ready for review.
func (c *Client) Ready() error {
	_, err := c.run("pr", "ready")
	return err
}

// Merge merges the current branch PR with the given method: squash|merge|rebase.
func (c *Client) Merge(method string) (string, error) {
	method = strings.ToLower(strings.TrimSpace(method))
	flag := "--squash"
	switch method {
	case "", "squash":
		flag = "--squash"
	case "merge":
		flag = "--merge"
	case "rebase":
		flag = "--rebase"
	default:
		return "", fmt.Errorf("método de merge inválido: %s (use squash, merge ou rebase)", method)
	}
	out, err := c.run("pr", "merge", flag)
	if err != nil {
		return "", err
	}
	return out, nil
}

// PRTimelineItem is a pull request with timestamps for the activity timeline.
type PRTimelineItem struct {
	Number      int
	Title       string
	URL         string
	State       string
	IsDraft     bool
	HeadRefName string
	Author      string
	CreatedAt   time.Time
	MergedAt    *time.Time
	ClosedAt    *time.Time
}

// ListRecent returns recent PRs (any state) for timeline events.
func (c *Client) ListRecent(limit int) ([]PRTimelineItem, error) {
	if limit <= 0 {
		limit = 30
	}
	out, err := c.run(
		"pr", "list",
		"--state", "all",
		"--limit", fmt.Sprintf("%d", limit),
		"--json", "number,title,url,state,isDraft,createdAt,mergedAt,closedAt,headRefName,author",
	)
	if err != nil {
		if isPRNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		URL         string `json:"url"`
		State       string `json:"state"`
		IsDraft     bool   `json:"isDraft"`
		CreatedAt   string `json:"createdAt"`
		MergedAt    string `json:"mergedAt"`
		ClosedAt    string `json:"closedAt"`
		HeadRefName string `json:"headRefName"`
		Author      struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}

	items := make([]PRTimelineItem, 0, len(raw))
	for _, item := range raw {
		created, err := time.Parse(time.RFC3339, item.CreatedAt)
		if err != nil {
			continue
		}
		pr := PRTimelineItem{
			Number:      item.Number,
			Title:       item.Title,
			URL:         item.URL,
			State:       item.State,
			IsDraft:     item.IsDraft,
			HeadRefName: item.HeadRefName,
			Author:      item.Author.Login,
			CreatedAt:   created,
		}
		if t := parseOptionalTime(item.MergedAt); t != nil {
			pr.MergedAt = t
		}
		if t := parseOptionalTime(item.ClosedAt); t != nil {
			pr.ClosedAt = t
		}
		items = append(items, pr)
	}
	return items, nil
}

func parseOptionalTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}

func isPRNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "no pull requests") ||
		strings.Contains(msg, "could not find") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no default remote")
}

func resolveBody(suggestion *ai.PRSuggestion, bodyOverride string) string {
	if strings.TrimSpace(bodyOverride) != "" {
		return bodyOverride
	}
	return FormatBody(suggestion)
}

func (c *Client) Create(suggestion *ai.PRSuggestion, base string, draft bool, bodyOverride string) (string, error) {
	exists, url, err := c.Exists()
	if err != nil {
		return "", err
	}
	if exists {
		return url, fmt.Errorf("PR já existe: %s", url)
	}

	body := resolveBody(suggestion, bodyOverride)

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

func (c *Client) PreviewCreate(suggestion *ai.PRSuggestion, base string, draft bool, bodyOverride string) string {
	body := resolveBody(suggestion, bodyOverride)
	draftFlag := ""
	if draft {
		draftFlag = " --draft"
	}
	return fmt.Sprintf("gh pr create --title %q --body %q --base %q%s",
		suggestion.Title, body, baseForGH(base), draftFlag)
}
