package pr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PRDetail is a rich PR payload for the desktop detail dialog.
type PRDetail struct {
	Number      int
	Title       string
	Body        string
	URL         string
	State       string
	IsDraft     bool
	Author      string
	HeadRefName string
	BaseRefName string
	CreatedAt   string
	MergedAt    string
	ClosedAt    string
	Additions   int
	Deletions   int
	ChangedFiles int
	Conversation []PRConversationItem
	Commits     []PRCommitItem
	Checks      []PRCheck
	Files       []PRFileItem
}

// PRConversationItem is one comment/review/description entry.
type PRConversationItem struct {
	Kind    string // description | comment | review
	Author  string
	Body    string
	At      string
	State   string // for reviews: APPROVED, CHANGES_REQUESTED, COMMENTED, …
}

// PRCommitItem is one commit on the PR.
type PRCommitItem struct {
	OID             string
	MessageHeadline string
	Authors         []string
	CommittedDate   string
}

// PRFileItem is one changed file on the PR.
type PRFileItem struct {
	Path      string
	Additions int
	Deletions int
	ChangeType string
}

// LoadDetail loads conversation, commits, checks and files for a PR number.
func (c *Client) LoadDetail(number int) (*PRDetail, error) {
	if number <= 0 {
		return nil, fmt.Errorf("número de PR inválido")
	}
	n := strconv.Itoa(number)
	out, err := c.run(
		"pr", "view", n,
		"--json",
		"number,title,body,url,state,isDraft,author,headRefName,baseRefName,createdAt,mergedAt,closedAt,additions,deletions,changedFiles,commits,files,comments,reviews",
	)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Number       int    `json:"number"`
		Title        string `json:"title"`
		Body         string `json:"body"`
		URL          string `json:"url"`
		State        string `json:"state"`
		IsDraft      bool   `json:"isDraft"`
		HeadRefName  string `json:"headRefName"`
		BaseRefName  string `json:"baseRefName"`
		CreatedAt    string `json:"createdAt"`
		MergedAt     string `json:"mergedAt"`
		ClosedAt     string `json:"closedAt"`
		Additions    int    `json:"additions"`
		Deletions    int    `json:"deletions"`
		ChangedFiles int    `json:"changedFiles"`
		Author       struct {
			Login string `json:"login"`
		} `json:"author"`
		Commits []struct {
			OID             string `json:"oid"`
			MessageHeadline string `json:"messageHeadline"`
			CommittedDate   string `json:"committedDate"`
			Authors         []struct {
				Login string `json:"login"`
				Name  string `json:"name"`
			} `json:"authors"`
		} `json:"commits"`
		Files []struct {
			Path       string `json:"path"`
			Additions  int    `json:"additions"`
			Deletions  int    `json:"deletions"`
			ChangeType string `json:"changeType"`
		} `json:"files"`
		Comments []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
		} `json:"comments"`
		Reviews []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string `json:"body"`
			State     string `json:"state"`
			SubmittedAt string `json:"submittedAt"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, err
	}

	detail := &PRDetail{
		Number:       raw.Number,
		Title:        raw.Title,
		Body:         strings.TrimSpace(raw.Body),
		URL:          raw.URL,
		State:        raw.State,
		IsDraft:      raw.IsDraft,
		Author:       raw.Author.Login,
		HeadRefName:  raw.HeadRefName,
		BaseRefName:  raw.BaseRefName,
		CreatedAt:    raw.CreatedAt,
		MergedAt:     raw.MergedAt,
		ClosedAt:     raw.ClosedAt,
		Additions:    raw.Additions,
		Deletions:    raw.Deletions,
		ChangedFiles: raw.ChangedFiles,
		Commits:      make([]PRCommitItem, 0, len(raw.Commits)),
		Files:        make([]PRFileItem, 0, len(raw.Files)),
	}

	for _, cmt := range raw.Commits {
		authors := make([]string, 0, len(cmt.Authors))
		for _, a := range cmt.Authors {
			name := strings.TrimSpace(a.Login)
			if name == "" {
				name = strings.TrimSpace(a.Name)
			}
			if name != "" {
				authors = append(authors, name)
			}
		}
		detail.Commits = append(detail.Commits, PRCommitItem{
			OID:             cmt.OID,
			MessageHeadline: cmt.MessageHeadline,
			Authors:         authors,
			CommittedDate:   cmt.CommittedDate,
		})
	}
	for _, f := range raw.Files {
		detail.Files = append(detail.Files, PRFileItem{
			Path:       f.Path,
			Additions:  f.Additions,
			Deletions:  f.Deletions,
			ChangeType: f.ChangeType,
		})
	}

	conv := make([]PRConversationItem, 0, 1+len(raw.Comments)+len(raw.Reviews))
	if detail.Body != "" || detail.CreatedAt != "" {
		conv = append(conv, PRConversationItem{
			Kind:   "description",
			Author: detail.Author,
			Body:   detail.Body,
			At:     detail.CreatedAt,
		})
	}
	for _, cmt := range raw.Comments {
		conv = append(conv, PRConversationItem{
			Kind:   "comment",
			Author: cmt.Author.Login,
			Body:   strings.TrimSpace(cmt.Body),
			At:     cmt.CreatedAt,
		})
	}
	for _, rev := range raw.Reviews {
		conv = append(conv, PRConversationItem{
			Kind:   "review",
			Author: rev.Author.Login,
			Body:   strings.TrimSpace(rev.Body),
			At:     rev.SubmittedAt,
			State:  rev.State,
		})
	}
	sort.SliceStable(conv, func(i, j int) bool {
		return parseTime(conv[i].At).Before(parseTime(conv[j].At))
	})
	detail.Conversation = conv
	detail.Checks = c.checksForNumber(number)
	return detail, nil
}

func (c *Client) checksForNumber(number int) []PRCheck {
	cmd := exec.Command("gh", "pr", "checks", strconv.Itoa(number), "--json", "name,state,bucket,link")
	cmd.Dir = c.dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()
	checks, err := parseChecksJSON(strings.TrimSpace(stdout.String()))
	if err != nil {
		return nil
	}
	return checks
}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	return time.Time{}
}
