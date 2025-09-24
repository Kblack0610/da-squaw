package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// execImpl is the concrete implementation of CommandExecutor
type execImpl struct {
	opts           *ExecutorOptions
	runningProcs   map[ProcessHandle]*processInfo
	procMutex      sync.RWMutex
	concurrentSem  chan struct{}
}

// processInfo holds information about a running process
type processInfo struct {
	cmd       *exec.Cmd
	startTime time.Time
	state     ProcessState
}

// processHandleImpl implements ProcessHandle
type processHandleImpl struct {
	cmd      *exec.Cmd
	info     *processInfo
	executor *execImpl
}

// NewExecutor creates a new command executor with the given options
func NewDefaultExecutor() CommandExecutor {
	return NewExecutor(&ExecutorOptions{
		DefaultTimeout: 120 * time.Second,
		MaxConcurrent:  10,
		CaptureOutput:  true,
	})
}

// NewExecutor creates a new command executor with the given options
func NewExecutor(opts *ExecutorOptions) CommandExecutor {
	if opts == nil {
		opts = &ExecutorOptions{
			DefaultTimeout: 120 * time.Second,
			MaxConcurrent:  10,
			CaptureOutput:  true,
		}
	}

	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 10
	}

	return &execImpl{
		opts:          opts,
		runningProcs:  make(map[ProcessHandle]*processInfo),
		concurrentSem: make(chan struct{}, opts.MaxConcurrent),
	}
}

// Basic execution

func (e *execImpl) Execute(ctx context.Context, cmd Command) (*Result, error) {
	// Acquire semaphore
	select {
	case e.concurrentSem <- struct{}{}:
		defer func() { <-e.concurrentSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Set timeout if not specified
	timeout := cmd.Timeout
	if timeout == 0 {
		timeout = e.opts.DefaultTimeout
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	execCmd := exec.CommandContext(execCtx, cmd.Program, cmd.Args...)
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	if cmd.Env != nil {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	} else if e.opts.DefaultEnv != nil {
		execCmd.Env = append(os.Environ(), e.opts.DefaultEnv...)
	}
	if e.opts.WorkingDir != "" && cmd.Dir == "" {
		execCmd.Dir = e.opts.WorkingDir
	}

	// Set up stdin
	if cmd.Stdin != nil {
		execCmd.Stdin = cmd.Stdin
	}

	// Capture output if enabled
	var stdout, stderr bytes.Buffer
	if e.opts.CaptureOutput {
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr
	}

	// Log command if logger is set
	if e.opts.Logger != nil {
		e.opts.Logger.Debug("Executing command: %s %v", cmd.Program, cmd.Args)
	}

	startTime := time.Now()

	// Execute with retry logic
	var err error
	var exitCode int
	retries := e.opts.RetryCount
	if retries < 0 {
		retries = 0
	}

	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			if e.opts.Logger != nil {
				e.opts.Logger.Info("Retrying command (attempt %d/%d)", attempt+1, retries+1)
			}
			time.Sleep(e.opts.RetryDelay)
		}

		err = execCmd.Run()
		exitCode = 0

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				} else {
					exitCode = 1
				}
			} else {
				exitCode = -1
			}

			// Check if we should retry based on exit code
			shouldRetry := false
			for _, retryCode := range e.opts.RetryOnErrors {
				if exitCode == retryCode {
					shouldRetry = true
					break
				}
			}

			if !shouldRetry || attempt == retries {
				break
			}
		} else {
			break // Success, no need to retry
		}
	}

	duration := time.Since(startTime)

	result := &Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode,
		Duration: duration,
		Error:    err,
	}

	if e.opts.Logger != nil {
		if err != nil {
			e.opts.Logger.Error("Command failed: %v (exit code: %d)", err, exitCode)
		} else {
			e.opts.Logger.Debug("Command succeeded in %v", duration)
		}
	}

	return result, nil
}

func (e *execImpl) ExecuteWithInput(ctx context.Context, cmd Command, input []byte) (*Result, error) {
	cmd.Stdin = bytes.NewReader(input)
	return e.Execute(ctx, cmd)
}

