package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"claude-squad/services/executor"
	"claude-squad/services/git"
	"claude-squad/services/storage"
	"claude-squad/services/tmux"
)

// orchestratorImpl is the concrete implementation of SessionOrchestrator
type orchestratorImpl struct {
	gitService  git.GitService
	tmuxService tmux.TmuxService
	storage     storage.StorageRepository
	executor    executor.CommandExecutor

	// In-memory cache of active sessions
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewOrchestrator creates a new SessionOrchestrator instance
func NewOrchestrator(
	gitService git.GitService,
	tmuxService tmux.TmuxService,
	storage storage.StorageRepository,
	executor executor.CommandExecutor,
) SessionOrchestrator {
	orch := &orchestratorImpl{
		gitService:  gitService,
		tmuxService: tmuxService,
		storage:     storage,
		executor:    executor,
		sessions:    make(map[string]*Session),
	}

	// Load existing sessions from storage
	ctx := context.Background()
	if sessions, err := storage.List(ctx, nil); err == nil {
		for _, s := range sessions {
			orch.sessions[s.ID] = &Session{
				ID:        s.ID,
				Title:     s.Title,
				Path:      s.Path,
				Branch:    s.Branch,
				Status:    s.Status,
				Program:   s.Program,
				Height:    s.Height,
				Width:     s.Width,
				CreatedAt: s.CreatedAt,
				UpdatedAt: s.UpdatedAt,
				AutoYes:   s.AutoYes,
				Prompt:    s.Prompt,
			}
		}
	}

	return orch
}

func (o *orchestratorImpl) CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	// Validate request
	if req.Title == "" {
		return nil, fmt.Errorf("session title is required")
	}
	if req.Path == "" {
		return nil, fmt.Errorf("session path is required")
	}

	// Check if path is a git repository
	isGitRepo, err := o.gitService.IsGitRepository(ctx, req.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to check git repository: %w", err)
	}
	if !isGitRepo {
		return nil, fmt.Errorf("path is not a git repository: %s", req.Path)
	}

	// Generate session ID
	sessionID := generateSessionID(req.Title)

	// Create branch if needed
	if req.Branch != "" {
		if err := o.gitService.CreateBranch(ctx, req.Path, req.Branch); err != nil {
			return nil, fmt.Errorf("failed to create branch: %w", err)
		}
	} else {
		// Use current branch
		currentBranch, err := o.gitService.GetCurrentBranch(ctx, req.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to get current branch: %w", err)
		}
		req.Branch = currentBranch.Name
	}

	// Create worktree
	worktreePath := fmt.Sprintf("%s-worktree-%s", req.Path, sessionID)
	worktree, err := o.gitService.CreateWorktree(ctx, req.Path, worktreePath, req.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	// Create tmux session
	tmuxSession, err := o.tmuxService.CreateSession(ctx, sessionID, worktree.Path, req.Program)
	if err != nil {
		// Cleanup worktree on failure
		_ = o.gitService.RemoveWorktree(ctx, worktreePath, true)
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Create session object
	session := &Session{
		ID:        sessionID,
		Title:     req.Title,
		Path:      worktree.Path,
		Branch:    req.Branch,
		Status:    StatusLoading,
		Program:   req.Program,
		Height:    req.Height,
		Width:     req.Width,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		AutoYes:   req.AutoYes,
		Prompt:    req.Prompt,
	}

	// Send initial prompt if provided
	if req.Prompt != "" {
		if err := o.tmuxService.SendKeys(ctx, tmuxSession.Name, req.Prompt); err != nil {
			// Log but don't fail
			fmt.Printf("warning: failed to send initial prompt: %v\n", err)
		}
	}

	// Save to storage
	storageData := &storage.SessionData{
		ID:        session.ID,
		Title:     session.Title,
		Path:      session.Path,
		Branch:    session.Branch,
		Status:    session.Status,
		Program:   session.Program,
		Height:    session.Height,
		Width:     session.Width,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
		AutoYes:   session.AutoYes,
		Prompt:    session.Prompt,
	}
	if err := o.storage.Create(ctx, storageData); err != nil {
		// Cleanup on failure
		_ = o.tmuxService.KillSession(ctx, tmuxSession.Name)
		_ = o.gitService.RemoveWorktree(ctx, worktreePath, true)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Cache session
	o.mu.Lock()
	o.sessions[sessionID] = session
	o.mu.Unlock()

	// Update status to ready
	go func() {
		time.Sleep(2 * time.Second) // Give the program time to start
		_ = o.UpdateSessionStatus(context.Background(), sessionID, StatusReady)
	}()

	return session, nil
}

func (o *orchestratorImpl) StartSession(ctx context.Context, sessionID string) error {
	session, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != StatusPaused {
		return fmt.Errorf("session is not paused")
	}

	// Recreate worktree
	worktree, err := o.gitService.CreateWorktree(ctx, session.Path, session.Path, session.Branch)
	if err != nil {
		return fmt.Errorf("failed to recreate worktree: %w", err)
	}

	// Recreate tmux session
	_, err = o.tmuxService.CreateSession(ctx, sessionID, worktree.Path, session.Program)
	if err != nil {
		return fmt.Errorf("failed to recreate tmux session: %w", err)
	}

	return o.UpdateSessionStatus(ctx, sessionID, StatusReady)
}

func (o *orchestratorImpl) PauseSession(ctx context.Context, sessionID string) error {
	session, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status == StatusPaused {
		return nil // Already paused
	}

	// Kill tmux session
	if err := o.tmuxService.KillSession(ctx, sessionID); err != nil {
		// Session might not exist, continue anyway
		fmt.Printf("warning: failed to kill tmux session: %v\n", err)
	}

	// Remove worktree but keep branch
	if err := o.gitService.RemoveWorktree(ctx, session.Path, false); err != nil {
		// Worktree might not exist, continue anyway
		fmt.Printf("warning: failed to remove worktree: %v\n", err)
	}

	return o.UpdateSessionStatus(ctx, sessionID, StatusPaused)
}

func (o *orchestratorImpl) ResumeSession(ctx context.Context, sessionID string) error {
	return o.StartSession(ctx, sessionID)
}

func (o *orchestratorImpl) StopSession(ctx context.Context, sessionID string) error {
	session, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Kill tmux session
	if err := o.tmuxService.KillSession(ctx, sessionID); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to kill tmux session: %v\n", err)
	}

	// Remove worktree
	if err := o.gitService.RemoveWorktree(ctx, session.Path, true); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to remove worktree: %v\n", err)
	}

	// Delete from storage
	if err := o.storage.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session from storage: %w", err)
	}

	// Remove from cache
	o.mu.Lock()
	delete(o.sessions, sessionID)
	o.mu.Unlock()

	return nil
}

