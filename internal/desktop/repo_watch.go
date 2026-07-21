package desktop

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const repoWatchDebounce = 400 * time.Millisecond

var skipWatchDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".cache":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".next":        true,
	".turbo":       true,
	"coverage":     true,
}

// RepoWatchCallback is invoked (debounced) when the working tree may have changed.
type RepoWatchCallback func()

// RepoWatcher watches the project tree for filesystem changes.
type RepoWatcher struct {
	done chan struct{}
	once sync.Once
}

// StartRepoWatcher watches root and calls onChange after debounce.
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

	rw := &RepoWatcher{done: make(chan struct{})}
	if err := addWatchTree(watcher, root); err != nil {
		watcher.Close()
		return nil, err
	}

	go rw.loop(watcher, onChange)
	return rw, nil
}

// Close stops the watcher.
func (rw *RepoWatcher) Close() {
	if rw == nil {
		return
	}
	rw.once.Do(func() { close(rw.done) })
}

func (rw *RepoWatcher) loop(watcher *fsnotify.Watcher, onChange RepoWatchCallback) {
	defer watcher.Close()

	var timer *time.Timer
	resetDebounce := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(repoWatchDebounce, func() {
			select {
			case <-rw.done:
				return
			default:
				if onChange != nil {
					onChange()
				}
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
		_ = watcher.Add(path)
		return nil
	})
}

func shouldSkipWatchDir(name string) bool {
	return skipWatchDirs[name]
}
