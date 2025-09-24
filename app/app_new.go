package app

import (
	"claude-squad/keys"
	"claude-squad/log"
	"claude-squad/services/session"
	"claude-squad/ui"
	"claude-squad/ui/overlay"
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const GlobalSessionLimit = 10

// RunNew is the main entrypoint into the application using new services
func RunNew(ctx context.Context, program string, autoYes bool) error {
	deps, err := InitializeDependencies()
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	defer deps.Cleanup()

	p := tea.NewProgram(
		newHomeV2(ctx, deps, program, autoYes),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err = p.Run()
	return err
}

type appState int

const (
	appStateDefault appState = iota
	appStateNew
	appStatePrompt
	appStateHelp
	appStateConfirm
)

// homeV2 is the new version using service architecture
type homeV2 struct {
	ctx  context.Context
	deps *Dependencies

	// Configuration
	program string
	autoYes bool

	// Application state
	state                appState
	newSessionFinalizer  func()
	promptAfterName      bool
	keySent              bool
	sessions             []*session.Session

	// UI Components
	list                *ui.List
	menu                *ui.Menu
	tabbedWindow        *ui.TabbedWindow
	errBox              *ui.ErrBox
	spinner             spinner.Model
	textInputOverlay    *overlay.TextInputOverlay
	textOverlay         *overlay.TextOverlay
	confirmationOverlay *overlay.ConfirmationOverlay
}

func newHomeV2(ctx context.Context, deps *Dependencies, program string, autoYes bool) *homeV2 {
	h := &homeV2{
		ctx:          ctx,
		deps:         deps,
		program:      program,
		autoYes:      autoYes,
		state:        appStateDefault,
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:         ui.NewMenu(),
		tabbedWindow: ui.NewTabbedWindow(ui.NewPreviewPane(), ui.NewDiffPane()),
		errBox:       ui.NewErrBox(),
	}
	h.list = ui.NewList(&h.spinner, autoYes)

	// Load existing sessions
	sessions, err := deps.Orchestrator.ListSessions(ctx)
	if err != nil {
		log.ErrorLog.Printf("Failed to load sessions: %v", err)
	} else {
		h.sessions = sessions
		// Add sessions to UI list
		for _, sess := range sessions {
			h.addSessionToList(sess)
		}
	}

	return h
}

func (h *homeV2) addSessionToList(sess *session.Session) {
	// Convert new session to old instance format temporarily
	// This will be removed when UI is updated to use sessions directly
	instance := h.sessionToInstance(sess)
	finalizer := h.list.AddInstance(instance)
	finalizer()
}

func (h *homeV2) sessionToInstance(sess *session.Session) *session.Instance {
	// Temporary conversion function
	return &session.Instance{
		Title:     sess.Title,
		Path:      sess.Path,
		Branch:    sess.Branch,
		Status:    session.Status(sess.Status),
		Program:   sess.Program,
		Height:    sess.Height,
		Width:     sess.Width,
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
		AutoYes:   sess.AutoYes,
		Prompt:    sess.Prompt,
	}
}

func (h *homeV2) Init() tea.Cmd {
	return tea.Batch(
		h.spinner.Tick,
		func() tea.Msg {
			time.Sleep(100 * time.Millisecond)
			return previewTickMsg{}
		},
		tickUpdateMetadataCmd,
	)
}

func (h *homeV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle global keys first
	switch msg := msg.(type) {
	case tea.KeyMsg:
		h.keySent = true

		// Handle overlay states
		if h.textInputOverlay != nil {
			return h.updateTextInputOverlay(msg)
		}
		if h.textOverlay != nil {
			return h.updateTextOverlay(msg)
		}
		if h.confirmationOverlay != nil {
			return h.updateConfirmationOverlay(msg)
		}

		// Handle app states
		switch h.state {
		case appStateNew:
			return h.handleNewSessionKeys(msg)
		case appStatePrompt:
			return h.handlePromptKeys(msg)
		case appStateHelp:
			return h.handleHelpKeys(msg)
		}

		// Default key handling
		switch msg.String() {
		case "ctrl+c", "q":
			if h.state == appStateDefault {
				return h, tea.Quit
			}
		case "n":
			if h.state == appStateDefault && len(h.sessions) < GlobalSessionLimit {
				h.startNewSession()
			}
		case "enter":
			if h.state == appStateDefault {
				h.attachToSelectedSession()
			}
		case "d":
			if h.state == appStateDefault {
				h.deleteSelectedSession()
			}
		case "p":
			if h.state == appStateDefault {
				h.pauseSelectedSession()
			}
		case "r":
			if h.state == appStateDefault {
				h.resumeSelectedSession()
			}
		case "?":
			h.showHelp()
		}

	case tea.WindowSizeMsg:
		h.updateHandleWindowSizeEvent(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		h.spinner, cmd = h.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case previewTickMsg:
		h.updatePreviews()
		cmds = append(cmds, func() tea.Msg {
			time.Sleep(1 * time.Second)
			return previewTickMsg{}
		})

	case tickUpdateMetadataMsg:
		h.updateMetadata()
		cmds = append(cmds, tickUpdateMetadataCmd)

	case sessionCreatedMsg:
		h.handleSessionCreated(msg)

	case errorMsg:
		h.errBox.SetError(msg.Error())
		cmds = append(cmds, func() tea.Msg {
			time.Sleep(5 * time.Second)
			return clearErrorMsg{}
		})

	case clearErrorMsg:
		h.errBox.ClearError()
	}

	// Update components
	list, cmd := h.list.Update(msg)
	h.list = list.(*ui.List)
	cmds = append(cmds, cmd)

	menu, cmd := h.menu.Update(msg)
	h.menu = menu.(*ui.Menu)
	cmds = append(cmds, cmd)

	tabbed, cmd := h.tabbedWindow.Update(msg)
	h.tabbedWindow = tabbed.(*ui.TabbedWindow)
	cmds = append(cmds, cmd)

	return h, tea.Batch(cmds...)
}

func (h *homeV2) View() string {
	// Build the main layout
	listView := h.list.View()
	tabbedView := h.tabbedWindow.View()
	menuView := h.menu.View()
	errorView := h.errBox.View()

	// Combine views horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		listView,
		tabbedView,
	)

	// Stack vertically
	fullView := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		menuView,
		errorView,
	)

	// Add overlays if active
	if h.textInputOverlay != nil {
		return h.textInputOverlay.View()
	}
	if h.textOverlay != nil {
		return h.textOverlay.View()
	}
	if h.confirmationOverlay != nil {
		return h.confirmationOverlay.View()
	}

	return fullView
}

// Session management methods

func (h *homeV2) startNewSession() {
	h.state = appStateNew
	h.textInputOverlay = overlay.NewTextInputOverlay(
		"New Session",
		"Enter session name:",
		"",
		func(value string) {
			h.createSession(value)
		},
		func() {
			h.state = appStateDefault
			h.textInputOverlay = nil
		},
	)
}

func (h *homeV2) createSession(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Get current directory for path
		path := "."

		req := session.CreateSessionRequest{
			Title:   name,
			Path:    path,
			Program: h.program,
			AutoYes: h.autoYes,
			Height:  24,
			Width:   80,
		}

		sess, err := h.deps.Orchestrator.CreateSession(ctx, req)
		if err != nil {
			return errorMsg{err}
		}

		return sessionCreatedMsg{session: sess}
	}
}

