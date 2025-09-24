package tmux

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"claude-squad/services/executor"
)

const tmuxPrefix = "claudesquad_"

var whiteSpaceRegex = regexp.MustCompile(`\s+`)

// execTmuxService is the concrete implementation of TmuxService using command execution
type execTmuxService struct {
	executor executor.CommandExecutor
}

// NewExecTmuxService creates a new TmuxService implementation using command execution
func NewExecTmuxService(exec executor.CommandExecutor) TmuxService {
	return &execTmuxService{
		executor: exec,
	}
}

// sanitizeTmuxName converts a string to a valid tmux session name
func (s *execTmuxService) sanitizeTmuxName(name string) string {
	name = whiteSpaceRegex.ReplaceAllString(name, "")
	name = strings.ReplaceAll(name, ".", "_") // tmux replaces dots with underscores
	return fmt.Sprintf("%s%s", tmuxPrefix, name)
}

// runTmuxCommand executes a tmux command
func (s *execTmuxService) runTmuxCommand(ctx context.Context, args ...string) (string, error) {
	cmd := executor.Command{
		Program: "tmux",
		Args:    args,
		Timeout: 10 * time.Second,
	}

	result, err := s.executor.Execute(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("tmux command failed: %w", err)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("tmux command failed with exit code %d: %s", result.ExitCode, string(result.Stderr))
	}

	return string(result.Stdout), nil
}

// Session management

func (s *execTmuxService) CreateSession(ctx context.Context, name, startDir, command string) (*Session, error) {
	sanitizedName := s.sanitizeTmuxName(name)

	// Check if session already exists
	if exists, _ := s.SessionExists(ctx, sanitizedName); exists {
		return nil, fmt.Errorf("session already exists: %s", sanitizedName)
	}

	// Create new detached session
	args := []string{"new-session", "-d", "-s", sanitizedName}
	if startDir != "" {
		args = append(args, "-c", startDir)
	}
	if command != "" {
		args = append(args, command)
	}

	if _, err := s.runTmuxCommand(ctx, args...); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return s.GetSession(ctx, sanitizedName)
}

func (s *execTmuxService) AttachSession(ctx context.Context, sessionName string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	// Check if session exists
	if exists, _ := s.SessionExists(ctx, sanitizedName); !exists {
		return fmt.Errorf("session does not exist: %s", sanitizedName)
	}

	// Use interactive execution for attachment
	cmd := executor.Command{
		Program: "tmux",
		Args:    []string{"attach-session", "-t", sanitizedName},
	}

	// This needs to be run interactively
	pipe, err := s.executor.ExecuteInteractive(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to attach session: %w", err)
	}

	// Wait for detach
	if pipe != nil {
		pipe.Close()
	}

	return nil
}

func (s *execTmuxService) DetachSession(ctx context.Context, sessionName string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	// Send detach key sequence
	return s.SendKeys(ctx, sanitizedName, "C-b d")
}

func (s *execTmuxService) KillSession(ctx context.Context, sessionName string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	if _, err := s.runTmuxCommand(ctx, "kill-session", "-t", sanitizedName); err != nil {
		// Session might not exist, which is ok
		if !strings.Contains(err.Error(), "can't find session") {
			return fmt.Errorf("failed to kill session: %w", err)
		}
	}
	return nil
}

