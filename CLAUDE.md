# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Core Commands

### Building
```bash
# Build the main binary
go build -o cs .

# Build and install to ~/.local/bin
./build-local.sh

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o cs
```

### Testing
```bash
# Run all tests
go test -v ./...

# Run tests for specific package
go test -v ./session/git/...
```

### Running
```bash
# Run the application (must be in a git repository)
cs

# Run with specific AI assistant
cs -p "aider --model gpt-4"
cs -p "codex"
cs -p "gemini"

# Run in auto-accept mode (experimental)
cs -y
```

## Architecture Overview

### Core Components

**Main Application Flow** (`main.go` → `app/app.go`)
- Entry point validates git repository presence and launches TUI
- Uses Cobra for CLI command structure (reset, debug, version)
- Manages daemon mode for auto-accept functionality

**Session Management** (`session/`)
- Each AI assistant runs in an isolated tmux session
- Git worktrees provide isolated codebases per task
- Sessions persist across application restarts via state storage
- Status tracking: Running, Checked Out, Paused, Auto-Yes modes

**TUI Interface** (`ui/`, `app/`)
- Built with Bubble Tea framework for terminal UI
- Split-pane view showing session list and diff preview
- Keyboard navigation with vim-like bindings
- Real-time status updates from tmux sessions

**Git Integration** (`session/git/`)
- Creates and manages git worktrees for isolation
- Each session works on its own branch (cs/<session-name>)
- Automatic branch creation, commit, and push capabilities
- Diff generation for preview functionality

**Tmux Integration** (`session/tmux/`)
- Creates persistent terminal sessions for each AI agent
- Captures output for display in TUI
- Handles auto-accept mode by sending "y" responses
- Session cleanup on reset

**Storage & Config** (`config/`)
- Instance persistence in `~/.config/claude-squad/`
- State management for session recovery
- Configuration for default program and auto-yes settings

### Key Interactions

1. **New Session Creation**: TUI → Session → Git Worktree → Tmux → AI Agent Launch
2. **Session Attachment**: TUI → Tmux Attach → Interactive AI Session
3. **Checkout Process**: TUI → Git Commit → Session Status Update → Pause
4. **Push to GitHub**: TUI → Git Commit → Git Push → Branch URL Display

### Important Design Decisions

- **Worktree Isolation**: Each session operates in a separate git worktree to prevent conflicts between concurrent AI tasks
- **Tmux for Persistence**: Sessions survive application restarts by using tmux as the session backend
- **Daemon Mode**: Auto-yes functionality runs as a separate daemon process to handle automatic prompt responses
- **Global Instance Limit**: Maximum 10 concurrent sessions to manage resource usage

## Development Notes

- Always run from within a git repository (required by design)
- The application depends on external tools: tmux, git, gh (GitHub CLI)
- Session names are used as branch names, so they must be valid git branch identifiers
- The TUI refreshes every 3 seconds to capture session output updates