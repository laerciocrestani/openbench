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
	out, err := r.run(
		"log",
		"--all",
		"--date-order",
		fmt.Sprintf("-%d", limit),
		"--pretty=format:%H%x00%h%x00%cI%x00%an%x00%P%x00%D%x00%s",
	)
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
