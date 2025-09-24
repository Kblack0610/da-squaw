package tmux

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// MockTmuxService is a mock implementation of TmuxService for testing
type MockTmuxService struct {
	// Session management mocks
	CreateSessionFunc     func(ctx context.Context, name, startDir, command string) (*Session, error)
	AttachSessionFunc     func(ctx context.Context, sessionName string) error
	DetachSessionFunc     func(ctx context.Context, sessionName string) error
	KillSessionFunc       func(ctx context.Context, sessionName string) error
	ListSessionsFunc      func(ctx context.Context) ([]*Session, error)
	GetSessionFunc        func(ctx context.Context, sessionName string) (*Session, error)
	RenameSessionFunc     func(ctx context.Context, oldName, newName string) error
	SessionExistsFunc     func(ctx context.Context, sessionName string) (bool, error)

	// Window management mocks
	CreateWindowFunc  func(ctx context.Context, sessionName, windowName, command string) (*Window, error)
	KillWindowFunc    func(ctx context.Context, sessionName, windowID string) error
	ListWindowsFunc   func(ctx context.Context, sessionName string) ([]*Window, error)
	RenameWindowFunc  func(ctx context.Context, sessionName, windowID, newName string) error
	SelectWindowFunc  func(ctx context.Context, sessionName, windowID string) error

	// Pane management mocks
	SplitPaneFunc  func(ctx context.Context, sessionName, windowID string, vertical bool, command string) (*Pane, error)
	KillPaneFunc   func(ctx context.Context, sessionName, paneID string) error
	ListPanesFunc  func(ctx context.Context, sessionName, windowID string) ([]*Pane, error)
	ResizePaneFunc func(ctx context.Context, sessionName, paneID string, width, height int) error
	SelectPaneFunc func(ctx context.Context, sessionName, paneID string) error

	// I/O mocks
	SendKeysFunc         func(ctx context.Context, sessionName string, keys string) error
	SendKeysToPaneFunc   func(ctx context.Context, sessionName, paneID, keys string) error
	CapturePaneFunc      func(ctx context.Context, sessionName, paneID string) (string, error)
	GetPaneOutputFunc    func(ctx context.Context, sessionName, paneID string, lines int) (string, error)
	GetPaneScrollbackFunc func(ctx context.Context, sessionName, paneID string) (string, error)

	// Streaming mocks
	StreamOutputFunc     func(ctx context.Context, sessionName string) (io.ReadCloser, error)
	StreamPaneOutputFunc func(ctx context.Context, sessionName, paneID string) (io.ReadCloser, error)

	// Configuration mocks
	SetOptionFunc       func(ctx context.Context, sessionName, option, value string) error
	GetOptionFunc       func(ctx context.Context, sessionName, option string) (string, error)
	ResizeSessionFunc   func(ctx context.Context, sessionName string, width, height int) error
	HasActivityFunc     func(ctx context.Context, sessionName string) (bool, error)
	GetSessionPIDFunc   func(ctx context.Context, sessionName string) (int, error)

	// Cleanup mocks
	CleanupSessionsFunc         func(ctx context.Context, prefix string) error
	CleanupOrphanedSessionsFunc func(ctx context.Context) error

	// Default data
	Sessions map[string]*Session
	Windows  map[string][]*Window
	Panes    map[string][]*Pane
	Output   map[string]string
}

// NewMockTmuxService creates a new mock with sensible defaults
func NewMockTmuxService() *MockTmuxService {
	return &MockTmuxService{
		Sessions: make(map[string]*Session),
		Windows:  make(map[string][]*Window),
		Panes:    make(map[string][]*Pane),
		Output:   make(map[string]string),
	}
}

func (m *MockTmuxService) CreateSession(ctx context.Context, name, startDir, command string) (*Session, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, name, startDir, command)
	}

	session := &Session{
		Name:      name,
		ID:        name,
		Windows:   1,
		Created:   "now",
		Attached:  false,
		Width:     80,
		Height:    24,
		Directory: startDir,
	}
	m.Sessions[name] = session
	return session, nil
}

func (m *MockTmuxService) AttachSession(ctx context.Context, sessionName string) error {
	if m.AttachSessionFunc != nil {
		return m.AttachSessionFunc(ctx, sessionName)
	}
	if sess, ok := m.Sessions[sessionName]; ok {
		sess.Attached = true
		return nil
	}
	return fmt.Errorf("session not found: %s", sessionName)
}

