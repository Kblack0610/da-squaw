package main

import (
	"fmt"
	"os"

	"claude-squad/delivery/cmd"
	"claude-squad/interface/coreadapter"
	"claude-squad/services/executor"
	"claude-squad/services/git"
	"claude-squad/services/session"
	"claude-squad/services/storage"
	"claude-squad/services/tmux"

	"github.com/spf13/cobra"
)

// Example of how to wire up the application using facades
func main() {
	// Initialize core services (this would be in app.InitializeDependencies)
	executor := executor.NewExecutor(nil)
	gitService := git.NewExecGitService(executor)
	tmuxService := tmux.NewExecTmuxService(executor)
	storage := storage.NewJSONRepository("~/.claude-squad/sessions")
	orchestrator := session.NewOrchestrator(gitService, tmuxService, storage, executor)

	// Create facades (thin adapters)
	sessionManager := coreadapter.NewSessionManager(orchestrator)
	sessionInteractor := coreadapter.NewSessionInteractor(orchestrator)
	sessionViewer := coreadapter.NewSessionViewer(orchestrator)
	diffViewer := coreadapter.NewDiffViewer(orchestrator, gitService)

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "cs",
		Short: "Claude Squad - Manage AI coding sessions",
	}

	// Add subcommands with facade dependencies
	rootCmd.AddCommand(cmd.NewListCmd(sessionManager))
	rootCmd.AddCommand(cmd.NewDiffCmd(sessionManager, diffViewer))

	// The TUI app would also receive facades:
	// rootCmd.AddCommand(cmd.NewUICmd(sessionManager, sessionViewer, sessionInteractor))

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Example of how a TUI widget would use facades
/*
type SessionListWidget struct {
    manager facade.SessionManager
    viewer  facade.SessionViewer
}

func NewSessionListWidget(m facade.SessionManager, v facade.SessionViewer) *SessionListWidget {
    return &SessionListWidget{manager: m, viewer: v}
}

func (w *SessionListWidget) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Use only facade methods, no direct service imports
    sessions, _ := w.manager.ListSessions(context.Background())
    // ...
}
*/