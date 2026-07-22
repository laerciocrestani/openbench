package git

import (
	"fmt"
	"strings"
	"time"
)

// DayActivity is commits on a single calendar day (YYYY-MM-DD).
type DayActivity struct {
	Date    string
	Count   int
	Commits []CommitEntry
}

// CommitEntry is one commit for the activity calendar.
type CommitEntry struct {
	Hash      string
	ShortHash string
	Subject   string
	Author    string
	Date      string
}

// CommitActivity is a GitHub-style contribution calendar payload.
type CommitActivity struct {
	AuthorEmail string
	AuthorName  string
	Since       string
	Until       string
	Total       int
	Days        []DayActivity
}

// LoadCommitActivity returns commits by day for the last `days` days.
// When authorOnly is true, filters by local git user.email (GitHub-style "my commits").
func (r *Repo) LoadCommitActivity(days int, authorOnly bool) (*CommitActivity, error) {
	if days <= 0 {
		days = 365
	}
	until := time.Now()
	since := until.AddDate(0, 0, -days)

	email, _ := r.run("config", "user.email")
	name, _ := r.run("config", "user.name")

	args := []string{
		"log",
		"--all",
		"--no-merges",
		"--pretty=format:%h%x00%H%x00%ad%x00%an%x00%ae%x00%s",
		"--date=short",
		"--since=" + since.Format("2006-01-02"),
		"--until=" + until.AddDate(0, 0, 1).Format("2006-01-02"),
	}
	if authorOnly && strings.TrimSpace(email) != "" {
		args = append(args, "--author="+email)
	}

	out, err := r.run(args...)
	if err != nil {
		return nil, err
	}

	byDay := map[string]*DayActivity{}
	total := 0
	if strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\x00")
			if len(parts) < 6 {
				continue
			}
			date := strings.TrimSpace(parts[2])
			if date == "" {
				continue
			}
			entry := CommitEntry{
				ShortHash: parts[0],
				Hash:      parts[1],
				Date:      date,
				Author:    parts[3],
				Subject:   parts[5],
			}
			day := byDay[date]
			if day == nil {
				day = &DayActivity{Date: date}
				byDay[date] = day
			}
			day.Commits = append(day.Commits, entry)
			day.Count++
			total++
		}
	}

	daysOut := make([]DayActivity, 0, len(byDay))
	for d := since; !d.After(until); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if day, ok := byDay[key]; ok {
			daysOut = append(daysOut, *day)
		}
	}

	return &CommitActivity{
		AuthorEmail: email,
		AuthorName:  name,
		Since:       since.Format("2006-01-02"),
		Until:       until.Format("2006-01-02"),
		Total:       total,
		Days:        daysOut,
	}, nil
}

// ConfigUserEmail returns git user.email for the repo.
func (r *Repo) ConfigUserEmail() (string, error) {
	out, err := r.run("config", "user.email")
	if err != nil {
		return "", fmt.Errorf("git user.email: %w", err)
	}
	return out, nil
}
