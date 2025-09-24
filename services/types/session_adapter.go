package types

import (
	"fmt"
	"path/filepath"
)

// DiffStats represents git diff statistics
type DiffStats struct {
	Added   int
	Removed int
	Content string
	Error   error
}

// IsEmpty checks if diff stats are empty
func (d *DiffStats) IsEmpty() bool {
	return d.Added == 0 && d.Removed == 0
}

// SessionAdapter adapts types.Session to work with UI components that expect the old Instance interface
type SessionAdapter struct {
	*Session
	lastPreview string
	previewWidth int
	previewHeight int
}

// NewSessionAdapter creates a new adapter from a Session
func NewSessionAdapter(session *Session) *SessionAdapter {
	return &SessionAdapter{
		Session: session,
	}
}

// RepoName returns the repository name from the path
func (s *SessionAdapter) RepoName() (string, error) {
	if s.Path == "" {
		return "", fmt.Errorf("no path set")
	}
	return filepath.Base(s.Path), nil
}

// SetStatus updates the status
func (s *SessionAdapter) SetStatus(status Status) {
	s.Status = status
}

// Preview returns the last captured output (mock implementation)
func (s *SessionAdapter) Preview() (string, error) {
	// In the new architecture, preview would be fetched from orchestrator
	// For now, return a placeholder
	if s.lastPreview == "" {
		return fmt.Sprintf("Session: %s\nPath: %s\nStatus: %v", s.Title, s.Path, s.Status), nil
	}
	return s.lastPreview, nil
}

// SetPreviewContent updates the cached preview content
func (s *SessionAdapter) SetPreviewContent(content string) {
	s.lastPreview = content
}

// HasUpdated checks if there are updates (mock implementation)
func (s *SessionAdapter) HasUpdated() (updated bool, hasPrompt bool) {
	// In new architecture, this would check with orchestrator
	return false, false
}

// TapEnter sends enter key (mock implementation)
func (s *SessionAdapter) TapEnter() {
	// In new architecture, this would use orchestrator.SendInput
}

// Attach attaches to the session (mock implementation)
func (s *SessionAdapter) Attach() (chan struct{}, error) {
	// In new architecture, this would use orchestrator.AttachSession
	done := make(chan struct{})
	close(done)
	return done, nil
}

// SetPreviewSize sets the preview dimensions
func (s *SessionAdapter) SetPreviewSize(width, height int) error {
	s.previewWidth = width
	s.previewHeight = height
	return nil
}

// Started returns whether the session is started
func (s *SessionAdapter) Started() bool {
	return s.Status == StatusRunning || s.Status == StatusReady
}

// SetTitle updates the title
func (s *SessionAdapter) SetTitle(title string) error {
	s.Title = title
	return nil
}

// Paused returns whether the session is paused
func (s *SessionAdapter) Paused() bool {
	return s.Status == StatusPaused
}

// TmuxAlive returns whether tmux session is alive (mock)
func (s *SessionAdapter) TmuxAlive() bool {
	return s.Status == StatusRunning || s.Status == StatusReady
}

// GetGitWorktree returns nil for now (mock)
func (s *SessionAdapter) GetGitWorktree() (interface{}, error) {
	return nil, nil
}

// Start starts the session (mock)
func (s *SessionAdapter) Start(firstTimeSetup bool) error {
	s.Status = StatusLoading
	return nil
}

// Kill kills the session (mock)
func (s *SessionAdapter) Kill() error {
	s.Status = StatusPaused
	return nil
}

// Pause pauses the session (mock)
func (s *SessionAdapter) Pause() error {
	s.Status = StatusPaused
	return nil
}

// Resume resumes the session (mock)
func (s *SessionAdapter) Resume() error {
	s.Status = StatusReady
	return nil
}

// ToInstanceData converts to old InstanceData format (for storage compatibility)
func (s *SessionAdapter) ToInstanceData() interface{} {
	return s.Session
}

// GetDiffStats returns git diff stats (mock implementation)
func (s *SessionAdapter) GetDiffStats() *DiffStats {
	// In new architecture, this would use GitService to get diff
	// For now, return empty stats
	return &DiffStats{Added: 0, Removed: 0}
}

// PreviewFullHistory returns full history preview (mock implementation)
func (s *SessionAdapter) PreviewFullHistory() (string, error) {
	// In new architecture, this would use orchestrator to get full output
	return s.Preview()
}