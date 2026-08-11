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
// Avoids hub.RefreshNow() — that used to re-scan every pinned project on each
// file save and was a major CPU amplifier alongside the watcher.
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

	dash, err := desktop.LoadGitStatus(path)
	if err != nil || dash == nil {
		return
	}
	if appRef != nil {
		// Must emit value type: RegisterEvent[desktop.Dashboard] rejects *Dashboard.
		appRef.Event.Emit("project:dashboard", *dash)
	}
	// Light status for the active project only (tabs / tray), not a full hub poll.
	if hub != nil && appRef != nil {
		st := desktop.LoadProjectStatus(path, false)
		st.Active = true
		hub.CacheStatus(st)
		appRef.Event.Emit("project:status", st)
	}
	s.refreshTray()
}
