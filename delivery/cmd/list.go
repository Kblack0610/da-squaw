package cmd

import (
	"context"
	"fmt"

	"claude-squad/interface/facade"

	"github.com/spf13/cobra"
)

// NewListCmd creates a list command using the facade pattern
func NewListCmd(sessionManager facade.SessionManager) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			sessions, err := sessionManager.ListSessions(ctx)
			if err != nil {
				return fmt.Errorf("failed to list sessions: %w", err)
			}

			if len(sessions) == 0 {
				fmt.Println("No active sessions")
				return nil
			}

			fmt.Printf("Active sessions:\n")
			for _, sess := range sessions {
				status := getStatusString(sess.Status)
				fmt.Printf("  [%s] %s - %s (%s)\n",
					status, sess.Title, sess.Path, sess.Branch)
			}

			return nil
		},
	}
}

func getStatusString(status facade.SessionStatus) string {
	switch status {
	case facade.StatusRunning:
		return "RUNNING"
	case facade.StatusReady:
		return "READY"
	case facade.StatusPaused:
		return "PAUSED"
	case facade.StatusLoading:
		return "LOADING"
	default:
		return "UNKNOWN"
	}
}