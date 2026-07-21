package tui

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	tea "github.com/charmbracelet/bubbletea"
)

// Watch git metadata only (+ light poll). Recursive directory watches exhaust
// FDs on macOS kqueue (one FD per file in each watched dir).

type repoWatcher struct {
	done chan struct{}
	once sync.Once
}

func startRepoWatcher(p *tea.Program, root string) (*repoWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, path := range gitWatchPaths(root) {
		_ = watcher.Add(path)
	}

	rw := &repoWatcher{done: make(chan struct{})}
	go rw.loop(p, watcher)
	return rw, nil
}

func (rw *repoWatcher) Close() {
	rw.once.Do(func() { close(rw.done) })
}

func (rw *repoWatcher) loop(p *tea.Program, watcher *fsnotify.Watcher) {
	defer watcher.Close()

	var timer *time.Timer

	resetDebounce := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(watchDebounce, func() {
			select {
			case <-rw.done:
				return
			default:
				p.Send(watchRefreshMsg{})
			}
		})
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rw.done:
			if timer != nil {
				timer.Stop()
			}
			return

		case <-ticker.C:
			resetDebounce()

		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Chmod) && !event.Has(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) {
				continue
			}
			resetDebounce()
		}
	}
}

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
	}
	out := make([]string, 0, len(candidates))
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
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
		if len(line) < 8 || !strings.EqualFold(line[:7], "gitdir:") {
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

func repoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
