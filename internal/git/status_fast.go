package git

import (
	"fmt"
	"strconv"
	"strings"
)

// ShellInfo is the cheapest git identity for opening a project instantly.
type ShellInfo struct {
	Root     string
	Branch   string
	HeadHash string
	Detached bool
}

// Shell loads only toplevel + branch + short HEAD (few git calls).
// Supports unborn HEAD (fresh `git init` with no commits yet).
func (r *Repo) Shell() (*ShellInfo, error) {
	root, err := r.run("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	branch, err := r.CurrentBranch()
	if err != nil {
		// Unborn HEAD: rev-parse HEAD fails; symbolic-ref still names the branch.
		branch, err = r.run("symbolic-ref", "--short", "HEAD")
		if err != nil {
			branch = "main"
		}
	}
	info := &ShellInfo{
		Root:     root,
		Branch:   branch,
		Detached: branch == "HEAD",
	}
	if hash, err := r.run("rev-parse", "--short", "HEAD"); err == nil {
		info.HeadHash = hash
	}
	return info, nil
}

// StatusSnapshot is branch tracking + dirty counts + optional file list from one porcelain pass.
type StatusSnapshot struct {
	Branch             string
	Upstream           string
	Ahead              int
	Behind             int
	Staged             int
	Modified           int
	Untracked          int
	FileChanges        []FileChange
	CommitsAheadOfBase int
	// Unpublished is commits on HEAD when there is no upstream (first-push candidates).
	Unpublished   int
	HasBranchDiff bool
	BaseBehind    int
	BaseBranch    string
	HeadHash      string
	HeadFullHash  string
	RemoteURL     string
	Detached      bool
	Root          string
}

// LoadStatus builds a dashboard-ready git status without branch lists / numstat / hygiene.
// Uses one porcelain --branch pass for dirty + ahead/behind when possible.
func (r *Repo) LoadStatus(baseBranch string, withFiles bool) (*StatusSnapshot, error) {
	root, err := r.run("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	s := &StatusSnapshot{Root: root, BaseBranch: baseBranch}

	branch, err := r.CurrentBranch()
	if err != nil {
		branch, err = r.run("symbolic-ref", "--short", "HEAD")
		if err != nil {
			branch = "main"
		}
	}
	s.Branch = branch
	s.Detached = branch == "HEAD"

	if hash, err := r.run("rev-parse", "--short", "HEAD"); err == nil {
		s.HeadHash = hash
	}
	if hash, err := r.run("rev-parse", "HEAD"); err == nil {
		s.HeadFullHash = hash
	}
	if url, err := r.run("remote", "get-url", "origin"); err == nil {
		s.RemoteURL = url
	}

	// Single porcelain --branch: "## branch...upstream [ahead N, behind M]" + file lines.
	// Works on unborn HEAD (fresh git init).
	out, err := r.runRaw("status", "--porcelain=v1", "--branch")
	if err != nil {
		return nil, err
	}
	s.Staged, s.Modified, s.Untracked, s.FileChanges, s.Branch, s.Upstream, s.Ahead, s.Behind =
		parsePorcelainBranch(out, s.Branch)

	if baseBranch != "" {
		if resolved, err := r.ResolveBase(baseBranch); err == nil {
			if count, err := r.run("rev-list", "--count", fmt.Sprintf("%s..HEAD", resolved)); err == nil {
				s.CommitsAheadOfBase, _ = strconv.Atoi(count)
			}
			// Cheap approximation: commits ahead ⇒ likely has branch diff (avoids
			// git diff --name-only base...HEAD which is slow on large histories).
			s.HasBranchDiff = s.CommitsAheadOfBase > 0
		}
		if n, err := r.BaseBehindOrigin(baseBranch); err == nil {
			s.BaseBehind = n
		}
	}

	// First push: no upstream yet, but HEAD has commits (common after git init + remote).
	if strings.TrimSpace(s.Upstream) == "" && s.HeadHash != "" {
		if count, err := r.run("rev-list", "--count", "HEAD"); err == nil {
			s.Unpublished, _ = strconv.Atoi(count)
		}
	}

	if !withFiles {
		s.FileChanges = nil
	}
	return s, nil
}

func parsePorcelainBranch(out, fallbackBranch string) (
	staged, modified, untracked int,
	changes []FileChange,
	branch, upstream string,
	ahead, behind int,
) {
	branch = fallbackBranch
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			branch, upstream, ahead, behind = parseBranchHeader(line[3:])
			continue
		}
		if strings.HasPrefix(line, "??") {
			untracked++
			path, status := parsePorcelainLine(line)
			if path != "" && !seen[path] {
				seen[path] = true
				changes = append(changes, FileChange{Path: path, Status: status})
			}
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
		path, status := parsePorcelainLine(line)
		if path != "" && !seen[path] {
			seen[path] = true
			changes = append(changes, FileChange{Path: path, Status: status})
		}
	}
	return staged, modified, untracked, changes, branch, upstream, ahead, behind
}

func parseBranchHeader(rest string) (branch, upstream string, ahead, behind int) {
	rest = strings.TrimSpace(rest)
	// Formats:
	//   main
	//   HEAD (no branch)
	//   main...origin/main
	//   main...origin/main [ahead 2]
	//   main...origin/main [behind 1]
	//   main...origin/main [ahead 2, behind 1]
	if i := strings.Index(rest, " ["); i >= 0 {
		tracking := rest[i+2:]
		rest = rest[:i]
		if j := strings.Index(tracking, "]"); j >= 0 {
			tracking = tracking[:j]
		}
		ahead, behind = parseTrackingCounts(strings.Fields(strings.ReplaceAll(tracking, ",", " ")))
		// parseTrackingCounts expects "ahead" "2" as separate? Looking at existing:
		// parseTrackingCounts checks strings.HasPrefix(part, "ahead") then TrimPrefix "ahead"
		// But fields of "ahead 2 behind 1" are ["ahead","2","behind","1"] - ahead prefix on "ahead" gives ""
		// Looking at original parseTrackingCounts - it expects "ahead2" glued? From branch -vv:
		// [origin/main: ahead 1, behind 2] - the fields after [ are weird.
		// Let me check original listBranches parsing...
		ahead, behind = parseAheadBehindBracket(tracking)
	}
	if strings.Contains(rest, "...") {
		parts := strings.SplitN(rest, "...", 2)
		branch = strings.TrimSpace(parts[0])
		upstream = strings.TrimSpace(parts[1])
	} else {
		branch = strings.TrimSpace(rest)
	}
	if branch == "HEAD (no branch)" {
		branch = "HEAD"
	}
	return branch, upstream, ahead, behind
}

func parseAheadBehindBracket(s string) (ahead, behind int) {
	// "ahead 2, behind 1" or "ahead 2" or "behind 1"
	s = strings.ReplaceAll(s, ",", " ")
	fields := strings.Fields(s)
	for i := 0; i < len(fields); i++ {
		switch fields[i] {
		case "ahead":
			if i+1 < len(fields) {
				ahead, _ = strconv.Atoi(fields[i+1])
				i++
			}
		case "behind":
			if i+1 < len(fields) {
				behind, _ = strconv.Atoi(fields[i+1])
				i++
			}
		}
	}
	return ahead, behind
}
