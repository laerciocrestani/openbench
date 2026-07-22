package main

import (
	"strings"

	"github.com/laerciocrestani/openbench/internal/desktop"
)

func (s *AppService) startRepoWatch(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	watchPath := path
	w, err := desktop.StartRepoWatcher(watchPath, func() {
		s.emitDashboardRefresh(watchPath)
	})
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !desktop.SamePath(s.projectPath, watchPath) {
		w.Close()
		return
	}
	if s.repoWatch != nil {
		s.repoWatch.Close()
	}
	s.repoWatch = w
}

func (s *AppService) stopRepoWatchLocked() {
	if s.repoWatch == nil {
		return
	}
	s.repoWatch.Close()
	s.repoWatch = nil
}

// emitDashboardRefresh reloads the dashboard for path and pushes it to the UI.
// Does not restart the repo watcher (path must still be the active project).
func (s *AppService) emitDashboardRefresh(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	s.mu.RLock()
	cur := s.projectPath
	appRef := s.app
	hub := s.hub
	s.mu.RUnlock()
	if !desktop.SamePath(cur, path) {
		return
	}

	dash, err := desktop.LoadDashboard(path)
	if err != nil || dash == nil {
		return
	}
	if appRef != nil {
		// Must emit value type: RegisterEvent[desktop.Dashboard] rejects *Dashboard.
		appRef.Event.Emit("project:dashboard", *dash)
	}
	if hub != nil {
		_ = hub.RefreshNow()
	}
	s.refreshTray()
}
