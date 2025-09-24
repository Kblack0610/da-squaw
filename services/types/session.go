package types

import "time"

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

// SessionData represents the persistent data of a session (for storage)
type SessionData struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Path      string            `json:"path"`
	Branch    string            `json:"branch"`
	Status    Status            `json:"status"`
	Program   string            `json:"program"`
	Height    int               `json:"height"`
	Width     int               `json:"width"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	AutoYes   bool              `json:"auto_yes"`
	Prompt    string            `json:"prompt"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}