package executor

import (
	"context"
	"io"
	"time"
)

// Command represents a command to be executed
type Command struct {
	Program  string
	Args     []string
	Dir      string
	Env      []string
	Stdin    io.Reader
	Timeout  time.Duration
}

// Result represents the result of a command execution
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
	Error    error
}

// Output represents streaming output from a command
type Output struct {
	Type      OutputType
	Data      []byte
	Timestamp time.Time
	Error     error
}

// OutputType indicates the type of output
type OutputType int

const (
	OutputTypeStdout OutputType = iota
	OutputTypeStderr
	OutputTypeExit
	OutputTypeError
)

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	PID       int
	StartTime time.Time
	State     ProcessState
	Command   string
	Args      []string
}

// ProcessState represents the state of a process
type ProcessState int

const (
	ProcessStateRunning ProcessState = iota
	ProcessStateStopped
	ProcessStateExited
	ProcessStateZombie
)

// CommandExecutor provides command execution operations
type CommandExecutor interface {
	// Basic execution
	Execute(ctx context.Context, cmd Command) (*Result, error)
	ExecuteWithInput(ctx context.Context, cmd Command, input []byte) (*Result, error)

	// Streaming execution
	ExecuteStreaming(ctx context.Context, cmd Command) (<-chan Output, error)
	ExecuteInteractive(ctx context.Context, cmd Command) (io.ReadWriteCloser, error)

	// Process management
	Start(ctx context.Context, cmd Command) (ProcessHandle, error)
	Kill(ctx context.Context, handle ProcessHandle) error
	Signal(ctx context.Context, handle ProcessHandle, signal int) error
	Wait(ctx context.Context, handle ProcessHandle) (*Result, error)

	// Process information
	GetProcessInfo(ctx context.Context, handle ProcessHandle) (*ProcessInfo, error)
	ListProcesses(ctx context.Context) ([]*ProcessInfo, error)
	FindProcess(ctx context.Context, pid int) (ProcessHandle, error)

	// Utilities
	CommandExists(ctx context.Context, program string) bool
	Which(ctx context.Context, program string) (string, error)
	GetEnvironment(ctx context.Context) []string
	GetWorkingDirectory(ctx context.Context) (string, error)
}

// ProcessHandle represents a handle to a running process
type ProcessHandle interface {
	PID() int
	Signal(sig int) error
	Kill() error
	Wait() (*Result, error)
	State() (ProcessState, error)
}

// ExecutorOptions provides configuration for executors
type ExecutorOptions struct {
	// Default timeout for all commands
	DefaultTimeout time.Duration

	// Maximum concurrent processes
	MaxConcurrent int

	// Whether to capture output by default
	CaptureOutput bool

	// Default environment variables
	DefaultEnv []string

	// Working directory for commands
	WorkingDir string

	// Logger for debugging
	Logger Logger

	// Retry configuration
	RetryCount    int
	RetryDelay    time.Duration
	RetryOnErrors []int // Exit codes to retry on
}

// Logger provides logging for executor operations
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// NewExecutor creates a new command executor with the given options
func NewExecutor(opts *ExecutorOptions) CommandExecutor {
	// Implementation will be provided in the concrete implementation file
	return nil
}

// MockExecutor provides a mock implementation for testing
type MockExecutor struct {
	ExecuteFunc          func(ctx context.Context, cmd Command) (*Result, error)
	ExecuteStreamingFunc func(ctx context.Context, cmd Command) (<-chan Output, error)
	StartFunc            func(ctx context.Context, cmd Command) (ProcessHandle, error)
	CommandExistsFunc    func(ctx context.Context, program string) bool
}

func (m *MockExecutor) Execute(ctx context.Context, cmd Command) (*Result, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, cmd)
	}
	return &Result{ExitCode: 0}, nil
}

func (m *MockExecutor) ExecuteWithInput(ctx context.Context, cmd Command, input []byte) (*Result, error) {
	cmd.Stdin = io.NopCloser(io.ByteReader(input[0]))
	return m.Execute(ctx, cmd)
}

func (m *MockExecutor) ExecuteStreaming(ctx context.Context, cmd Command) (<-chan Output, error) {
	if m.ExecuteStreamingFunc != nil {
		return m.ExecuteStreamingFunc(ctx, cmd)
	}
	ch := make(chan Output)
	close(ch)
	return ch, nil
}

func (m *MockExecutor) ExecuteInteractive(ctx context.Context, cmd Command) (io.ReadWriteCloser, error) {
	return nil, nil
}

func (m *MockExecutor) Start(ctx context.Context, cmd Command) (ProcessHandle, error) {
	if m.StartFunc != nil {
		return m.StartFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockExecutor) Kill(ctx context.Context, handle ProcessHandle) error {
	return handle.Kill()
}

func (m *MockExecutor) Signal(ctx context.Context, handle ProcessHandle, signal int) error {
	return handle.Signal(signal)
}

func (m *MockExecutor) Wait(ctx context.Context, handle ProcessHandle) (*Result, error) {
	return handle.Wait()
}

func (m *MockExecutor) GetProcessInfo(ctx context.Context, handle ProcessHandle) (*ProcessInfo, error) {
	return &ProcessInfo{PID: handle.PID()}, nil
}

func (m *MockExecutor) ListProcesses(ctx context.Context) ([]*ProcessInfo, error) {
	return []*ProcessInfo{}, nil
}

func (m *MockExecutor) FindProcess(ctx context.Context, pid int) (ProcessHandle, error) {
	return nil, nil
}

func (m *MockExecutor) CommandExists(ctx context.Context, program string) bool {
	if m.CommandExistsFunc != nil {
		return m.CommandExistsFunc(ctx, program)
	}
	return true
}

func (m *MockExecutor) Which(ctx context.Context, program string) (string, error) {
	return "/usr/bin/" + program, nil
}

func (m *MockExecutor) GetEnvironment(ctx context.Context) []string {
	return []string{}
}

func (m *MockExecutor) GetWorkingDirectory(ctx context.Context) (string, error) {
	return "/tmp", nil
}