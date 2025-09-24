package session

import (
	"context"
	"time"
)

// Status represents the state of a session
type Status int

const (
	StatusRunning Status = iota
	StatusReady
	StatusLoading
	StatusPaused
)

// Session represents a managed work session
type Session struct {
	ID        string
	Title     string
	Path      string
	Branch    string
	Status    Status
	Program   string
	Height    int
	Width     int
	CreatedAt time.Time
	UpdatedAt time.Time
	AutoYes   bool
	Prompt    string
}

// CreateSessionRequest contains parameters for creating a new session
type CreateSessionRequest struct {
	Title   string
	Path    string
	Branch  string
	Program string
	Height  int
	Width   int
	AutoYes bool
	Prompt  string
}

// SessionOrchestrator coordinates session lifecycle operations
type SessionOrchestrator interface {
	// CreateSession creates a new session with the given parameters
	CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error)

	// StartSession starts an existing session
	StartSession(ctx context.Context, sessionID string) error

	// PauseSession pauses a running session
	PauseSession(ctx context.Context, sessionID string) error

	// ResumeSession resumes a paused session
	ResumeSession(ctx context.Context, sessionID string) error

	// StopSession stops and cleans up a session
	StopSession(ctx context.Context, sessionID string) error

	// GetSession retrieves session information
	GetSession(ctx context.Context, sessionID string) (*Session, error)

	// ListSessions lists all available sessions
	ListSessions(ctx context.Context) ([]*Session, error)

	// AttachSession attaches to a running session
	AttachSession(ctx context.Context, sessionID string) error

	// SendInput sends input to a session
	SendInput(ctx context.Context, sessionID string, input string) error

	// GetOutput retrieves recent output from a session
	GetOutput(ctx context.Context, sessionID string) (string, error)

	// UpdateSessionStatus updates the status of a session
	UpdateSessionStatus(ctx context.Context, sessionID string, status Status) error
}