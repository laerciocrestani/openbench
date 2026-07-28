package git

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// GitHub rejects blobs over 100 MiB (without LFS).
const GitHubMaxFileBytes = 100 * 1024 * 1024

// LargeFileHit is one blob above the push limit in an unpushed range.
type LargeFileHit struct {
	Path string
	Size int64
}

// UnpushedBlockers lists reasons a push of local-only commits will fail or is unsafe.
type UnpushedBlockers struct {
	LargeFiles []LargeFileHit
	JunkPaths  []string // concrete paths or roots to git rm --cached
}

// HasBlockers reports whether cleanup is warranted before push.
func (b *UnpushedBlockers) HasBlockers() bool {
	return b != nil && (len(b.LargeFiles) > 0 || len(b.JunkPaths) > 0)
}

// CachedRoots returns unique paths suitable for `git rm --cached` (roots collapsed).
func (b *UnpushedBlockers) CachedRoots() []string {
	if b == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(filepath.ToSlash(p))
		p = strings.TrimPrefix(p, "./")
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, hit := range b.LargeFiles {
		add(hit.Path)
	}
	for _, p := range b.JunkPaths {
		add(collapseJunkRoot(p))
	}
	return out
}

func collapseJunkRoot(path string) string {
	norm := filepath.ToSlash(path)
	lower := strings.ToLower(norm)
	for _, root := range []string{"node_modules", "vendor", "dist", "build", ".next", "coverage"} {
		if lower == root || strings.HasPrefix(lower, root+"/") {
			return root
		}
	}
	return norm
}

// ScanUnpushedBlockers inspects commits in baseRef..headRef for oversized blobs and junk paths.
func (r *Repo) ScanUnpushedBlockers(baseRef, headRef string) (*UnpushedBlockers, error) {
	baseRef = strings.TrimSpace(baseRef)
	headRef = strings.TrimSpace(headRef)
	if baseRef == "" || headRef == "" {
		return nil, fmt.Errorf("refs vazias")
	}
	if _, err := r.run("rev-parse", "--verify", baseRef); err != nil {
		return nil, err
	}
	if _, err := r.run("rev-parse", "--verify", headRef); err != nil {
		return nil, err
	}

	countOut, err := r.run("rev-list", "--count", fmt.Sprintf("%s..%s", baseRef, headRef))
	if err != nil {
		return nil, err
	}
	n, _ := strconv.Atoi(strings.TrimSpace(countOut))
	if n == 0 {
		return &UnpushedBlockers{}, nil
	}

	out := &UnpushedBlockers{}

	objects, err := r.run("rev-list", "--objects", fmt.Sprintf("%s..%s", baseRef, headRef))
	if err != nil {
		return nil, err
	}
	var oids []string
	oidPath := map[string]string{}
	for _, line := range splitLines(objects) {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		oid := parts[0]
		oids = append(oids, oid)
		if len(parts) > 1 {
			oidPath[oid] = strings.TrimSpace(parts[1])
		}
	}
	if len(oids) > 0 {
		sizes, err := r.batchBlobSizes(oids)
		if err == nil {
			seenLarge := map[string]bool{}
			for oid, size := range sizes {
				if size <= GitHubMaxFileBytes {
					continue
				}
				path := oidPath[oid]
				if path == "" || seenLarge[path] {
					continue
				}
				seenLarge[path] = true
				out.LargeFiles = append(out.LargeFiles, LargeFileHit{Path: path, Size: size})
			}
		}
	}

	names, err := r.run("diff", "--name-only", fmt.Sprintf("%s..%s", baseRef, headRef))
	if err != nil {
		return out, nil
	}
	vendorOnBase := r.pathExistsInTree(baseRef, "vendor")
	seenJunk := map[string]bool{}
	for _, name := range splitLines(names) {
		name = filepath.ToSlash(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		if !isUnpushedJunkPath(name, vendorOnBase) {
			continue
		}
		root := collapseJunkRoot(name)
		if seenJunk[root] {
			continue
		}
		seenJunk[root] = true
		out.JunkPaths = append(out.JunkPaths, root)
	}

	return out, nil
}

func (r *Repo) pathExistsInTree(ref, path string) bool {
	_, err := r.run("cat-file", "-e", ref+":"+path)
	return err == nil
}

func (r *Repo) batchBlobSizes(oids []string) (map[string]int64, error) {
	out := map[string]int64{}
	const chunk = 400
	for i := 0; i < len(oids); i += chunk {
		end := i + chunk
		if end > len(oids) {
			end = len(oids)
		}
		var b strings.Builder
		for _, oid := range oids[i:end] {
			b.WriteString(oid)
			b.WriteByte('\n')
		}
		text, err := r.runWithStdin(b.String(), "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
		if err != nil {
			return out, err
		}
		for _, line := range splitLines(text) {
			parts := strings.Fields(line)
			if len(parts) < 3 || parts[1] != "blob" {
				continue
			}
			size, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				continue
			}
			out[parts[0]] = size
		}
	}
	return out, nil
}

func isUnpushedJunkPath(path string, vendorTrackedOnBase bool) bool {
	norm := filepath.ToSlash(path)
	lower := strings.ToLower(norm)
	base := filepath.Base(lower)

	if strings.HasPrefix(lower, "node_modules/") || lower == "node_modules" {
		return true
	}
	if !vendorTrackedOnBase && (strings.HasPrefix(lower, "vendor/") || lower == "vendor") {
		return true
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	if strings.HasSuffix(lower, ".dmg") || strings.HasSuffix(lower, ".iso") || strings.HasSuffix(lower, ".pkg") {
		return true
	}
	if base == ".ds_store" || base == "composer.phar" || base == "thumbs.db" {
		return true
	}
	if strings.HasPrefix(lower, "storage/logs/") ||
		strings.HasPrefix(lower, "storage/framework/sessions/") ||
		strings.HasPrefix(lower, "storage/framework/views/") {
		return true
	}
	return false
}

// SoftReset moves HEAD to ref, keeping all changes staged.
func (r *Repo) SoftReset(ref string) error {
	if err := r.EnsureWritableIndex(); err != nil {
		return err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("ref vazia")
	}
	_, err := r.run("reset", "--soft", ref)
	return err
}

// MixedReset moves HEAD to ref and resets the index to match, preserving the working tree.
// This is the safe cleanup for unpushed junk history: commits disappear, files stay on disk unstaged.
func (r *Repo) MixedReset(ref string) error {
	if err := r.EnsureWritableIndex(); err != nil {
		return err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("ref vazia")
	}
	_, err := r.run("reset", "--mixed", ref)
	return err
}

// RmCached removes paths from the index only (--cached). Missing paths are ignored.
func (r *Repo) RmCached(paths []string) error {
	if err := r.EnsureWritableIndex(); err != nil {
		return err
	}
	clean := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	if len(clean) == 0 {
		return nil
	}
	args := append([]string{"rm", "-r", "--cached", "--ignore-unmatch", "--"}, clean...)
	_, err := r.run(args...)
	return err
}
