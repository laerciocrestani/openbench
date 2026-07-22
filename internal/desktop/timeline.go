package desktop

import (
	"fmt"
	"sort"
	"strings"
	"time"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
	prpkg "github.com/laerciocrestani/openbench/internal/pr"
)

// Timeline event kinds for the desktop UI.
const (
	TimelineKindCommit   = "commit"
	TimelineKindMerge    = "merge"
	TimelineKindPROpened = "pr_opened"
	TimelineKindPRMerged = "pr_merged"
	TimelineKindPRClosed = "pr_closed"
)

// TimelineEventView is one row in the activity timeline.
type TimelineEventView struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	At        string   `json:"at"` // RFC3339
	Title     string   `json:"title"`
	Subtitle  string   `json:"subtitle"`
	Author    string   `json:"author,omitempty"`
	Hash      string   `json:"hash,omitempty"`
	ShortHash string   `json:"shortHash,omitempty"`
	URL       string   `json:"url,omitempty"`
	PRNumber  int      `json:"prNumber,omitempty"`
	Refs      []string `json:"refs,omitempty"`
}

// TimelineView is the desktop DTO for the GitLens-style MVP timeline.
type TimelineView struct {
	Events     []TimelineEventView `json:"events"`
	HasGH      bool                `json:"hasGH"`
	PRIncluded bool                `json:"prIncluded"`
	Limit      int                 `json:"limit"`
	HasMore    bool                `json:"hasMore"`
}

// LoadTimeline builds a unified timeline of commits + PR events.
func LoadTimeline(projectPath string, limit int) (*TimelineView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	if limit <= 0 {
		limit = 50
	}

	repo, err := gitpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	// Fetch a bit beyond the page size so HasMore is reliable after merge/sort.
	commits, err := repo.LoadTimelineCommits(limit + 15)
	if err != nil {
		return nil, err
	}

	view := &TimelineView{
		Events: make([]TimelineEventView, 0, limit*2),
		HasGH:  false,
	}

	type stamped struct {
		at time.Time
		ev TimelineEventView
	}
	items := make([]stamped, 0, limit*2)

	for _, c := range commits {
		kind := TimelineKindCommit
		title := c.Subject
		if c.IsMerge {
			kind = TimelineKindMerge
		}
		subtitle := c.ShortHash
		if len(c.Refs) > 0 {
			subtitle += " · " + strings.Join(c.Refs, ", ")
		}
		items = append(items, stamped{
			at: c.At,
			ev: TimelineEventView{
				ID:        "commit:" + c.Hash,
				Kind:      kind,
				At:        c.At.UTC().Format(time.RFC3339),
				Title:     title,
				Subtitle:  subtitle,
				Author:    c.Author,
				Hash:      c.Hash,
				ShortHash: c.ShortHash,
				Refs:      c.Refs,
			},
		})
	}

	if client, err := prpkg.Open(projectPath); err == nil {
		view.HasGH = true
		if prs, err := client.ListRecent(30); err == nil {
			view.PRIncluded = true
			for _, pr := range prs {
				author := pr.Author
				items = append(items, stamped{
					at: pr.CreatedAt,
					ev: TimelineEventView{
						ID:       fmt.Sprintf("pr_opened:%d", pr.Number),
						Kind:     TimelineKindPROpened,
						At:       pr.CreatedAt.UTC().Format(time.RFC3339),
						Title:    fmt.Sprintf("PR #%d aberto: %s", pr.Number, pr.Title),
						Subtitle: branchSubtitle(pr.HeadRefName, pr.IsDraft),
						Author:   author,
						URL:      pr.URL,
						PRNumber: pr.Number,
					},
				})
				if pr.MergedAt != nil {
					items = append(items, stamped{
						at: *pr.MergedAt,
						ev: TimelineEventView{
							ID:       fmt.Sprintf("pr_merged:%d", pr.Number),
							Kind:     TimelineKindPRMerged,
							At:       pr.MergedAt.UTC().Format(time.RFC3339),
							Title:    fmt.Sprintf("PR #%d mergeado: %s", pr.Number, pr.Title),
							Subtitle: branchSubtitle(pr.HeadRefName, false),
							Author:   author,
							URL:      pr.URL,
							PRNumber: pr.Number,
						},
					})
				} else if pr.ClosedAt != nil && !strings.EqualFold(pr.State, "OPEN") {
					items = append(items, stamped{
						at: *pr.ClosedAt,
						ev: TimelineEventView{
							ID:       fmt.Sprintf("pr_closed:%d", pr.Number),
							Kind:     TimelineKindPRClosed,
							At:       pr.ClosedAt.UTC().Format(time.RFC3339),
							Title:    fmt.Sprintf("PR #%d fechado: %s", pr.Number, pr.Title),
							Subtitle: branchSubtitle(pr.HeadRefName, false),
							Author:   author,
							URL:      pr.URL,
							PRNumber: pr.Number,
						},
					})
				}
			}
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].at.After(items[j].at)
	})
	view.Limit = limit
	view.HasMore = len(items) > limit
	if len(items) > limit {
		items = items[:limit]
	}
	for _, it := range items {
		view.Events = append(view.Events, it.ev)
	}
	return view, nil
}

func branchSubtitle(branch string, draft bool) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(branch) != "" {
		parts = append(parts, branch)
	}
	if draft {
		parts = append(parts, "draft")
	}
	return strings.Join(parts, " · ")
}
