package coreadapter

import (
	"context"
	"fmt"
	"path/filepath"

	"claude-squad/interface/facade"
	"claude-squad/services/git"
	"claude-squad/services/session"
)

// diffViewerAdapter adapts the git service to the DiffViewer facade
type diffViewerAdapter struct {
	orchestrator session.SessionOrchestrator
	gitService   git.GitService
}

// NewDiffViewer creates a new DiffViewer facade
func NewDiffViewer(orchestrator session.SessionOrchestrator, gitService git.GitService) facade.DiffViewer {
	return &diffViewerAdapter{
		orchestrator: orchestrator,
		gitService:   gitService,
	}
}

func (d *diffViewerAdapter) GetDiffStats(ctx context.Context, sessionID string) (*facade.DiffStats, error) {
	sess, err := d.orchestrator.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Get diff from git service
	diff, err := d.gitService.GetDiff(ctx, sess.Path)
	if err != nil {
		return nil, err
	}

	// Parse diff for stats (simplified - real implementation would parse properly)
	stats := &facade.DiffStats{
		Added:   0,
		Removed: 0,
		Content: diff,
	}

	// Count lines in diff (very simplified)
	for _, line := range []byte(diff) {
		if line == '+' {
			stats.Added++
		} else if line == '-' {
			stats.Removed++
		}
	}

	return stats, nil
}

func (d *diffViewerAdapter) UpdateDiffStats(ctx context.Context, sessionID string) error {
	// In the real implementation, this might trigger a cache refresh
	// For now, just validate the session exists
	_, err := d.orchestrator.GetSession(ctx, sessionID)
	return err
}

func (d *diffViewerAdapter) GetRepoName(ctx context.Context, sessionID string) (string, error) {
	sess, err := d.orchestrator.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}

	if sess.Path == "" {
		return "", fmt.Errorf("session has no path")
	}

	return filepath.Base(sess.Path), nil
}