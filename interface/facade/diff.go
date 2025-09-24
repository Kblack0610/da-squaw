package facade

import (
	"context"
)

// DiffStats contains git diff statistics
type DiffStats struct {
	Added   int
	Removed int
	Content string
}

// DiffViewer provides git diff information for sessions
type DiffViewer interface {
	// Get diff statistics for a session
	GetDiffStats(ctx context.Context, sessionID string) (*DiffStats, error)

	// Update diff stats (trigger refresh)
	UpdateDiffStats(ctx context.Context, sessionID string) error

	// Get repository name from session path
	GetRepoName(ctx context.Context, sessionID string) (string, error)
}