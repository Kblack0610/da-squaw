package tmux

import (
	"context"
	"io"
)

// Session represents a tmux session
type Session struct {
	Name      string
	ID        string
	Windows   int
	Created   string
	Attached  bool
	Width     int
	Height    int
	Directory string
}

// Window represents a tmux window
type Window struct {
	ID     string
	Name   string
	Active bool
	Panes  int
}

// Pane represents a tmux pane
type Pane struct {
	ID       string
	Active   bool
	Width    int
	Height   int
	Command  string
	PID      int
	Directory string
}

// TmuxService provides tmux session management operations
type TmuxService interface {
	// Session management
	CreateSession(ctx context.Context, name, startDir, command string) (*Session, error)
	AttachSession(ctx context.Context, sessionName string) error
	DetachSession(ctx context.Context, sessionName string) error
	KillSession(ctx context.Context, sessionName string) error
	ListSessions(ctx context.Context) ([]*Session, error)
	GetSession(ctx context.Context, sessionName string) (*Session, error)
	RenameSession(ctx context.Context, oldName, newName string) error
	SessionExists(ctx context.Context, sessionName string) (bool, error)

	// Window management
	CreateWindow(ctx context.Context, sessionName, windowName, command string) (*Window, error)
	KillWindow(ctx context.Context, sessionName, windowID string) error
	ListWindows(ctx context.Context, sessionName string) ([]*Window, error)
	RenameWindow(ctx context.Context, sessionName, windowID, newName string) error
	SelectWindow(ctx context.Context, sessionName, windowID string) error

	// Pane management
	SplitPane(ctx context.Context, sessionName, windowID string, vertical bool, command string) (*Pane, error)
	KillPane(ctx context.Context, sessionName, paneID string) error
	ListPanes(ctx context.Context, sessionName, windowID string) ([]*Pane, error)
	ResizePane(ctx context.Context, sessionName, paneID string, width, height int) error
	SelectPane(ctx context.Context, sessionName, paneID string) error

	// Input/Output operations
	SendKeys(ctx context.Context, sessionName string, keys string) error
	SendKeysToPane(ctx context.Context, sessionName, paneID, keys string) error
	CapturePane(ctx context.Context, sessionName, paneID string) (string, error)
	GetPaneOutput(ctx context.Context, sessionName, paneID string, lines int) (string, error)
	GetPaneScrollback(ctx context.Context, sessionName, paneID string) (string, error)

	// Streaming operations
	StreamOutput(ctx context.Context, sessionName string) (io.ReadCloser, error)
	StreamPaneOutput(ctx context.Context, sessionName, paneID string) (io.ReadCloser, error)

	// Configuration and utilities
	SetOption(ctx context.Context, sessionName, option, value string) error
	GetOption(ctx context.Context, sessionName, option string) (string, error)
	ResizeSession(ctx context.Context, sessionName string, width, height int) error
	HasActivity(ctx context.Context, sessionName string) (bool, error)
	GetSessionPID(ctx context.Context, sessionName string) (int, error)

	// Cleanup operations
	CleanupSessions(ctx context.Context, prefix string) error
	CleanupOrphanedSessions(ctx context.Context) error
}