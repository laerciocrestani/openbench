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
		day = time.Now().In(time.Local).Format("2006-01-02")
	}
	start, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return nil, fmt.Errorf("dia inválido %q: %w", day, err)
	}
	end := start.AddDate(0, 0, 1)
	// Two Git quirks matter here:
	// 1) Bare YYYY-MM-DD for "today" is treated as now (not local midnight), so
	//    --since=<today> hides all of today's commits.
	// 2) Plain --since prunes the revwalk when a tip is older than the bound
	//    (backdated HEAD), skipping newer parents. --since-as-filter keeps walking.
	since := start.Format("2006-01-02 15:04:05")
	until := end.Format("2006-01-02 15:04:05")
	commits, err := r.loadTimelineCommits(
		"log",
		"--all",
		"--date-order",
		"--since-as-filter="+since,
		"--until="+until,
		"--pretty=format:%H%x00%h%x00%cI%x00%an%x00%P%x00%D%x00%s",
	)
	if err != nil {
		return nil, err
	}
	out := make([]TimelineCommit, 0, len(commits))
	for _, c := range commits {
		at := c.At.In(time.Local)
		if !at.Before(start) && at.Before(end) {
			out = append(out, c)
		}
	}
	return out, nil
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
