package coreadapter

import (
	"context"
	"strings"

	"claude-squad/interface/facade"
	"claude-squad/services/session"
)

// sessionInteractorAdapter adapts the orchestrator to the SessionInteractor facade
type sessionInteractorAdapter struct {
	orchestrator session.SessionOrchestrator
}

// NewSessionInteractor creates a new SessionInteractor facade
func NewSessionInteractor(orchestrator session.SessionOrchestrator) facade.SessionInteractor {
	return &sessionInteractorAdapter{
		orchestrator: orchestrator,
	}
}

func (s *sessionInteractorAdapter) AttachSession(ctx context.Context, id string) error {
	return s.orchestrator.AttachSession(ctx, id)
}

func (s *sessionInteractorAdapter) SendKeys(ctx context.Context, id string, keys string) error {
	return s.orchestrator.SendInput(ctx, id, keys)
}

func (s *sessionInteractorAdapter) SendPrompt(ctx context.Context, id string, prompt string) error {
	return s.orchestrator.SendInput(ctx, id, prompt+"\n")
}

func (s *sessionInteractorAdapter) HasPrompt(ctx context.Context, id string) (bool, error) {
	output, err := s.orchestrator.GetOutput(ctx, id)
	if err != nil {
		return false, err
	}

	// Simple heuristic for detecting prompts
	hasPrompt := strings.Contains(output, "[Y/n]") ||
		strings.Contains(output, "(y/N)") ||
		strings.Contains(output, "Continue?") ||
		strings.Contains(output, "Press Enter") ||
		strings.HasSuffix(strings.TrimSpace(output), ">")

	return hasPrompt, nil
}