func (s *execTmuxService) ListSessions(ctx context.Context) ([]*Session, error) {
	output, err := s.runTmuxCommand(ctx, "ls", "-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}:#{session_width}:#{session_height}:#{pane_current_path}")
	if err != nil {
		if strings.Contains(err.Error(), "no server running") {
			return []*Session{}, nil
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	sessions := make([]*Session, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}

		windows, _ := strconv.Atoi(parts[1])
		attached := parts[3] == "1"
		width, _ := strconv.Atoi(parts[4])
		height, _ := strconv.Atoi(parts[5])

		sessions = append(sessions, &Session{
			Name:      parts[0],
			ID:        parts[0],
			Windows:   windows,
			Created:   parts[2],
			Attached:  attached,
			Width:     width,
			Height:    height,
			Directory: parts[6],
		})
	}

	return sessions, nil
}

func (s *execTmuxService) GetSession(ctx context.Context, sessionName string) (*Session, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	sessions, err := s.ListSessions(ctx)
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.Name == sanitizedName {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sanitizedName)
}

func (s *execTmuxService) RenameSession(ctx context.Context, oldName, newName string) error {
	oldSanitized := s.sanitizeTmuxName(oldName)
	newSanitized := s.sanitizeTmuxName(newName)

	if _, err := s.runTmuxCommand(ctx, "rename-session", "-t", oldSanitized, newSanitized); err != nil {
		return fmt.Errorf("failed to rename session: %w", err)
	}
	return nil
}

func (s *execTmuxService) SessionExists(ctx context.Context, sessionName string) (bool, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	_, err := s.runTmuxCommand(ctx, "has-session", "-t", sanitizedName)
	if err != nil {
		if strings.Contains(err.Error(), "can't find session") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Window management

func (s *execTmuxService) CreateWindow(ctx context.Context, sessionName, windowName, command string) (*Window, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	args := []string{"new-window", "-t", sanitizedName}
	if windowName != "" {
		args = append(args, "-n", windowName)
	}
	if command != "" {
		args = append(args, command)
	}

	if _, err := s.runTmuxCommand(ctx, args...); err != nil {
		return nil, fmt.Errorf("failed to create window: %w", err)
	}

	// Get window info
	windows, err := s.ListWindows(ctx, sessionName)
	if err != nil {
		return nil, err
	}

	for _, w := range windows {
		if w.Name == windowName {
			return w, nil
		}
	}

	// Return the last created window if name doesn't match
	if len(windows) > 0 {
		return windows[len(windows)-1], nil
	}

	return nil, fmt.Errorf("window not found after creation")
}

func (s *execTmuxService) KillWindow(ctx context.Context, sessionName, windowID string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, windowID)

	if _, err := s.runTmuxCommand(ctx, "kill-window", "-t", target); err != nil {
		return fmt.Errorf("failed to kill window: %w", err)
	}
	return nil
}

func (s *execTmuxService) ListWindows(ctx context.Context, sessionName string) ([]*Window, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	output, err := s.runTmuxCommand(ctx, "list-windows", "-t", sanitizedName, "-F", "#{window_id}:#{window_name}:#{window_active}:#{window_panes}")
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	windows := make([]*Window, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 4 {
			continue
		}

		active := parts[2] == "1"
		panes, _ := strconv.Atoi(parts[3])

		windows = append(windows, &Window{
			ID:     parts[0],
			Name:   parts[1],
			Active: active,
			Panes:  panes,
		})
	}

	return windows, nil
}

func (s *execTmuxService) RenameWindow(ctx context.Context, sessionName, windowID, newName string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, windowID)

	if _, err := s.runTmuxCommand(ctx, "rename-window", "-t", target, newName); err != nil {
		return fmt.Errorf("failed to rename window: %w", err)
	}
	return nil
}

func (s *execTmuxService) SelectWindow(ctx context.Context, sessionName, windowID string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, windowID)

	if _, err := s.runTmuxCommand(ctx, "select-window", "-t", target); err != nil {
		return fmt.Errorf("failed to select window: %w", err)
	}
	return nil
}

// Pane management

func (s *execTmuxService) SplitPane(ctx context.Context, sessionName, windowID string, vertical bool, command string) (*Pane, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, windowID)

	args := []string{"split-window", "-t", target}
	if vertical {
		args = append(args, "-v")
	} else {
		args = append(args, "-h")
	}
	if command != "" {
		args = append(args, command)
	}

	if _, err := s.runTmuxCommand(ctx, args...); err != nil {
		return nil, fmt.Errorf("failed to split pane: %w", err)
	}

	// Get pane info
	panes, err := s.ListPanes(ctx, sessionName, windowID)
	if err != nil {
		return nil, err
	}

	// Return the last created pane
	if len(panes) > 0 {
		return panes[len(panes)-1], nil
	}

	return nil, fmt.Errorf("pane not found after creation")
}

func (s *execTmuxService) KillPane(ctx context.Context, sessionName, paneID string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, paneID)

	if _, err := s.runTmuxCommand(ctx, "kill-pane", "-t", target); err != nil {
		return fmt.Errorf("failed to kill pane: %w", err)
	}
	return nil
}

func (s *execTmuxService) ListPanes(ctx context.Context, sessionName, windowID string) ([]*Pane, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, windowID)

	output, err := s.runTmuxCommand(ctx, "list-panes", "-t", target, "-F", "#{pane_id}:#{pane_active}:#{pane_width}:#{pane_height}:#{pane_current_command}:#{pane_pid}:#{pane_current_path}")
	if err != nil {
		return nil, fmt.Errorf("failed to list panes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	panes := make([]*Pane, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}

		active := parts[1] == "1"
		width, _ := strconv.Atoi(parts[2])
		height, _ := strconv.Atoi(parts[3])
		pid, _ := strconv.Atoi(parts[5])

		panes = append(panes, &Pane{
			ID:        parts[0],
			Active:    active,
			Width:     width,
			Height:    height,
			Command:   parts[4],
			PID:       pid,
			Directory: parts[6],
		})
	}

	return panes, nil
}

func (s *execTmuxService) ResizePane(ctx context.Context, sessionName, paneID string, width, height int) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, paneID)

	// Resize width
	if width > 0 {
		if _, err := s.runTmuxCommand(ctx, "resize-pane", "-t", target, "-x", strconv.Itoa(width)); err != nil {
			return fmt.Errorf("failed to resize pane width: %w", err)
		}
	}

	// Resize height
	if height > 0 {
		if _, err := s.runTmuxCommand(ctx, "resize-pane", "-t", target, "-y", strconv.Itoa(height)); err != nil {
			return fmt.Errorf("failed to resize pane height: %w", err)
		}
	}

	return nil
}

func (s *execTmuxService) SelectPane(ctx context.Context, sessionName, paneID string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, paneID)

	if _, err := s.runTmuxCommand(ctx, "select-pane", "-t", target); err != nil {
		return fmt.Errorf("failed to select pane: %w", err)
	}
	return nil
}