func (o *orchestratorImpl) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	o.mu.RLock()
	session, exists := o.sessions[sessionID]
	o.mu.RUnlock()

	if exists {
		return session, nil
	}

	// Try loading from storage
	data, err := o.storage.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session = &Session{
		ID:        data.ID,
		Title:     data.Title,
		Path:      data.Path,
		Branch:    data.Branch,
		Status:    data.Status,
		Program:   data.Program,
		Height:    data.Height,
		Width:     data.Width,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
		AutoYes:   data.AutoYes,
		Prompt:    data.Prompt,
	}

	// Cache it
	o.mu.Lock()
	o.sessions[sessionID] = session
	o.mu.Unlock()

	return session, nil
}

func (o *orchestratorImpl) ListSessions(ctx context.Context) ([]*Session, error) {
	data, err := o.storage.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]*Session, len(data))
	for i, d := range data {
		sessions[i] = &Session{
			ID:        d.ID,
			Title:     d.Title,
			Path:      d.Path,
			Branch:    d.Branch,
			Status:    d.Status,
			Program:   d.Program,
			Height:    d.Height,
			Width:     d.Width,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			AutoYes:   d.AutoYes,
			Prompt:    d.Prompt,
		}
	}

	return sessions, nil
}

func (o *orchestratorImpl) AttachSession(ctx context.Context, sessionID string) error {
	session, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != StatusReady && session.Status != StatusRunning {
		return fmt.Errorf("session is not ready or running")
	}

	return o.tmuxService.AttachSession(ctx, sessionID)
}

func (o *orchestratorImpl) SendInput(ctx context.Context, sessionID string, input string) error {
	session, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != StatusReady && session.Status != StatusRunning {
		return fmt.Errorf("session is not ready or running")
	}

	return o.tmuxService.SendKeys(ctx, sessionID, input)
}

func (o *orchestratorImpl) GetOutput(ctx context.Context, sessionID string) (string, error) {
	session, err := o.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}

	if session.Status == StatusPaused {
		return "", fmt.Errorf("session is paused")
	}

	// Get the last pane of the session (assuming single window/pane for simplicity)
	output, err := o.tmuxService.CapturePane(ctx, sessionID, "0")
	if err != nil {
		return "", fmt.Errorf("failed to capture output: %w", err)
	}

	return output, nil
}

func (o *orchestratorImpl) UpdateSessionStatus(ctx context.Context, sessionID string, status Status) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	session, exists := o.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	// Update storage
	return o.storage.UpdateStatus(ctx, sessionID, status)
}

// generateSessionID creates a unique session ID from the title
func generateSessionID(title string) string {
	// Simple implementation - in production, use a proper ID generator
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", title, timestamp)
}