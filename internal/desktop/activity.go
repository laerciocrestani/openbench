package desktop

import (
	"fmt"
	"strings"

	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

// CommitActivityView is the desktop DTO for the contribution calendar.
type CommitActivityView struct {
	AuthorEmail string           `json:"authorEmail"`
	AuthorName  string           `json:"authorName"`
	Since       string           `json:"since"`
	Until       string           `json:"until"`
	Total       int              `json:"total"`
	AuthorOnly  bool             `json:"authorOnly"`
	Days        []DayActivityView `json:"days"`
}

// DayActivityView is one day with commits.
type DayActivityView struct {
	Date    string            `json:"date"`
	Count   int               `json:"count"`
	Commits []CommitEntryView `json:"commits"`
}

// CommitEntryView is one commit in the calendar day list.
type CommitEntryView struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Subject   string `json:"subject"`
	Author    string `json:"author"`
	Date      string `json:"date"`
}

// LoadCommitActivity loads a GitHub-style commit calendar for projectPath.
// authorOnly filters by local git user.email when true.
func LoadCommitActivity(projectPath string, authorOnly bool) (*CommitActivityView, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("no project open")
	}
	repo, err := gitpkg.Open(projectPath)
	if err != nil {
		return nil, err
	}
	if err := repo.IsRepo(); err != nil {
		return nil, fmt.Errorf("diretório atual não é um repositório git")
	}

	act, err := repo.LoadCommitActivity(365, authorOnly)
	if err != nil {
		return nil, err
	}

	view := &CommitActivityView{
		AuthorEmail: act.AuthorEmail,
		AuthorName:  act.AuthorName,
		Since:       act.Since,
		Until:       act.Until,
		Total:       act.Total,
		AuthorOnly:  authorOnly,
		Days:        make([]DayActivityView, 0, len(act.Days)),
	}
	for _, d := range act.Days {
		day := DayActivityView{
			Date:    d.Date,
			Count:   d.Count,
			Commits: make([]CommitEntryView, 0, len(d.Commits)),
		}
		for _, c := range d.Commits {
			day.Commits = append(day.Commits, CommitEntryView{
				Hash:      c.Hash,
				ShortHash: c.ShortHash,
				Subject:   c.Subject,
				Author:    c.Author,
				Date:      c.Date,
			})
		}
		view.Days = append(view.Days, day)
	}
	return view, nil
}
