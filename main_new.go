package main

import (
	"claude-squad/app"
	"claude-squad/config"
	"claude-squad/daemon"
	"claude-squad/log"
	"claude-squad/services/executor"
	"claude-squad/services/git"
	"claude-squad/services/session"
	"claude-squad/services/storage"
	"claude-squad/services/tmux"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	version     = "1.0.13"
	programFlag string
	autoYesFlag bool
	daemonFlag  bool

	// Service dependencies
	deps *app.Dependencies

	rootCmd = &cobra.Command{
		Use:   "claude-squad",
		Short: "Claude Squad - Manage multiple AI agents like Claude Code, Aider, Codex, and Amp.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			log.Initialize(daemonFlag)
			defer log.Close()

			if daemonFlag {
				return runDaemon(ctx)
			}

			// Check if we're in a git repository
			currentDir, err := filepath.Abs(".")
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// Initialize dependencies
			deps, err = app.InitializeDependencies()
			if err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}
			defer deps.Cleanup()

			// Check if current directory is a git repo
			isRepo, err := deps.GitService.IsGitRepository(ctx, currentDir)
			if err != nil {
				return fmt.Errorf("failed to check git repository: %w", err)
			}
			if !isRepo {
				return fmt.Errorf("error: claude-squad must be run from within a git repository")
			}

			// Program flag overrides config
			program := deps.Config.DefaultProgram
			if programFlag != "" {
				program = programFlag
			}

			// AutoYes flag overrides config
			autoYes := deps.Config.AutoYes
			if autoYesFlag {
				autoYes = true
			}

			// Launch daemon if autoYes is enabled
			if autoYes {
				defer func() {
					if err := daemon.LaunchDaemon(); err != nil {
						log.ErrorLog.Printf("failed to launch daemon: %v", err)
					}
				}()
			}

			// Kill any running daemon
			if err := daemon.StopDaemon(); err != nil {
				log.ErrorLog.Printf("failed to stop daemon: %v", err)
			}

			// Run the application
			return app.RunNew(ctx, program, autoYes)
		},
	}

	resetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset all stored instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			log.Initialize(false)
			defer log.Close()

			// Initialize dependencies
			deps, err := app.InitializeDependencies()
			if err != nil {
				return fmt.Errorf("failed to initialize dependencies: %w", err)
			}
			defer deps.Cleanup()

			// Delete all sessions from storage
			if err := deps.Storage.DeleteAll(ctx); err != nil {
				return fmt.Errorf("failed to reset storage: %w", err)
			}
			fmt.Println("Storage has been reset successfully")

			// Cleanup tmux sessions
			if err := deps.TmuxService.CleanupSessions(ctx, "claudesquad_"); err != nil {
				return fmt.Errorf("failed to cleanup tmux sessions: %w", err)
			}
			fmt.Println("Tmux sessions have been cleaned up")

			// Cleanup git worktrees
			repoPath, err := deps.GitService.GetRepositoryRoot(ctx, ".")
			if err == nil {
				if err := deps.GitService.CleanupWorktrees(ctx, repoPath); err != nil {
					return fmt.Errorf("failed to cleanup worktrees: %w", err)
				}
				fmt.Println("Worktrees have been cleaned up")
			}

			// Kill any daemon
			if err := daemon.StopDaemon(); err != nil {
				return err
			}
			fmt.Println("Daemon has been stopped")

			return nil
		},
	}

	debugCmd = &cobra.Command{
		Use:   "debug",
		Short: "Print debug information like config paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Initialize(false)
			defer log.Close()

			cfg := config.LoadConfig()

			configDir, err := config.GetConfigDir()
			if err != nil {
				return fmt.Errorf("failed to get config directory: %w", err)
			}
			configJson, _ := json.MarshalIndent(cfg, "", "  ")

			fmt.Printf("Config: %s\n%s\n", filepath.Join(configDir, config.ConfigFileName), configJson)

			return nil
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of claude-squad",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("claude-squad version %s\n", version)
			fmt.Printf("https://github.com/smtg-ai/claude-squad/releases/tag/v%s\n", version)
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&programFlag, "program", "p", "",
		"Program to run in new instances (e.g. 'aider --model ollama_chat/gemma3:1b')")
	rootCmd.Flags().BoolVarP(&autoYesFlag, "autoyes", "y", false,
		"[experimental] If enabled, all instances will automatically accept prompts")
	rootCmd.Flags().BoolVar(&daemonFlag, "daemon", false, "Run a program that loads all sessions"+
		" and runs autoyes mode on them.")

	// Hide the daemonFlag as it's only for internal use
	err := rootCmd.Flags().MarkHidden("daemon")
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(resetCmd)
}

func runDaemon(ctx context.Context) error {
	// Initialize dependencies for daemon
	deps, err := app.InitializeDependencies()
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	defer deps.Cleanup()

	// Create daemon service
	d := daemon.NewDaemon(deps.Orchestrator, deps.Config)
	return d.Run(ctx)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}