func (h *homeV2) handleSessionCreated(msg sessionCreatedMsg) {
	h.sessions = append(h.sessions, msg.session)
	h.addSessionToList(msg.session)
	h.state = appStateDefault
	h.textInputOverlay = nil
}

func (h *homeV2) attachToSelectedSession() {
	selected := h.list.SelectedInstance()
	if selected == nil {
		return
	}

	// Find matching session
	for _, sess := range h.sessions {
		if sess.Title == selected.Title {
			go func() {
				ctx := context.Background()
				if err := h.deps.Orchestrator.AttachSession(ctx, sess.ID); err != nil {
					log.ErrorLog.Printf("Failed to attach session: %v", err)
				}
			}()
			break
		}
	}
}

func (h *homeV2) deleteSelectedSession() {
	selected := h.list.SelectedInstance()
	if selected == nil {
		return
	}

	h.confirmationOverlay = overlay.NewConfirmationOverlay(
		"Delete Session",
		fmt.Sprintf("Are you sure you want to delete '%s'?", selected.Title),
		func() {
			// Find and delete session
			for i, sess := range h.sessions {
				if sess.Title == selected.Title {
					go func() {
						ctx := context.Background()
						if err := h.deps.Orchestrator.StopSession(ctx, sess.ID); err != nil {
							log.ErrorLog.Printf("Failed to delete session: %v", err)
						}
					}()
					// Remove from local list
					h.sessions = append(h.sessions[:i], h.sessions[i+1:]...)
					h.list.RemoveSelectedInstance()
					break
				}
			}
			h.confirmationOverlay = nil
		},
		func() {
			h.confirmationOverlay = nil
		},
	)
}

func (h *homeV2) pauseSelectedSession() {
	selected := h.list.SelectedInstance()
	if selected == nil {
		return
	}

	for _, sess := range h.sessions {
		if sess.Title == selected.Title {
			go func() {
				ctx := context.Background()
				if err := h.deps.Orchestrator.PauseSession(ctx, sess.ID); err != nil {
					log.ErrorLog.Printf("Failed to pause session: %v", err)
				}
				// Update status
				sess.Status = session.StatusPaused
			}()
			break
		}
	}
}