func (m *MockTmuxService) DetachSession(ctx context.Context, sessionName string) error {
	if m.DetachSessionFunc != nil {
		return m.DetachSessionFunc(ctx, sessionName)
	}
	if sess, ok := m.Sessions[sessionName]; ok {
		sess.Attached = false
		return nil
	}
	return fmt.Errorf("session not found: %s", sessionName)
}

func (m *MockTmuxService) KillSession(ctx context.Context, sessionName string) error {
	if m.KillSessionFunc != nil {
		return m.KillSessionFunc(ctx, sessionName)
	}
	delete(m.Sessions, sessionName)
	return nil
}

func (m *MockTmuxService) ListSessions(ctx context.Context) ([]*Session, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc(ctx)
	}
	sessions := make([]*Session, 0, len(m.Sessions))
	for _, sess := range m.Sessions {
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (m *MockTmuxService) GetSession(ctx context.Context, sessionName string) (*Session, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(ctx, sessionName)
	}
	if sess, ok := m.Sessions[sessionName]; ok {
		return sess, nil
	}
	return nil, fmt.Errorf("session not found: %s", sessionName)
}

func (m *MockTmuxService) RenameSession(ctx context.Context, oldName, newName string) error {
	if m.RenameSessionFunc != nil {
		return m.RenameSessionFunc(ctx, oldName, newName)
	}
	if sess, ok := m.Sessions[oldName]; ok {
		sess.Name = newName
		sess.ID = newName
		m.Sessions[newName] = sess
		delete(m.Sessions, oldName)
		return nil
	}
	return fmt.Errorf("session not found: %s", oldName)
}

func (m *MockTmuxService) SessionExists(ctx context.Context, sessionName string) (bool, error) {
	if m.SessionExistsFunc != nil {
		return m.SessionExistsFunc(ctx, sessionName)
	}
	_, exists := m.Sessions[sessionName]
	return exists, nil
}

func (m *MockTmuxService) CreateWindow(ctx context.Context, sessionName, windowName, command string) (*Window, error) {
	if m.CreateWindowFunc != nil {
		return m.CreateWindowFunc(ctx, sessionName, windowName, command)
	}
	window := &Window{
		ID:     windowName,
		Name:   windowName,
		Active: false,
		Panes:  1,
	}
	m.Windows[sessionName] = append(m.Windows[sessionName], window)
	return window, nil
}

func (m *MockTmuxService) KillWindow(ctx context.Context, sessionName, windowID string) error {
	if m.KillWindowFunc != nil {
		return m.KillWindowFunc(ctx, sessionName, windowID)
	}
	return nil
}

func (m *MockTmuxService) ListWindows(ctx context.Context, sessionName string) ([]*Window, error) {
	if m.ListWindowsFunc != nil {
		return m.ListWindowsFunc(ctx, sessionName)
	}
	return m.Windows[sessionName], nil
}

func (m *MockTmuxService) RenameWindow(ctx context.Context, sessionName, windowID, newName string) error {
	if m.RenameWindowFunc != nil {
		return m.RenameWindowFunc(ctx, sessionName, windowID, newName)
	}
	return nil
}

func (m *MockTmuxService) SelectWindow(ctx context.Context, sessionName, windowID string) error {
	if m.SelectWindowFunc != nil {
		return m.SelectWindowFunc(ctx, sessionName, windowID)
	}
	return nil
}

func (m *MockTmuxService) SplitPane(ctx context.Context, sessionName, windowID string, vertical bool, command string) (*Pane, error) {
	if m.SplitPaneFunc != nil {
		return m.SplitPaneFunc(ctx, sessionName, windowID, vertical, command)
	}
	pane := &Pane{
		ID:      fmt.Sprintf("pane%d", len(m.Panes[sessionName])),
		Active:  true,
		Width:   40,
		Height:  24,
		Command: command,
	}
	m.Panes[sessionName] = append(m.Panes[sessionName], pane)
	return pane, nil
}

func (m *MockTmuxService) KillPane(ctx context.Context, sessionName, paneID string) error {
	if m.KillPaneFunc != nil {
		return m.KillPaneFunc(ctx, sessionName, paneID)
	}
	return nil
}

func (m *MockTmuxService) ListPanes(ctx context.Context, sessionName, windowID string) ([]*Pane, error) {
	if m.ListPanesFunc != nil {
		return m.ListPanesFunc(ctx, sessionName, windowID)
	}
	return m.Panes[sessionName], nil
}

func (m *MockTmuxService) ResizePane(ctx context.Context, sessionName, paneID string, width, height int) error {
	if m.ResizePaneFunc != nil {
		return m.ResizePaneFunc(ctx, sessionName, paneID, width, height)
	}
	return nil
}