// Streaming execution

func (e *execImpl) ExecuteStreaming(ctx context.Context, cmd Command) (<-chan Output, error) {
	outputCh := make(chan Output, 100)

	// Acquire semaphore
	select {
	case e.concurrentSem <- struct{}{}:
		// Will be released when command completes
	case <-ctx.Done():
		close(outputCh)
		return outputCh, ctx.Err()
	}

	// Set timeout if not specified
	timeout := cmd.Timeout
	if timeout == 0 {
		timeout = e.opts.DefaultTimeout
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)

	// Create command
	execCmd := exec.CommandContext(execCtx, cmd.Program, cmd.Args...)
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	if cmd.Env != nil {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	// Set up pipes for stdout and stderr
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		<-e.concurrentSem
		cancel()
		close(outputCh)
		return outputCh, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		<-e.concurrentSem
		cancel()
		close(outputCh)
		return outputCh, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		<-e.concurrentSem
		cancel()
		close(outputCh)
		return outputCh, fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output in background
	go func() {
		defer func() {
			<-e.concurrentSem
			cancel()
			close(outputCh)
		}()

		var wg sync.WaitGroup
		wg.Add(2)

		// Read stdout
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := stdoutPipe.Read(buf)
				if n > 0 {
					outputCh <- Output{
						Type:      OutputTypeStdout,
						Data:      append([]byte{}, buf[:n]...),
						Timestamp: time.Now(),
					}
				}
				if err != nil {
					if err != io.EOF {
						outputCh <- Output{
							Type:      OutputTypeError,
							Error:     err,
							Timestamp: time.Now(),
						}
					}
					break
				}
			}
		}()

		// Read stderr
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := stderrPipe.Read(buf)
				if n > 0 {
					outputCh <- Output{
						Type:      OutputTypeStderr,
						Data:      append([]byte{}, buf[:n]...),
						Timestamp: time.Now(),
					}
				}
				if err != nil {
					if err != io.EOF {
						outputCh <- Output{
							Type:      OutputTypeError,
							Error:     err,
							Timestamp: time.Now(),
						}
					}
					break
				}
			}
		}()

		wg.Wait()

		// Wait for command to finish
		err := execCmd.Wait()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
			}
		}

		outputCh <- Output{
			Type:      OutputTypeExit,
			Data:      []byte(fmt.Sprintf("%d", exitCode)),
			Timestamp: time.Now(),
			Error:     err,
		}
	}()

	return outputCh, nil
}

func (e *execImpl) ExecuteInteractive(ctx context.Context, cmd Command) (io.ReadWriteCloser, error) {
	// Create command
	execCmd := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	if cmd.Env != nil {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	// Create pipes for stdin, stdout, stderr
	stdin, err := execCmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Create a combined reader/writer
	return &interactivePipe{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		cmd:    execCmd,
	}, nil
}

// Process management

func (e *execImpl) Start(ctx context.Context, cmd Command) (ProcessHandle, error) {
	// Create command
	execCmd := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}
	if cmd.Env != nil {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	info := &processInfo{
		cmd:       execCmd,
		startTime: time.Now(),
		state:     ProcessStateRunning,
	}

	handle := &processHandleImpl{
		cmd:      execCmd,
		info:     info,
		executor: e,
	}

	// Register process
	e.procMutex.Lock()
	e.runningProcs[handle] = info
	e.procMutex.Unlock()

	return handle, nil
}

func (e *execImpl) Kill(ctx context.Context, handle ProcessHandle) error {
	return handle.Kill()
}

func (e *execImpl) Signal(ctx context.Context, handle ProcessHandle, signal int) error {
	return handle.Signal(signal)
}

func (e *execImpl) Wait(ctx context.Context, handle ProcessHandle) (*Result, error) {
	return handle.Wait()
}

func (e *execImpl) GetProcessInfo(ctx context.Context, handle ProcessHandle) (*ProcessInfo, error) {
	e.procMutex.RLock()
	info, exists := e.runningProcs[handle]
	e.procMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("process not found")
	}

	return &ProcessInfo{
		PID:       info.cmd.Process.Pid,
		StartTime: info.startTime,
		State:     info.state,
		Command:   info.cmd.Path,
		Args:      info.cmd.Args,
	}, nil
}

