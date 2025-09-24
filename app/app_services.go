package app

import (
	"claude-squad/config"
	"claude-squad/keys"
	"claude-squad/log"
	"claude-squad/services/adapter"
	"claude-squad/services/session"
	"claude-squad/services/types"
	"claude-squad/ui"
	"claude-squad/ui/overlay"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RunWithServices is the main entrypoint using the new service architecture
func RunWithServices(ctx context.Context, program string, autoYes bool) error {
	deps, err := InitializeDependencies()
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}
	defer deps.Cleanup()

	p := tea.NewProgram(
		newHomeWithServices(ctx, deps, program, autoYes),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err = p.Run()
	return err
}

// homeWithServices uses the new service architecture
type homeWithServices struct {
	ctx  context.Context
	deps *Dependencies

	// Configuration
	program string
	autoYes bool

	// Application config and state
	appConfig *config.Config
	appState  config.AppState

	// Application state
	state                state
	newInstanceFinalizer func()
	promptAfterName      bool
	keySent              bool

	// UI Components (reusing existing ones)
	list                *ui.List
	menu                *ui.Menu
	tabbedWindow        *ui.TabbedWindow
	errBox              *ui.ErrBox
	spinner             spinner.Model
	textInputOverlay    *overlay.TextInputOverlay
	textOverlay         *overlay.TextOverlay
	confirmationOverlay *overlay.ConfirmationOverlay

	// Adapter instances for UI compatibility
	instances map[string]*adapter.SessionInstance
}

func newHomeWithServices(ctx context.Context, deps *Dependencies, program string, autoYes bool) *homeWithServices {
	// Load application config
	appConfig := config.LoadConfig()
	appState := config.LoadState()

	h := &homeWithServices{
		ctx:       ctx,
		deps:      deps,
		program:   program,
		autoYes:   autoYes,
		appConfig: appConfig,
		appState:  appState,
		state:     stateDefault,
		spinner:   spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:      ui.NewMenu(),
		tabbedWindow: ui.NewTabbedWindow(ui.NewPreviewPane(), ui.NewDiffPane()),
		errBox:    ui.NewErrBox(),
		instances: make(map[string]*adapter.SessionInstance),
	}
	h.list = ui.NewList(&h.spinner, autoYes)

	// Load existing sessions from orchestrator
	sessions, err := deps.Orchestrator.ListSessions(ctx)
	if err != nil {
		log.ErrorLog.Printf("Failed to load sessions: %v", err)
	} else {
		// Convert sessions to adapter instances for UI compatibility
		for _, sess := range sessions {
			instance := adapter.NewSessionInstance(sess, deps.Orchestrator)
			h.instances[sess.ID] = instance
			// Add to UI list
			finalizer := h.list.AddInstance(instance)
			finalizer() // Call immediately for loaded sessions
		}
	}

	return h
}

// createNewSession creates a new session through the orchestrator
func (h *homeWithServices) createNewSession(title string) (*adapter.SessionInstance, error) {
	req := types.CreateSessionRequest{
		Title:   title,
		Path:    ".",
		Program: h.program,
		AutoYes: h.autoYes,
		Height:  24,
		Width:   80,
	}

	sess, err := h.deps.Orchestrator.CreateSession(h.ctx, req)
	if err != nil {
		return nil, err
	}

	// Create adapter instance
	instance := adapter.NewSessionInstance(sess, h.deps.Orchestrator)
	h.instances[sess.ID] = instance
	return instance, nil
}

// The rest of the methods would be similar to the original home struct,
// but using the adapter instances instead of direct session.Instance

func (h *homeWithServices) Init() tea.Cmd {
	return tea.Batch(
		h.spinner.Tick,
		func() tea.Msg {
			time.Sleep(100 * time.Millisecond)
			return previewTickMsg{}
		},
		tickUpdateMetadataCmd,
	)
}

func (h *homeWithServices) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// This would be similar to the original Update method
	// but would use the service architecture through the adapters
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return h.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		h.updateHandleWindowSizeEvent(msg)
		return h, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		h.spinner, cmd = h.spinner.Update(msg)
		return h, cmd
	default:
		return h, nil
	}
}

func (h *homeWithServices) View() string {
	listWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(h.list.String())
	previewWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(h.tabbedWindow.String())
	listAndPreview := lipgloss.JoinHorizontal(lipgloss.Top, listWithPadding, previewWithPadding)

	mainView := lipgloss.JoinVertical(
		lipgloss.Center,
		listAndPreview,
		h.menu.String(),
		h.errBox.String(),
	)

	if h.state == statePrompt && h.textInputOverlay != nil {
		return overlay.PlaceOverlay(0, 0, h.textInputOverlay.Render(), mainView, true, true)
	} else if h.state == stateHelp && h.textOverlay != nil {
		return overlay.PlaceOverlay(0, 0, h.textOverlay.Render(), mainView, true, true)
	} else if h.state == stateConfirm && h.confirmationOverlay != nil {
		return overlay.PlaceOverlay(0, 0, h.confirmationOverlay.Render(), mainView, true, true)
	}

	return mainView
}

func (h *homeWithServices) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle quit
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		return h, tea.Quit
	}

	// Handle new session
	if msg.String() == "n" {
		if h.state == stateDefault {
			// Start creating a new session
			h.state = stateNew
			h.menu.SetState(ui.StateNewInstance)

			// Create a temporary instance for name entry
			tempInstance, err := h.createNewSession("")
			if err != nil {
				h.errBox.SetError(err)
				return h, nil
			}

			h.newInstanceFinalizer = h.list.AddInstance(tempInstance)
			h.list.SetSelectedInstance(h.list.NumInstances() - 1)
		}
		return h, nil
	}

	return h, nil
}

func (h *homeWithServices) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
	// Same layout logic as original
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}