package git

import (
	"fmt"
	"strconv"
	"strings"
)

type BranchInfo struct {
	Name     string
	Current  bool
	Upstream string
	Ahead    int
	Behind   int
}

type StashInfo struct {
	Ref     string
	Branch  string
	Message string
}

type FileChange struct {
	Path       string
	Status     string
	Insertions int
	Deletions  int
}

type Overview struct {
	Root               string
	Branch             string
	HeadHash           string
	HeadFullHash       string
	Detached           bool
	RemoteURL          string
	Upstream           string
	Ahead              int
	Behind             int
	BaseBranch         string
	CommitsAheadOfBase int
	HasBranchDiff      bool // true when base...HEAD has at least one file
	BaseBehind         int  // local base behind origin/<base>
	Staged             int
	Modified           int
	Untracked          int
	Branches           []BranchInfo
	RecentCommits      []string
	Stashes            []StashInfo
	FileChanges        []FileChange
}

func (r *Repo) Overview(baseBranch string) (*Overview, error) {
	o := &Overview{BaseBranch: baseBranch}

	root, err := r.run("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	o.Root = root

	branch, err := r.CurrentBranch()
	if err != nil {
		return nil, err
	}
	o.Branch = branch
	o.Detached = branch == "HEAD"

	if hash, err := r.run("rev-parse", "--short", "HEAD"); err == nil {
		o.HeadHash = hash
	}
	if hash, err := r.run("rev-parse", "HEAD"); err == nil {
		o.HeadFullHash = hash
	}

	if url, err := r.run("remote", "get-url", "origin"); err == nil {
		o.RemoteURL = url
	}

	if upstream, err := r.run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil {
		o.Upstream = upstream
		if ahead, behind, err := r.aheadBehind(upstream); err == nil {
			o.Ahead = ahead
			o.Behind = behind
		}
	}

	if baseBranch != "" {
		if resolved, err := r.ResolveBase(baseBranch); err == nil {
			if count, err := r.run("rev-list", "--count", fmt.Sprintf("%s..HEAD", resolved)); err == nil {
				o.CommitsAheadOfBase, _ = strconv.Atoi(count)
			}
			if names, err := r.DiffBranchNames(resolved); err == nil {
				o.HasBranchDiff = strings.TrimSpace(names) != ""
			}
		}
		if n, err := r.BaseBehindOrigin(baseBranch); err == nil {
			o.BaseBehind = n
		}
	}

	staged, modified, untracked, err := r.worktreeCounts()
	if err != nil {
		return nil, err
	}
	o.Staged = staged
	o.Modified = modified
	o.Untracked = untracked

	branches, err := r.listBranches()
	if err != nil {
		return nil, err
	}
	o.Branches = branches

	log, err := r.run("log", "-3", "--oneline", "--decorate")
	if err == nil && log != "" {
		o.RecentCommits = strings.Split(log, "\n")
	}

	if stashes, err := r.listStashes(); err == nil {
		o.Stashes = stashes
	}

	if changes, err := r.fileChanges(); err == nil {
		o.FileChanges = changes
	}

	return o, nil
}

func (r *Repo) listStashes() ([]StashInfo, error) {
	out, err := r.run("stash", "list")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var stashes []StashInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		info := StashInfo{}
		colon := strings.Index(line, ": ")
		if colon == -1 {
			info.Ref = line
			stashes = append(stashes, info)
			continue
		}

		info.Ref = line[:colon]
		rest := line[colon+2:]

		if strings.HasPrefix(rest, "WIP on ") {
			rest = strings.TrimPrefix(rest, "WIP on ")
		} else if strings.HasPrefix(rest, "On ") {
			rest = strings.TrimPrefix(rest, "On ")
		}

		if idx := strings.Index(rest, ": "); idx != -1 {
			info.Branch = rest[:idx]
			info.Message = rest[idx+2:]
		} else {
			info.Message = rest
		}

		stashes = append(stashes, info)
	}

	return stashes, nil
}

