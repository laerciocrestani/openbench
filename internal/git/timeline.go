package git

import (
	"fmt"
	"strings"
	"time"
)

// TimelineCommit is one commit entry for the activity timeline.
type TimelineCommit struct {
	Hash      string
	ShortHash string
	At        time.Time
	Author    string
	Subject   string
	IsMerge   bool
	Refs      []string // decorated refs (branches/tags)
}

// LoadTimelineCommits returns recent commits across all refs (newest first).
func (r *Repo) LoadTimelineCommits(limit int) ([]TimelineCommit, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.loadTimelineCommits(
		"log",
		"--all",
		"--date-order",
		fmt.Sprintf("-%d", limit),
		"--pretty=format:%H%x00%h%x00%cI%x00%an%x00%P%x00%D%x00%s",
	)
}

// LoadTimelineCommitsOnDay returns commits whose committer date falls on day (YYYY-MM-DD, local).
func (r *Repo) LoadTimelineCommitsOnDay(day string) ([]TimelineCommit, error) {
	day = strings.TrimSpace(day)
	if day == "" {
		day = time.Now().Format("2006-01-02")
	}
	parsed, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return nil, fmt.Errorf("dia inválido %q: %w", day, err)
	}
	until := parsed.AddDate(0, 0, 1).Format("2006-01-02")
	return r.loadTimelineCommits(
		"log",
		"--all",
		"--date-order",
		"--since="+day,
		"--until="+until,
		"--pretty=format:%H%x00%h%x00%cI%x00%an%x00%P%x00%D%x00%s",
	)
}

func (r *Repo) loadTimelineCommits(args ...string) ([]TimelineCommit, error) {
	out, err := r.run(args...)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var commits []TimelineCommit
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 7 {
			continue
		}
		at, err := time.Parse(time.RFC3339, parts[2])
		if err != nil {
			at, err = time.Parse("2006-01-02T15:04:05-07:00", parts[2])
			if err != nil {
				continue
			}
		}
		parents := strings.Fields(parts[4])
		refs := parseDecorations(parts[5])
		commits = append(commits, TimelineCommit{
			Hash:      parts[0],
			ShortHash: parts[1],
			At:        at,
			Author:    parts[3],
			Subject:   parts[6],
			IsMerge:   len(parents) > 1,
			Refs:      refs,
		})
	}
	return commits, nil
}

func parseDecorations(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var refs []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "HEAD -> ")
		part = strings.TrimPrefix(part, "tag: ")
		if part == "" || part == "HEAD" {
			continue
		}
		refs = append(refs, part)
	}
	return refs
}
