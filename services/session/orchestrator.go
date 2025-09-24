package session

import (
	"claude-squad/services/types"
	"context"
)

// SessionOrchestrator coordinates session lifecycle operations
type SessionOrchestrator interface {
	// CreateSession creates a new session with the given parameters
	CreateSession(ctx context.Context, req types.CreateSessionRequest) (*types.Session, error)

	// StartSession starts an existing session
	StartSession(ctx context.Context, sessionID string) error

	// PauseSession pauses a running session
	PauseSession(ctx context.Context, sessionID string) error

	// ResumeSession resumes a paused session
	ResumeSession(ctx context.Context, sessionID string) error

	// StopSession stops and cleans up a session
	StopSession(ctx context.Context, sessionID string) error

	// GetSession retrieves session information
	GetSession(ctx context.Context, sessionID string) (*types.Session, error)

	// ListSessions lists all available sessions
	ListSessions(ctx context.Context) ([]*types.Session, error)

	// AttachSession attaches to a running session
	AttachSession(ctx context.Context, sessionID string) error

	// SendInput sends input to a session
	SendInput(ctx context.Context, sessionID string, input string) error

	// GetOutput retrieves recent output from a session
	GetOutput(ctx context.Context, sessionID string) (string, error)

	// UpdateSessionStatus updates the status of a session
	UpdateSessionStatus(ctx context.Context, sessionID string, status types.Status) error
}