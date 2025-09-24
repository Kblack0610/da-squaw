package coreadapter

import (
	"context"

	"claude-squad/interface/facade"
	"claude-squad/services/session"
)

// sessionViewerAdapter adapts the orchestrator to the SessionViewer facade
type sessionViewerAdapter struct {
	orchestrator session.SessionOrchestrator
}

// NewSessionViewer creates a new SessionViewer facade
func NewSessionViewer(orchestrator session.SessionOrchestrator) facade.SessionViewer {
	return &sessionViewerAdapter{
		orchestrator: orchestrator,
	}
}

func (s *sessionViewerAdapter) GetPreview(ctx context.Context, id string) (string, error) {
	return s.orchestrator.GetOutput(ctx, id)
}

func (s *sessionViewerAdapter) GetFullHistory(ctx context.Context, id string) (string, error) {
	// In a real implementation, this might use a different method or flag
	// For now, just return the same as preview
	return s.orchestrator.GetOutput(ctx, id)
}

func (s *sessionViewerAdapter) HasUpdated(ctx context.Context, id string, lastPreview string) (bool, error) {
	current, err := s.orchestrator.GetOutput(ctx, id)
	if err != nil {
		return false, err
	}
	return current != lastPreview, nil
}