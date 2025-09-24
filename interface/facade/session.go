package facade

import (
	"context"
)

// SessionInfo contains basic session information
type SessionInfo struct {
	ID        string
	Title     string
	Path      string
	Branch    string
	Status    SessionStatus
	Program   string
	AutoYes   bool
}

// SessionStatus represents the state of a session
type SessionStatus int

const (
	StatusRunning SessionStatus = iota
	StatusReady
	StatusLoading
	StatusPaused
)

// SessionManager handles session lifecycle operations
type SessionManager interface {
	// List returns all sessions
	ListSessions(ctx context.Context) ([]SessionInfo, error)

	// Create a new session
	CreateSession(ctx context.Context, title, path, program string) (*SessionInfo, error)

	// Start/Stop operations
	StartSession(ctx context.Context, id string) error
	StopSession(ctx context.Context, id string) error
	PauseSession(ctx context.Context, id string) error
	ResumeSession(ctx context.Context, id string) error

	// Get single session info
	GetSession(ctx context.Context, id string) (*SessionInfo, error)

	// Update session title
	UpdateTitle(ctx context.Context, id string, title string) error
}

// SessionInteractor handles interaction with running sessions
type SessionInteractor interface {
	// Attach to a session's tmux
	AttachSession(ctx context.Context, id string) error

	// Send input to session
	SendKeys(ctx context.Context, id string, keys string) error
	SendPrompt(ctx context.Context, id string, prompt string) error

	// Check if session has prompts waiting
	HasPrompt(ctx context.Context, id string) (bool, error)
}

// SessionViewer handles viewing session output
type SessionViewer interface {
	// Get current output preview
	GetPreview(ctx context.Context, id string) (string, error)

	// Get full output history
	GetFullHistory(ctx context.Context, id string) (string, error)

	// Check if output has updated
	HasUpdated(ctx context.Context, id string, lastPreview string) (bool, error)
}