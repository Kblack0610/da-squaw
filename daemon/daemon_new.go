package daemon

import (
	"claude-squad/config"
	"claude-squad/log"
	"claude-squad/services/session"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Daemon manages the background process that handles AutoYes mode for all sessions
type Daemon struct {
	orchestrator session.SessionOrchestrator
	config       *config.Config
	sessions     map[string]*session.Session
	mu           sync.RWMutex
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewDaemon creates a new daemon instance
func NewDaemon(orchestrator session.SessionOrchestrator, config *config.Config) *Daemon {
	return &Daemon{
		orchestrator: orchestrator,
		config:       config,
		sessions:     make(map[string]*session.Session),
		stopCh:       make(chan struct{}),
	}
}

// Run starts the daemon process
func (d *Daemon) Run(ctx context.Context) error {
	log.InfoLog.Printf("starting daemon")

	// Load initial sessions
	if err := d.loadSessions(ctx); err != nil {
		return fmt.Errorf("failed to load sessions: %w", err)
	}

	pollInterval := time.Duration(d.config.DaemonPollInterval) * time.Millisecond
	everyN := log.NewEvery(60 * time.Second)

	// Start monitoring goroutine
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		ticker := time.NewTimer(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-d.stopCh:
				return
			case <-ticker.C:
				d.processSessions(ctx, everyN)
				ticker.Reset(pollInterval)
			}
		}
	}()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.InfoLog.Printf("received signal %s", sig.String())
	case <-ctx.Done():
		log.InfoLog.Printf("context cancelled")
	}

	// Shutdown
	close(d.stopCh)
	d.wg.Wait()

	// Save session states
	if err := d.saveSessions(ctx); err != nil {
		log.ErrorLog.Printf("failed to save sessions: %v", err)
	}

	log.InfoLog.Printf("daemon stopped")
	return nil
}

func (d *Daemon) loadSessions(ctx context.Context) error {
	sessions, err := d.orchestrator.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, sess := range sessions {
		// Enable AutoYes for all sessions in daemon mode
		sess.AutoYes = true
		d.sessions[sess.ID] = sess
	}

	log.InfoLog.Printf("loaded %d sessions", len(d.sessions))
	return nil
}

func (d *Daemon) processSessions(ctx context.Context, everyN *log.Every) {
	d.mu.RLock()
	sessionIDs := make([]string, 0, len(d.sessions))
	for id := range d.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	d.mu.RUnlock()

	for _, id := range sessionIDs {
		d.processSession(ctx, id, everyN)
	}
}

func (d *Daemon) processSession(ctx context.Context, sessionID string, everyN *log.Every) {
	d.mu.RLock()
	sess, exists := d.sessions[sessionID]
	d.mu.RUnlock()

	if !exists {
		return
	}

	// Only process running or ready sessions
	if sess.Status != session.StatusRunning && sess.Status != session.StatusReady {
		return
	}

	// Check if session has output that needs response
	output, err := d.orchestrator.GetOutput(ctx, sessionID)
	if err != nil {
		if everyN.ShouldLog() {
			log.WarningLog.Printf("could not get output for session %s: %v", sess.Title, err)
		}
		return
	}

	// Simple heuristic: if output ends with prompt-like patterns, send Enter
	if d.shouldRespond(output) {
		if err := d.orchestrator.SendInput(ctx, sessionID, "\n"); err != nil {
			if everyN.ShouldLog() {
				log.WarningLog.Printf("could not send input to session %s: %v", sess.Title, err)
			}
		} else {
			log.InfoLog.Printf("sent auto-response to session %s", sess.Title)
		}
	}

	// Update session status
	updatedSession, err := d.orchestrator.GetSession(ctx, sessionID)
	if err == nil {
		d.mu.Lock()
		d.sessions[sessionID] = updatedSession
		d.mu.Unlock()
	}
}

func (d *Daemon) shouldRespond(output string) bool {
	// Check for common prompt patterns that indicate waiting for input
	promptPatterns := []string{
		"[Y/n]",
		"(y/N)",
		"Continue?",
		"Proceed?",
		"Press Enter",
		"press enter",
		"Hit enter",
		"hit enter",
		">>> ",
		"claude> ",
		"aider> ",
		"> ",
	}

	for _, pattern := range promptPatterns {
		if len(output) > len(pattern) {
			tail := output[len(output)-len(pattern)-10:]
			if containsPattern(tail, pattern) {
				return true
			}
		}
	}

	return false
}

func containsPattern(text, pattern string) bool {
	// Simple substring check - could be enhanced with regex
	return len(text) > 0 && len(pattern) > 0 &&
		   (text == pattern ||
		    (len(text) > len(pattern) && text[len(text)-len(pattern):] == pattern))
}

func (d *Daemon) saveSessions(ctx context.Context) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Sessions are automatically persisted by the orchestrator
	// This is a no-op but could be used for final cleanup
	log.InfoLog.Printf("saved %d sessions", len(d.sessions))
	return nil
}

// Stop gracefully stops the daemon
func (d *Daemon) Stop() {
	close(d.stopCh)
	d.wg.Wait()
}