func (h *homeV2) resumeSelectedSession() {
	selected := h.list.SelectedInstance()
	if selected == nil {
		return
	}

	for _, sess := range h.sessions {
		if sess.Title == selected.Title && sess.Status == session.StatusPaused {
			go func() {
				ctx := context.Background()
				if err := h.deps.Orchestrator.ResumeSession(ctx, sess.ID); err != nil {
					log.ErrorLog.Printf("Failed to resume session: %v", err)
				}
				// Update status
				sess.Status = session.StatusReady
			}()
			break
		}
	}
}

func (h *homeV2) showHelp() {
	h.state = appStateHelp
	h.textOverlay = overlay.NewTextOverlay(
		"Help",
		getHelpText(),
		func() {
			h.state = appStateDefault
			h.textOverlay = nil
		},
	)
}

func (h *homeV2) updatePreviews() {
	// Update preview content for selected session
	selected := h.list.SelectedInstance()
	if selected == nil {
		return
	}

	for _, sess := range h.sessions {
		if sess.Title == selected.Title {
			ctx := context.Background()
			output, err := h.deps.Orchestrator.GetOutput(ctx, sess.ID)
			if err != nil {
				log.ErrorLog.Printf("Failed to get output: %v", err)
				return
			}
			// Update preview pane
			h.tabbedWindow.SetPreviewContent(output)
			break
		}
	}
}

func (h *homeV2) updateMetadata() {
	// Update session metadata periodically
	ctx := context.Background()
	sessions, err := h.deps.Orchestrator.ListSessions(ctx)
	if err != nil {
		log.ErrorLog.Printf("Failed to update metadata: %v", err)
		return
	}
	h.sessions = sessions
}

// Helper methods for overlay handling

func (h *homeV2) updateTextInputOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	overlay, cmd := h.textInputOverlay.Update(msg)
	h.textInputOverlay = overlay.(*overlay.TextInputOverlay)
	return h, cmd
}

func (h *homeV2) updateTextOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "q" {
		h.textOverlay = nil
		h.state = appStateDefault
	}
	return h, nil
}

func (h *homeV2) updateConfirmationOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	overlay, cmd := h.confirmationOverlay.Update(msg)
	h.confirmationOverlay = overlay.(*overlay.ConfirmationOverlay)
	return h, cmd
}

func (h *homeV2) handleNewSessionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if h.textInputOverlay != nil {
		return h.updateTextInputOverlay(msg)
	}
	return h, nil
}

func (h *homeV2) handlePromptKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if h.textInputOverlay != nil {
		return h.updateTextInputOverlay(msg)
	}
	return h, nil
}

func (h *homeV2) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return h.updateTextOverlay(msg)
}

func (h *homeV2) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
	// Layout calculations
	var listWidth int
	var tabsWidth int

	if msg.Width < 100 {
		listWidth = int(float32(msg.Width) * 0.25)
	} else if msg.Width < 150 {
		listWidth = int(float32(msg.Width) * 0.28)
	} else if msg.Width < 200 {
		listWidth = int(float32(msg.Width) * 0.30)
	} else {
		listWidth = min(60, int(float32(msg.Width)*0.30))
	}

	tabsWidth = msg.Width - listWidth

	contentHeight := int(float32(msg.Height) * 0.9)
	menuHeight := msg.Height - contentHeight - 1
	h.errBox.SetSize(int(float32(msg.Width)*0.9), 1)

	h.tabbedWindow.SetSize(tabsWidth, contentHeight)
	h.list.SetSize(listWidth, contentHeight)

	if h.textInputOverlay != nil {
		h.textInputOverlay.SetSize(int(float32(msg.Width)*0.6), int(float32(msg.Height)*0.4))
	}
	if h.textOverlay != nil {
		h.textOverlay.SetWidth(int(float32(msg.Width) * 0.6))
	}

	previewWidth, previewHeight := h.tabbedWindow.GetPreviewSize()
	if err := h.list.SetSessionPreviewSize(previewWidth, previewHeight); err != nil {
		log.ErrorLog.Print(err)
	}
	h.menu.SetSize(msg.Width, menuHeight)
}

// Message types

type previewTickMsg struct{}
type tickUpdateMetadataMsg struct{}
type clearErrorMsg struct{}

type sessionCreatedMsg struct {
	session *session.Session
}

type errorMsg struct {
	error
}

var tickUpdateMetadataCmd = func() tea.Msg {
	time.Sleep(5 * time.Second)
	return tickUpdateMetadataMsg{}
}

func getHelpText() string {
	return `
Keyboard Shortcuts:

n       - Create new session
Enter   - Attach to selected session
d       - Delete selected session
p       - Pause selected session
r       - Resume paused session
Tab     - Switch between preview/diff tabs
↑/↓     - Navigate sessions
q       - Quit
?       - Show this help

Press ESC to close this help.
`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}