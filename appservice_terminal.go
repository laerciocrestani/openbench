package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/laerciocrestani/openbench/internal/desktop"
)

// TerminalStart opens (or restarts) an interactive shell session.
// Uses the open project root when available; otherwise the user home directory.
func (s *AppService) TerminalStart(cols, rows uint16) error {
	cwd := strings.TrimSpace(s.currentPath())
	if cwd == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return fmt.Errorf("não foi possível resolver o diretório home do usuário")
		}
		cwd = home
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.term != nil && s.term.Cwd() == cwd && s.term.Label() == "host" {
		if cols > 0 || rows > 0 {
			_ = s.term.Resize(cols, rows)
		}
		return nil
	}

	if s.term != nil {
		s.term.Close()
		s.term = nil
	}

	app := s.app
	emit := func(event, data string) {
		if app == nil {
			return
		}
		app.Event.Emit(event, data)
	}

	sess, err := desktop.NewTerminalSession(cwd, cols, rows, emit)
	if err != nil {
		return err
	}
	s.term = sess
	return nil
}

// DockerShellStart opens an interactive shell inside a compose service (PTY).
// When presetID is set and interactive, runs that preset command instead of sh.
func (s *AppService) DockerShellStart(service string, cols, rows uint16, presetID string) error {
	cwd := s.currentPath()
	if cwd == "" {
		return fmt.Errorf("abra um projeto para usar o terminal")
	}
	compose, argv, err := desktop.ResolveDockerShellCommand(cwd, service, presetID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.term != nil {
		s.term.Close()
		s.term = nil
	}

	app := s.app
	emit := func(event, data string) {
		if app == nil {
			return
		}
		app.Event.Emit(event, data)
	}

	sess, err := desktop.NewDockerShellSession(cwd, compose, service, argv, cols, rows, emit)
	if err != nil && (len(argv) == 1 && argv[0] == "sh") {
		// Fallback to bash when plain sh fails to start.
		sess, err = desktop.NewDockerShellSession(cwd, compose, service, []string{"bash"}, cols, rows, emit)
	}
	if err != nil {
		return err
	}
	s.term = sess
	return nil
}

// TerminalWrite sends keystrokes / paste to the active PTY.
func (s *AppService) TerminalWrite(data string) error {
	s.mu.RLock()
	term := s.term
	s.mu.RUnlock()
	if term == nil {
		return fmt.Errorf("terminal não iniciado")
	}
	return term.Write(data)
}

// TerminalResize updates columns/rows for the active PTY.
func (s *AppService) TerminalResize(cols, rows uint16) error {
	s.mu.RLock()
	term := s.term
	s.mu.RUnlock()
	if term == nil {
		return nil
	}
	return term.Resize(cols, rows)
}

// TerminalStop kills the active shell session.
func (s *AppService) TerminalStop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.term != nil {
		s.term.Close()
		s.term = nil
	}
}

// TerminalRestart forces a new host shell (project root or user home).
func (s *AppService) TerminalRestart(cols, rows uint16) error {
	s.mu.Lock()
	if s.term != nil {
		s.term.Close()
		s.term = nil
	}
	s.mu.Unlock()
	return s.TerminalStart(cols, rows)
}

// TerminalLabel returns the active session label (host / docker:…).
func (s *AppService) TerminalLabel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.term == nil {
		return ""
	}
	return strings.TrimSpace(s.term.Label())
}

func (s *AppService) stopTerminalLocked() {
	if s.term != nil {
		s.term.Close()
		s.term = nil
	}
}
