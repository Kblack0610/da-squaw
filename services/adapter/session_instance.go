package adapter

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"claude-squad/services/session"
	"claude-squad/services/types"
	"claude-squad/session/git"
	"claude-squad/session/tmux"
)

// SessionInstance adapts the new service architecture to work with UI components
// that expect the old Instance interface. It wraps a types.Session and uses
// the orchestrator to perform operations.
type SessionInstance struct {
	*types.Session
	orchestrator session.SessionOrchestrator
	ctx          context.Context

	// Cached data
	lastPreview   string
	previewWidth  int
	previewHeight int
	diffStats     *git.DiffStats
	gitWorktree   *git.GitWorktree
	tmuxSession   *tmux.TmuxSession
}

// NewSessionInstance creates a new adapter from a Session
func NewSessionInstance(sess *types.Session, orchestrator session.SessionOrchestrator) *SessionInstance {
	return &SessionInstance{
		Session:      sess,
		orchestrator: orchestrator,
		ctx:          context.Background(),
	}
}

// RepoName returns the repository name from the path
func (s *SessionInstance) RepoName() (string, error) {
	if s.Path == "" {
		return "", fmt.Errorf("no path set")
	}
	return filepath.Base(s.Path), nil
}

// SetStatus updates the status
func (s *SessionInstance) SetStatus(status types.Status) {
	s.Status = status
	// Update in orchestrator
	_ = s.orchestrator.UpdateSessionStatus(s.ctx, s.ID, status)
}

// Start starts the session
func (s *SessionInstance) Start(firstTimeSetup bool) error {
	if s.Status == types.StatusPaused {
		return s.orchestrator.ResumeSession(s.ctx, s.ID)
	}
	return s.orchestrator.StartSession(s.ctx, s.ID)
}

// Kill kills the session
func (s *SessionInstance) Kill() error {
	return s.orchestrator.StopSession(s.ctx, s.ID)
}

// Preview returns the last captured output
func (s *SessionInstance) Preview() (string, error) {
	output, err := s.orchestrator.GetOutput(s.ctx, s.ID)
	if err != nil {
		return s.lastPreview, err
	}
	s.lastPreview = output
	return output, nil
}

// PreviewFullHistory returns full history preview
func (s *SessionInstance) PreviewFullHistory() (string, error) {
	// Get full output from orchestrator
	return s.orchestrator.GetOutput(s.ctx, s.ID)
}

// HasUpdated checks if there are updates
func (s *SessionInstance) HasUpdated() (updated bool, hasPrompt bool) {
	// Check if output has changed since last preview
	output, err := s.orchestrator.GetOutput(s.ctx, s.ID)
	if err != nil {
		return false, false
	}

	updated = output != s.lastPreview
	// Simple heuristic: check for prompt patterns
	hasPrompt = strings.Contains(output, "[Y/n]") ||
	           strings.Contains(output, "(y/N)") ||
	           strings.Contains(output, "Continue?") ||
	           strings.HasSuffix(strings.TrimSpace(output), ">")

	return updated, hasPrompt
}

// TapEnter sends enter key
func (s *SessionInstance) TapEnter() {
	_ = s.orchestrator.SendInput(s.ctx, s.ID, "\n")
}

// SendKeys sends keys to the session
func (s *SessionInstance) SendKeys(keys string) error {
	return s.orchestrator.SendInput(s.ctx, s.ID, keys)
}

// SendPrompt sends a prompt to the session
func (s *SessionInstance) SendPrompt(prompt string) error {
	return s.orchestrator.SendInput(s.ctx, s.ID, prompt+"\n")
}

// Attach attaches to the session
func (s *SessionInstance) Attach() (chan struct{}, error) {
	done := make(chan struct{})
	go func() {
		_ = s.orchestrator.AttachSession(s.ctx, s.ID)
		close(done)
	}()
	return done, nil
}

// SetPreviewSize sets the preview dimensions
func (s *SessionInstance) SetPreviewSize(width, height int) error {
	s.previewWidth = width
	s.previewHeight = height
	return nil
}

// Started returns whether the session is started
func (s *SessionInstance) Started() bool {
	return s.Status == types.StatusRunning || s.Status == types.StatusReady
}

// SetTitle updates the title
func (s *SessionInstance) SetTitle(title string) error {
	s.Title = title
	// Update in storage through orchestrator
	sess, err := s.orchestrator.GetSession(s.ctx, s.ID)
	if err != nil {
		return err
	}
	sess.Title = title
	return s.orchestrator.UpdateSessionStatus(s.ctx, s.ID, s.Status)
}

// Paused returns whether the session is paused
func (s *SessionInstance) Paused() bool {
	return s.Status == types.StatusPaused
}

// TmuxAlive returns whether tmux session is alive
func (s *SessionInstance) TmuxAlive() bool {
	return s.Status == types.StatusRunning || s.Status == types.StatusReady
}

// Pause pauses the session
func (s *SessionInstance) Pause() error {
	return s.orchestrator.PauseSession(s.ctx, s.ID)
}

// Resume resumes the session
func (s *SessionInstance) Resume() error {
	return s.orchestrator.ResumeSession(s.ctx, s.ID)
}

// GetGitWorktree returns the git worktree (compatibility method)
func (s *SessionInstance) GetGitWorktree() (*git.GitWorktree, error) {
	// In new architecture, worktree is managed by GitService
	// Return a mock or cached worktree for compatibility
	if s.gitWorktree == nil {
		s.gitWorktree = &git.GitWorktree{
			Path:   s.Path,
			Branch: s.Branch,
		}
	}
	return s.gitWorktree, nil
}

// SetTmuxSession sets the tmux session (compatibility method)
func (s *SessionInstance) SetTmuxSession(session *tmux.TmuxSession) {
	s.tmuxSession = session
}

// UpdateDiffStats updates git diff stats
func (s *SessionInstance) UpdateDiffStats() error {
	// In new architecture, this would use GitService
	// For now, create empty stats
	s.diffStats = &git.DiffStats{
		Added:   0,
		Removed: 0,
	}
	return nil
}

// GetDiffStats returns git diff stats
func (s *SessionInstance) GetDiffStats() *git.DiffStats {
	if s.diffStats == nil {
		s.diffStats = &git.DiffStats{
			Added:   0,
			Removed: 0,
		}
	}
	return s.diffStats
}

// ToInstanceData converts to storage format (compatibility)
func (s *SessionInstance) ToInstanceData() interface{} {
	return &types.SessionData{
		ID:        s.ID,
		Title:     s.Title,
		Path:      s.Path,
		Branch:    s.Branch,
		Status:    s.Status,
		Program:   s.Program,
		Height:    s.Height,
		Width:     s.Width,
		CreatedAt: s.CreatedAt,
		UpdatedAt: time.Now(),
		AutoYes:   s.AutoYes,
		Prompt:    s.Prompt,
	}
}