// Input/Output operations

func (s *execTmuxService) SendKeys(ctx context.Context, sessionName string, keys string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	if _, err := s.runTmuxCommand(ctx, "send-keys", "-t", sanitizedName, keys); err != nil {
		return fmt.Errorf("failed to send keys: %w", err)
	}
	return nil
}

func (s *execTmuxService) SendKeysToPane(ctx context.Context, sessionName, paneID, keys string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, paneID)

	if _, err := s.runTmuxCommand(ctx, "send-keys", "-t", target, keys); err != nil {
		return fmt.Errorf("failed to send keys to pane: %w", err)
	}
	return nil
}

func (s *execTmuxService) CapturePane(ctx context.Context, sessionName, paneID string) (string, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, paneID)

	output, err := s.runTmuxCommand(ctx, "capture-pane", "-t", target, "-p")
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}
	return output, nil
}

func (s *execTmuxService) GetPaneOutput(ctx context.Context, sessionName, paneID string, lines int) (string, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, paneID)

	args := []string{"capture-pane", "-t", target, "-p"}
	if lines > 0 {
		args = append(args, "-S", fmt.Sprintf("-%d", lines))
	}

	output, err := s.runTmuxCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get pane output: %w", err)
	}
	return output, nil
}

func (s *execTmuxService) GetPaneScrollback(ctx context.Context, sessionName, paneID string) (string, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)
	target := fmt.Sprintf("%s:%s", sanitizedName, paneID)

	// Capture entire scrollback buffer
	output, err := s.runTmuxCommand(ctx, "capture-pane", "-t", target, "-p", "-S", "-")
	if err != nil {
		return "", fmt.Errorf("failed to get pane scrollback: %w", err)
	}
	return output, nil
}

// Streaming operations

func (s *execTmuxService) StreamOutput(ctx context.Context, sessionName string) (io.ReadCloser, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	// Create a pipe for streaming output
	pr, pw := io.Pipe()

	// Start streaming in background
	go func() {
		defer pw.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Capture current pane output
				output, err := s.CapturePane(ctx, sanitizedName, "0")
				if err != nil {
					pw.CloseWithError(err)
					return
				}
				pw.Write([]byte(output))
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return pr, nil
}

func (s *execTmuxService) StreamPaneOutput(ctx context.Context, sessionName, paneID string) (io.ReadCloser, error) {
	// Similar to StreamOutput but for specific pane
	return s.StreamOutput(ctx, sessionName)
}

// Configuration and utilities

func (s *execTmuxService) SetOption(ctx context.Context, sessionName, option, value string) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	if _, err := s.runTmuxCommand(ctx, "set-option", "-t", sanitizedName, option, value); err != nil {
		return fmt.Errorf("failed to set option: %w", err)
	}
	return nil
}

func (s *execTmuxService) GetOption(ctx context.Context, sessionName, option string) (string, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	output, err := s.runTmuxCommand(ctx, "show-options", "-t", sanitizedName, "-v", option)
	if err != nil {
		return "", fmt.Errorf("failed to get option: %w", err)
	}
	return strings.TrimSpace(output), nil
}

func (s *execTmuxService) ResizeSession(ctx context.Context, sessionName string, width, height int) error {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	// Set window size
	if _, err := s.runTmuxCommand(ctx, "resize-window", "-t", sanitizedName, "-x", strconv.Itoa(width), "-y", strconv.Itoa(height)); err != nil {
		return fmt.Errorf("failed to resize session: %w", err)
	}
	return nil
}

func (s *execTmuxService) HasActivity(ctx context.Context, sessionName string) (bool, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	// Check for window activity
	output, err := s.runTmuxCommand(ctx, "list-windows", "-t", sanitizedName, "-F", "#{window_activity}")
	if err != nil {
		return false, fmt.Errorf("failed to check activity: %w", err)
	}

	// If any window has recent activity
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line != "" && line != "0" {
			return true, nil
		}
	}

	return false, nil
}

func (s *execTmuxService) GetSessionPID(ctx context.Context, sessionName string) (int, error) {
	sanitizedName := s.sanitizeTmuxName(sessionName)

	output, err := s.runTmuxCommand(ctx, "list-panes", "-t", sanitizedName, "-F", "#{pane_pid}")
	if err != nil {
		return 0, fmt.Errorf("failed to get session PID: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 && lines[0] != "" {
		pid, err := strconv.Atoi(lines[0])
		if err != nil {
			return 0, fmt.Errorf("failed to parse PID: %w", err)
		}
		return pid, nil
	}

	return 0, fmt.Errorf("no PID found for session")
}

// Cleanup operations

func (s *execTmuxService) CleanupSessions(ctx context.Context, prefix string) error {
	sessions, err := s.ListSessions(ctx)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if strings.HasPrefix(session.Name, prefix) {
			_ = s.KillSession(ctx, session.Name)
		}
	}

	return nil
}

func (s *execTmuxService) CleanupOrphanedSessions(ctx context.Context) error {
	// Kill all sessions with the claudesquad prefix
	return s.CleanupSessions(ctx, tmuxPrefix)
}