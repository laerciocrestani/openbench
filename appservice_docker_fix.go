package main

import (
	"context"

	"github.com/laerciocrestani/openbench/internal/desktop"
)

// PlanDockerFix asks the AI for a Docker remediation plan from a failed action snapshot.
func (s *AppService) PlanDockerFix(failure desktop.DockerFixFailureView) (*desktop.DockerFixPlanView, error) {
	return desktop.PlanDockerFix(context.Background(), s.currentPath(), failure)
}

// BeginDockerFix prepares a step-by-step Docker Fix session with the user's selection.
func (s *AppService) BeginDockerFix(enabledStepIDs, enabledFilePaths []string) (*desktop.DockerFixPlanView, error) {
	path := s.currentPath()
	plan, sess, err := desktop.BeginDockerFix(path, enabledStepIDs, enabledFilePaths)
	s.mu.Lock()
	if err != nil {
		s.dockerFixSession = nil
		s.mu.Unlock()
		return plan, err
	}
	s.dockerFixSession = sess
	s.mu.Unlock()
	return plan, nil
}

// AdvanceDockerFix runs the next step of the active Docker Fix session.
func (s *AppService) AdvanceDockerFix() (*desktop.DockerFixAdvanceView, error) {
	s.mu.Lock()
	sess := s.dockerFixSession
	s.mu.Unlock()
	out, err := desktop.AdvanceDockerFixSession(sess)
	if err != nil {
		return nil, err
	}
	if out != nil && (out.Done || !out.OK) {
		s.mu.Lock()
		s.dockerFixSession = nil
		s.mu.Unlock()
	}
	return out, nil
}
