package tui

import (
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

var skipWatchDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".cache":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
}

type repoWatcher struct {
	done chan struct{}
	once sync.Once
}

func startRepoWatcher(p *tea.Program, root string) (*repoWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	rw := &repoWatcher{done: make(chan struct{})}

	if err := addWatchTree(watcher, root); err != nil {
		watcher.Close()
		return nil, err
	}

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

	for {
		select {
		case <-rw.done:
			if timer != nil {
				timer.Stop()
			}
			return

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
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() && !shouldSkipWatchDir(info.Name()) {
					_ = addWatchTree(watcher, event.Name)
				}
			}
			resetDebounce()
		}
	}
}

func addWatchTree(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && shouldSkipWatchDir(d.Name()) {
			return filepath.SkipDir
		}
		if err := watcher.Add(path); err != nil {
			return nil
		}
		return nil
	})
}

func shouldSkipWatchDir(name string) bool {
	return skipWatchDirs[name]
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