func (m *MockTmuxService) SelectPane(ctx context.Context, sessionName, paneID string) error {
	if m.SelectPaneFunc != nil {
		return m.SelectPaneFunc(ctx, sessionName, paneID)
	}
	return nil
}

func (m *MockTmuxService) SendKeys(ctx context.Context, sessionName string, keys string) error {
	if m.SendKeysFunc != nil {
		return m.SendKeysFunc(ctx, sessionName, keys)
	}
	m.Output[sessionName] += keys
	return nil
}

func (m *MockTmuxService) SendKeysToPane(ctx context.Context, sessionName, paneID, keys string) error {
	if m.SendKeysToPaneFunc != nil {
		return m.SendKeysToPaneFunc(ctx, sessionName, paneID, keys)
	}
	m.Output[sessionName+":"+paneID] += keys
	return nil
}

func (m *MockTmuxService) CapturePane(ctx context.Context, sessionName, paneID string) (string, error) {
	if m.CapturePaneFunc != nil {
		return m.CapturePaneFunc(ctx, sessionName, paneID)
	}
	return m.Output[sessionName], nil
}

func (m *MockTmuxService) GetPaneOutput(ctx context.Context, sessionName, paneID string, lines int) (string, error) {
	if m.GetPaneOutputFunc != nil {
		return m.GetPaneOutputFunc(ctx, sessionName, paneID, lines)
	}
	output := m.Output[sessionName]
	if lines > 0 {
		outputLines := strings.Split(output, "\n")
		if len(outputLines) > lines {
			outputLines = outputLines[len(outputLines)-lines:]
		}
		output = strings.Join(outputLines, "\n")
	}
	return output, nil
}

func (m *MockTmuxService) GetPaneScrollback(ctx context.Context, sessionName, paneID string) (string, error) {
	if m.GetPaneScrollbackFunc != nil {
		return m.GetPaneScrollbackFunc(ctx, sessionName, paneID)
	}
	return m.Output[sessionName], nil
}

func (m *MockTmuxService) StreamOutput(ctx context.Context, sessionName string) (io.ReadCloser, error) {
	if m.StreamOutputFunc != nil {
		return m.StreamOutputFunc(ctx, sessionName)
	}
	return io.NopCloser(strings.NewReader(m.Output[sessionName])), nil
}

func (m *MockTmuxService) StreamPaneOutput(ctx context.Context, sessionName, paneID string) (io.ReadCloser, error) {
	if m.StreamPaneOutputFunc != nil {
		return m.StreamPaneOutputFunc(ctx, sessionName, paneID)
	}
	return io.NopCloser(strings.NewReader(m.Output[sessionName+":"+paneID])), nil
}

func (m *MockTmuxService) SetOption(ctx context.Context, sessionName, option, value string) error {
	if m.SetOptionFunc != nil {
		return m.SetOptionFunc(ctx, sessionName, option, value)
	}
	return nil
}

func (m *MockTmuxService) GetOption(ctx context.Context, sessionName, option string) (string, error) {
	if m.GetOptionFunc != nil {
		return m.GetOptionFunc(ctx, sessionName, option)
	}
	return "", nil
}

func (m *MockTmuxService) ResizeSession(ctx context.Context, sessionName string, width, height int) error {
	if m.ResizeSessionFunc != nil {
		return m.ResizeSessionFunc(ctx, sessionName, width, height)
	}
	if sess, ok := m.Sessions[sessionName]; ok {
		sess.Width = width
		sess.Height = height
	}
	return nil
}

func (m *MockTmuxService) HasActivity(ctx context.Context, sessionName string) (bool, error) {
	if m.HasActivityFunc != nil {
		return m.HasActivityFunc(ctx, sessionName)
	}
	return false, nil
}

func (m *MockTmuxService) GetSessionPID(ctx context.Context, sessionName string) (int, error) {
	if m.GetSessionPIDFunc != nil {
		return m.GetSessionPIDFunc(ctx, sessionName)
	}
	return 12345, nil
}

func (m *MockTmuxService) CleanupSessions(ctx context.Context, prefix string) error {
	if m.CleanupSessionsFunc != nil {
		return m.CleanupSessionsFunc(ctx, prefix)
	}
	for name := range m.Sessions {
		if strings.HasPrefix(name, prefix) {
			delete(m.Sessions, name)
		}
	}
	return nil
}

func (m *MockTmuxService) CleanupOrphanedSessions(ctx context.Context) error {
	if m.CleanupOrphanedSessionsFunc != nil {
		return m.CleanupOrphanedSessionsFunc(ctx)
	}
	return nil
}