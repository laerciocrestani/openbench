package desktop

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	gitpkg "github.com/laerciocrestani/openbench/internal/git"
)

var (
	repoWatchDebounce     = 400 * time.Millisecond
	repoWatchPollInterval = 5 * time.Second
)

// RepoWatchCallback is invoked (debounced) when the working tree may have changed.
type RepoWatchCallback func()

// statusFingerprinter is overridable in tests.
var statusFingerprinter = gitpkg.StatusFingerprintAt

// RepoWatcher watches git metadata (not the full tree) and polls a cheap
// status fingerprint. Full-tree fsnotify is unsafe on macOS: kqueue opens one
// FD per file in each watched directory and large repos hit EMFILE.
type RepoWatcher struct {
	done   chan struct{}
	once   sync.Once
	paused atomic.Bool
}

// StartRepoWatcher watches git metadata under root and polls fingerprints.
// Caller must Close when done.
func StartRepoWatcher(root string, onChange RepoWatchCallback) (*RepoWatcher, error) {
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return nil, os.ErrInvalid
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrInvalid
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, p := range gitWatchPaths(root) {
		if err := watcher.Add(p); err != nil {
			continue
		}
	}

	rw := &RepoWatcher{done: make(chan struct{})}
	go rw.loop(root, watcher, onChange)
	return rw, nil
}

// Close stops the watcher.
func (rw *RepoWatcher) Close() {
	if rw == nil {
		return
	}
	rw.once.Do(func() { close(rw.done) })
}

// SetPaused suspends fingerprint polling and change notifications (app backgrounded).
func (rw *RepoWatcher) SetPaused(paused bool) {
	if rw == nil {
		return
	}
	rw.paused.Store(paused)
}

func (rw *RepoWatcher) loop(root string, watcher *fsnotify.Watcher, onChange RepoWatchCallback) {
	defer watcher.Close()

	var (
		timer  *time.Timer
		lastFP string
		haveFP bool
	)

	notify := func() {
		if onChange == nil || rw.paused.Load() {
			return
		}
		onChange()
	}

	resetDebounce := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(repoWatchDebounce, func() {
			select {
			case <-rw.done:
				return
			default:
				notify()
			}
		})
	}

	pollFingerprint := func() {
		fp, err := statusFingerprinter(root)
		if err != nil {
			return
		}
		if !haveFP {
			lastFP = fp
			haveFP = true
			return
		}
		if fp == lastFP {
			return
		}
		lastFP = fp
		resetDebounce()
	}

	// Seed fingerprint so the first tick does not spuriously refresh.
	if fp, err := statusFingerprinter(root); err == nil {
		lastFP = fp
		haveFP = true
	}

	ticker := time.NewTicker(repoWatchPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rw.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case <-ticker.C:
			if rw.paused.Load() {
				continue
			}
			pollFingerprint()

		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if rw.paused.Load() {
				continue
			}
			if event.Has(fsnotify.Chmod) && !event.Has(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) {
				continue
			}
			// After git metadata change, refresh fingerprint baseline on next poll.
			if fp, err := statusFingerprinter(root); err == nil {
				lastFP = fp
				haveFP = true
			}
			resetDebounce()
		}
	}
}

// gitWatchPaths returns a small set of git metadata paths (few FDs).
func gitWatchPaths(root string) []string {
	gitDir, err := resolveGitDir(root)
	if err != nil || gitDir == "" {
		return nil
	}

	candidates := []string{
		filepath.Join(gitDir, "HEAD"),
		filepath.Join(gitDir, "index"),
		filepath.Join(gitDir, "COMMIT_EDITMSG"),
		filepath.Join(gitDir, "packed-refs"),
		filepath.Join(gitDir, "refs", "heads"),
		filepath.Join(gitDir, "refs", "tags"),
	}

	out := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func resolveGitDir(root string) (string, error) {
	gitPath := filepath.Join(root, ".git")
	fi, err := os.Lstat(gitPath)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return gitPath, nil
	}
	f, err := os.Open(gitPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if len(line) < 8 {
			continue
		}
		if !strings.EqualFold(line[:7], "gitdir:") {
			continue
		}
		dir := strings.TrimSpace(line[7:])
		if dir == "" {
			continue
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, dir)
		}
		return filepath.Clean(dir), nil
	}
	return "", os.ErrNotExist
}