func (e *execImpl) ListProcesses(ctx context.Context) ([]*ProcessInfo, error) {
	e.procMutex.RLock()
	defer e.procMutex.RUnlock()

	processes := make([]*ProcessInfo, 0, len(e.runningProcs))
	for _, info := range e.runningProcs {
		processes = append(processes, &ProcessInfo{
			PID:       info.cmd.Process.Pid,
			StartTime: info.startTime,
			State:     info.state,
			Command:   info.cmd.Path,
			Args:      info.cmd.Args,
		})
	}

	return processes, nil
}

func (e *execImpl) FindProcess(ctx context.Context, pid int) (ProcessHandle, error) {
	e.procMutex.RLock()
	defer e.procMutex.RUnlock()

	for handle, info := range e.runningProcs {
		if info.cmd.Process.Pid == pid {
			return handle, nil
		}
	}

	return nil, fmt.Errorf("process with PID %d not found", pid)
}

// Utilities

func (e *execImpl) CommandExists(ctx context.Context, program string) bool {
	_, err := exec.LookPath(program)
	return err == nil
}

func (e *execImpl) Which(ctx context.Context, program string) (string, error) {
	return exec.LookPath(program)
}

func (e *execImpl) GetEnvironment(ctx context.Context) []string {
	return os.Environ()
}

func (e *execImpl) GetWorkingDirectory(ctx context.Context) (string, error) {
	return os.Getwd()
}

// ProcessHandle implementation

func (h *processHandleImpl) PID() int {
	return h.cmd.Process.Pid
}

func (h *processHandleImpl) Signal(sig int) error {
	return h.cmd.Process.Signal(syscall.Signal(sig))
}

func (h *processHandleImpl) Kill() error {
	err := h.cmd.Process.Kill()

	// Update state
	h.executor.procMutex.Lock()
	if info, exists := h.executor.runningProcs[h]; exists {
		info.state = ProcessStateExited
	}
	h.executor.procMutex.Unlock()

	return err
}

func (h *processHandleImpl) Wait() (*Result, error) {
	err := h.cmd.Wait()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	// Update state and remove from running processes
	h.executor.procMutex.Lock()
	if info, exists := h.executor.runningProcs[h]; exists {
		info.state = ProcessStateExited
		delete(h.executor.runningProcs, h)
	}
	h.executor.procMutex.Unlock()

	return &Result{
		ExitCode: exitCode,
		Duration: time.Since(h.info.startTime),
		Error:    err,
	}, nil
}

func (h *processHandleImpl) State() (ProcessState, error) {
	h.executor.procMutex.RLock()
	info, exists := h.executor.runningProcs[h]
	h.executor.procMutex.RUnlock()

	if !exists {
		return ProcessStateExited, nil
	}

	return info.state, nil
}

// interactivePipe implements io.ReadWriteCloser for interactive commands
type interactivePipe struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
	mu     sync.Mutex
	closed bool
}

func (p *interactivePipe) Read(b []byte) (n int, err error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, io.EOF
	}
	p.mu.Unlock()

	// Try to read from stdout first
	n, err = p.stdout.Read(b)
	if n > 0 {
		return n, nil
	}

	// If stdout is empty, try stderr
	n, err = p.stderr.Read(b)
	return n, err
}

func (p *interactivePipe) Write(b []byte) (n int, err error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	p.mu.Unlock()

	return p.stdin.Write(b)
}

func (p *interactivePipe) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	var errs []error
	if err := p.stdin.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := p.stdout.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := p.stderr.Close(); err != nil {
		errs = append(errs, err)
	}

	// Terminate the command
	if err := p.cmd.Process.Kill(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}