func (r *Repo) fileChanges() ([]FileChange, error) {
	porcelain, err := r.run("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	stats := map[string][2]int{}
	for _, source := range []struct {
		staged bool
	}{
		{staged: false},
		{staged: true},
	} {
		args := []string{"diff", "--numstat"}
		if source.staged {
			args = []string{"diff", "--cached", "--numstat"}
		}
		out, err := r.run(args...)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) < 3 {
				continue
			}
			add, _ := strconv.Atoi(parts[0])
			del, _ := strconv.Atoi(parts[1])
			path := parts[2]
			cur := stats[path]
			cur[0] += add
			cur[1] += del
			stats[path] = cur
		}
	}

	var changes []FileChange
	seen := map[string]bool{}

	for _, line := range strings.Split(porcelain, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		path, status := parsePorcelainLine(line)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true

		change := FileChange{
			Path:   path,
			Status: status,
		}
		if stat, ok := stats[path]; ok {
			change.Insertions = stat[0]
			change.Deletions = stat[1]
		}
		changes = append(changes, change)
	}

	return changes, nil
}

func parsePorcelainLine(line string) (path, status string) {
	if strings.HasPrefix(line, "??") {
		return strings.TrimSpace(line[2:]), "untracked"
	}
	if len(line) < 3 {
		return "", ""
	}

	index := line[0]
	worktree := line[1]
	raw := strings.TrimSpace(line[2:])

	if strings.Contains(raw, " -> ") {
		parts := strings.Split(raw, " -> ")
		raw = parts[len(parts)-1]
		status = "renamed"
	}

	switch {
	case index == 'D' || worktree == 'D':
		status = "deleted"
	case index == 'A' && worktree == ' ':
		status = "new"
	case index == 'A' || index == 'M' || index == 'R' || index == 'C':
		if worktree == 'M' || worktree == 'D' {
			status = "staged+modified"
		} else if status == "" {
			status = "staged"
		}
	case worktree == 'M' || worktree == 'D':
		status = "modified"
	default:
		if status == "" {
			status = "changed"
		}
	}

	return raw, status
}

func (r *Repo) aheadBehind(upstream string) (ahead, behind int, err error) {
	out, err := r.run("rev-list", "--left-right", "--count", fmt.Sprintf("HEAD...%s", upstream))
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("formato inesperado: %q", out)
	}
	ahead, _ = strconv.Atoi(parts[0])
	behind, _ = strconv.Atoi(parts[1])
	return ahead, behind, nil
}

func (r *Repo) worktreeCounts() (staged, modified, untracked int, err error) {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return 0, 0, 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "??") {
			untracked++
			continue
		}
		if len(line) < 2 {
			continue
		}
		index := line[0]
		worktree := line[1]
		if index != ' ' {
			staged++
		}
		if worktree != ' ' {
			modified++
		}
	}
	return staged, modified, untracked, nil
}

func (r *Repo) listBranches() ([]BranchInfo, error) {
	out, err := r.run("branch", "-vv", "--color=never")
	if err != nil {
		return nil, err
	}

	var branches []BranchInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		current := strings.HasPrefix(line, "*")
		line = strings.TrimPrefix(strings.TrimPrefix(line, "*"), " ")
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		info := BranchInfo{
			Name:    parts[0],
			Current: current,
		}

		for i, part := range parts {
			if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
				tracking := strings.Trim(part, "[]")
				info.Upstream = tracking
				if i+1 < len(parts) && (strings.HasPrefix(parts[i+1], "ahead") || strings.HasPrefix(parts[i+1], "behind")) {
					info.Ahead, info.Behind = parseTrackingCounts(parts[i+1:])
				}
				break
			}
		}

		branches = append(branches, info)
	}

	return branches, nil
}

func parseTrackingCounts(parts []string) (ahead, behind int) {
	for _, part := range parts {
		if strings.HasPrefix(part, "ahead") {
			ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead"))
		}
		if strings.HasPrefix(part, "behind") {
			behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind"))
		}
	}
	return ahead, behind
}

func (o Overview) IsDirty() bool {
	return o.Staged > 0 || o.Modified > 0 || o.Untracked > 0
}

func (f FileChange) StatsLabel() string {
	if f.Insertions == 0 && f.Deletions == 0 {
		return ""
	}
	return fmt.Sprintf("+%d -%d", f.Insertions, f.Deletions)
}
