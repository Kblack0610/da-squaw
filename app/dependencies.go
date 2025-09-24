package app

import (
	"claude-squad/config"
	"claude-squad/services/executor"
	"claude-squad/services/git"
	"claude-squad/services/session"
	"claude-squad/services/storage"
	"claude-squad/services/tmux"
	"fmt"
	"path/filepath"
)

// Dependencies holds all service dependencies for the application
type Dependencies struct {
	Executor     executor.CommandExecutor
	GitService   git.GitService
	TmuxService  tmux.TmuxService
	Storage      storage.StorageRepository
	Orchestrator session.SessionOrchestrator
	Config       *config.Config
	State        config.AppState
}

// InitializeDependencies creates and wires up all service dependencies
func InitializeDependencies() (*Dependencies, error) {
	// Load configuration
	cfg := config.LoadConfig()
	state := config.LoadState()

	// Create executor
	exec := executor.NewDefaultExecutor()

	// Create services
	gitService := git.NewExecGitService(exec)
	tmuxService := tmux.NewExecTmuxService(exec)

	// Set up storage path
	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	storagePath := filepath.Join(configDir, "sessions")

	// Create storage repository
	storageRepo, err := storage.NewJSONRepository(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage repository: %w", err)
	}

	// Create orchestrator
	orchestrator := session.NewOrchestrator(gitService, tmuxService, storageRepo, exec)

	return &Dependencies{
		Executor:     exec,
		GitService:   gitService,
		TmuxService:  tmuxService,
		Storage:      storageRepo,
		Orchestrator: orchestrator,
		Config:       cfg,
		State:        state,
	}, nil
}

// Cleanup performs cleanup operations for all services
func (d *Dependencies) Cleanup() error {
	// Currently services don't require explicit cleanup
	// Add cleanup logic here if needed in the future
	return nil
}