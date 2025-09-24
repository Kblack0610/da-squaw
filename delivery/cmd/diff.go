package cmd

import (
	"context"
	"fmt"

	"claude-squad/interface/facade"

	"github.com/spf13/cobra"
)

// NewDiffCmd creates a diff command using the facade pattern
func NewDiffCmd(sessionManager facade.SessionManager, diffViewer facade.DiffViewer) *cobra.Command {
	return &cobra.Command{
		Use:   "diff [session-title]",
		Short: "Show git diff for a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			title := args[0]

			// Find session by title
			sessions, err := sessionManager.ListSessions(ctx)
			if err != nil {
				return fmt.Errorf("failed to list sessions: %w", err)
			}

			var sessionID string
			for _, sess := range sessions {
				if sess.Title == title {
					sessionID = sess.ID
					break
				}
			}

			if sessionID == "" {
				return fmt.Errorf("session '%s' not found", title)
			}

			// Get diff stats
			stats, err := diffViewer.GetDiffStats(ctx, sessionID)
			if err != nil {
				return fmt.Errorf("failed to get diff: %w", err)
			}

			if stats.Added == 0 && stats.Removed == 0 {
				fmt.Println("No changes")
				return nil
			}

			fmt.Printf("Changes in session '%s':\n", title)
			fmt.Printf("  +%d additions\n", stats.Added)
			fmt.Printf("  -%d deletions\n", stats.Removed)

			if stats.Content != "" {
				fmt.Printf("\n%s\n", stats.Content)
			}

			return nil
		},
	}
}