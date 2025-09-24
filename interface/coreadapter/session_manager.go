package coreadapter

import (
	"context"
	"path/filepath"

	"claude-squad/interface/facade"
	"claude-squad/services/session"
	"claude-squad/services/types"
)

// sessionManagerAdapter adapts the orchestrator to the SessionManager facade
type sessionManagerAdapter struct {
	orchestrator session.SessionOrchestrator
}

// NewSessionManager creates a new SessionManager facade
func NewSessionManager(orchestrator session.SessionOrchestrator) facade.SessionManager {
	return &sessionManagerAdapter{
		orchestrator: orchestrator,
	}
}

func (s *sessionManagerAdapter) ListSessions(ctx context.Context) ([]facade.SessionInfo, error) {
	sessions, err := s.orchestrator.ListSessions(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]facade.SessionInfo, len(sessions))
	for i, sess := range sessions {
		result[i] = toFacadeInfo(sess)
	}
	return result, nil
}

func (s *sessionManagerAdapter) CreateSession(ctx context.Context, title, path, program string) (*facade.SessionInfo, error) {
	req := types.CreateSessionRequest{
		Title:   title,
		Path:    path,
		Program: program,
		Height:  24,
		Width:   80,
	}

	sess, err := s.orchestrator.CreateSession(ctx, req)
	if err != nil {
		return nil, err
	}

	info := toFacadeInfo(sess)
	return &info, nil
}

func (s *sessionManagerAdapter) StartSession(ctx context.Context, id string) error {
	return s.orchestrator.StartSession(ctx, id)
}

func (s *sessionManagerAdapter) StopSession(ctx context.Context, id string) error {
	return s.orchestrator.StopSession(ctx, id)
}

func (s *sessionManagerAdapter) PauseSession(ctx context.Context, id string) error {
	return s.orchestrator.PauseSession(ctx, id)
}

func (s *sessionManagerAdapter) ResumeSession(ctx context.Context, id string) error {
	return s.orchestrator.ResumeSession(ctx, id)
}

func (s *sessionManagerAdapter) GetSession(ctx context.Context, id string) (*facade.SessionInfo, error) {
	sess, err := s.orchestrator.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	info := toFacadeInfo(sess)
	return &info, nil
}

func (s *sessionManagerAdapter) UpdateTitle(ctx context.Context, id string, title string) error {
	sess, err := s.orchestrator.GetSession(ctx, id)
	if err != nil {
		return err
	}
	sess.Title = title
	return s.orchestrator.UpdateSessionStatus(ctx, id, sess.Status)
}

// Helper to convert types.Session to facade.SessionInfo
func toFacadeInfo(sess *types.Session) facade.SessionInfo {
	return facade.SessionInfo{
		ID:      sess.ID,
		Title:   sess.Title,
		Path:    sess.Path,
		Branch:  sess.Branch,
		Status:  facade.SessionStatus(sess.Status),
		Program: sess.Program,
		AutoYes: sess.AutoYes,
	}
}