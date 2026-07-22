package desktop

import (
	"fmt"
	"strings"

	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

// PRConversationItemView is one conversation entry.
type PRConversationItemView struct {
	Kind   string `json:"kind"`
	Author string `json:"author"`
	Body   string `json:"body"`
	At     string `json:"at"`
	State  string `json:"state,omitempty"`
}

// PRCommitItemView is one commit on the PR.
type PRCommitItemView struct {
	OID             string   `json:"oid"`
	ShortOID        string   `json:"shortOid"`
	MessageHeadline string   `json:"messageHeadline"`
	Authors         []string `json:"authors,omitempty"`
	CommittedDate   string   `json:"committedDate,omitempty"`
}

// PRCheckItemView is one CI check.
type PRCheckItemView struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Bucket string `json:"bucket"`
	Link   string `json:"link,omitempty"`
}

// PRFileItemView is one changed file.
type PRFileItemView struct {
	Path       string `json:"path"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
	ChangeType string `json:"changeType,omitempty"`
}

// PRDetailView is the desktop DTO for the PR detail dialog.
type PRDetailView struct {
	Number       int                      `json:"number"`
	Title        string                   `json:"title"`
	Body         string                   `json:"body"`
	URL          string                   `json:"url"`
	State        string                   `json:"state"`
	IsDraft      bool                     `json:"isDraft"`
	Author       string                   `json:"author"`
	HeadRefName  string                   `json:"headRefName"`
	BaseRefName  string                   `json:"baseRefName"`
	CreatedAt    string                   `json:"createdAt,omitempty"`
	MergedAt     string                   `json:"mergedAt,omitempty"`
	ClosedAt     string                   `json:"closedAt,omitempty"`
	Additions    int                      `json:"additions"`
	Deletions    int                      `json:"deletions"`
	ChangedFiles int                      `json:"changedFiles"`
	Conversation []PRConversationItemView `json:"conversation"`
	Commits      []PRCommitItemView       `json:"commits"`
	Checks       []PRCheckItemView        `json:"checks"`
	Files        []PRFileItemView         `json:"files"`
}

// LoadPRDetail loads rich PR details for the open project.
func LoadPRDetail(projectPath string, number int) (*PRDetailView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	if number <= 0 {
		return nil, fmt.Errorf("número de PR inválido")
	}
	client, err := prpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	detail, err := client.LoadDetail(number)
	if err != nil {
		return nil, err
	}
	return mapPRDetail(detail), nil
}

func mapPRDetail(d *prpkg.PRDetail) *PRDetailView {
	if d == nil {
		return nil
	}
	view := &PRDetailView{
		Number:       d.Number,
		Title:        d.Title,
		Body:         d.Body,
		URL:          d.URL,
		State:        d.State,
		IsDraft:      d.IsDraft,
		Author:       d.Author,
		HeadRefName:  d.HeadRefName,
		BaseRefName:  d.BaseRefName,
		CreatedAt:    d.CreatedAt,
		MergedAt:     d.MergedAt,
		ClosedAt:     d.ClosedAt,
		Additions:    d.Additions,
		Deletions:    d.Deletions,
		ChangedFiles: d.ChangedFiles,
		Conversation: make([]PRConversationItemView, 0, len(d.Conversation)),
		Commits:      make([]PRCommitItemView, 0, len(d.Commits)),
		Checks:       make([]PRCheckItemView, 0, len(d.Checks)),
		Files:        make([]PRFileItemView, 0, len(d.Files)),
	}
	for _, item := range d.Conversation {
		view.Conversation = append(view.Conversation, PRConversationItemView{
			Kind:   item.Kind,
			Author: item.Author,
			Body:   item.Body,
			At:     item.At,
			State:  item.State,
		})
	}
	for _, c := range d.Commits {
		short := c.OID
		if len(short) > 7 {
			short = short[:7]
		}
		view.Commits = append(view.Commits, PRCommitItemView{
			OID:             c.OID,
			ShortOID:        short,
			MessageHeadline: c.MessageHeadline,
			Authors:         append([]string{}, c.Authors...),
			CommittedDate:   c.CommittedDate,
		})
	}
	for _, ch := range d.Checks {
		view.Checks = append(view.Checks, PRCheckItemView{
			Name:   ch.Name,
			State:  ch.State,
			Bucket: ch.Bucket,
			Link:   ch.Link,
		})
	}
	for _, f := range d.Files {
		view.Files = append(view.Files, PRFileItemView{
			Path:       f.Path,
			Additions:  f.Additions,
			Deletions:  f.Deletions,
			ChangeType: f.ChangeType,
		})
	}
	return view
}
