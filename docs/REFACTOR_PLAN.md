# Da-Squaw Refactoring Plan - Phased Approach

## Overview
This document outlines a phased refactoring plan to decouple the da-squaw repository into more modular services and layers, improving testability, maintainability, and enabling future scaling.

## Phase 1: Tactical Service Extraction (Quick Wins)
**Goal**: Make existing code easier to test and reason about without changing external behavior

### 1.1 Extract Session Orchestration Service

**Current State**:
- `session.Instance` tightly couples Git, Tmux, and state management
- Business logic mixed with infrastructure concerns

**Target State**:
```go
// services/session/orchestrator.go
type SessionOrchestrator interface {
    CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error)
    StartSession(ctx context.Context, sessionID string) error
    PauseSession(ctx context.Context, sessionID string) error
    GetSessionStatus(ctx context.Context, sessionID string) (Status, error)
}

// services/session/orchestrator_impl.go
type orchestrator struct {
    gitService  GitService
    tmuxService TmuxService
    storage     StorageRepository
}
```

**Implementation Steps**:
1. Create `services/` directory structure
2. Extract session lifecycle methods from `session.Instance`
3. Move orchestration logic to new service
4. Update `app/app.go` to use orchestrator
5. Keep `session.Instance` as a data struct only

### 1.2 Create Git Service Interface

**Current State**:
- Direct `os/exec` calls throughout `session/git/`
- Tight coupling to file system

**Target State**:
```go
// services/git/service.go
type GitService interface {
    CreateWorktree(path, branch string) (*Worktree, error)
    GetDiffStats(worktree *Worktree) (*DiffStats, error)
    GetBranches(path string) ([]Branch, error)
    Commit(worktree *Worktree, message string) error
    CleanupWorktree(worktree *Worktree) error
}

// services/git/exec_adapter.go
type execGitService struct {
    executor CommandExecutor
}
```

**Implementation Steps**:
1. Define `GitService` interface
2. Create `execGitService` implementation using existing logic
3. Extract pure domain types (Branch, DiffStats, Worktree)
4. Inject GitService into SessionOrchestrator
5. Add mock implementation for testing

### 1.3 Create Tmux Service Interface

**Current State**:
- Platform-specific implementations (`tmux_unix.go`, `tmux_windows.go`)
- Direct tmux command execution

**Target State**:
```go
// services/tmux/service.go
type TmuxService interface {
    CreateSession(name string, command string) (*TmuxSession, error)
    AttachSession(session *TmuxSession) error
    SendKeys(session *TmuxSession, keys string) error
    GetOutput(session *TmuxSession) (string, error)
    KillSession(session *TmuxSession) error
}

// services/tmux/exec_adapter.go
type execTmuxService struct {
    executor CommandExecutor
    platform Platform
}
```

**Implementation Steps**:
1. Define `TmuxService` interface
2. Consolidate platform-specific logic into adapter
3. Extract `TmuxSession` as pure domain type
4. Inject TmuxService into SessionOrchestrator
5. Create mock for testing

### 1.4 Implement Storage Repository Pattern

**Current State**:
- `session.Storage` directly handles JSON file I/O
- No abstraction for different storage backends

**Target State**:
```go
// services/storage/repository.go
type StorageRepository interface {
    SaveSession(session *Session) error
    LoadSession(id string) (*Session, error)
    ListSessions() ([]*Session, error)
    DeleteSession(id string) error
    UpdateSessionStatus(id string, status Status) error
}

// services/storage/json_repository.go
type jsonRepository struct {
    basePath string
    mutex    sync.RWMutex
}
```

**Implementation Steps**:
1. Define `StorageRepository` interface
2. Refactor existing JSON storage to implement interface
3. Add in-memory implementation for testing
4. Consider adding caching layer
5. Update all consumers to use interface

### 1.5 Create Command Executor Abstraction

**Current State**:
- `cmd.Executor` exists but is underutilized
- Direct `os/exec` calls scattered throughout

**Target State**:
```go
// services/executor/executor.go
type CommandExecutor interface {
    Execute(ctx context.Context, command Command) (*Result, error)
    ExecuteStreaming(ctx context.Context, command Command) (<-chan Output, error)
}

type Command struct {
    Program string
    Args    []string
    Dir     string
    Env     []string
}

type Result struct {
    Stdout   []byte
    Stderr   []byte
    ExitCode int
}
```

**Implementation Steps**:
1. Expand existing `cmd.Executor` interface
2. Create comprehensive implementation
3. Add logging and retry capabilities
4. Replace all direct `os/exec` calls
5. Create mock for deterministic testing

## Phase 1 Testing Strategy

### Unit Tests
```go
// services/session/orchestrator_test.go
func TestOrchestratorCreateSession(t *testing.T) {
    gitMock := &MockGitService{}
    tmuxMock := &MockTmuxService{}
    storageMock := &MockStorageRepository{}

    orch := NewOrchestrator(gitMock, tmuxMock, storageMock)

    // Test session creation with mocked dependencies
}
```

### Integration Tests
```go
// tests/integration/session_test.go
func TestSessionLifecycle(t *testing.T) {
    // Test with real implementations but in isolated environment
}
```

## Phase 1 Success Criteria
- ✅ All existing functionality preserved
- ✅ Unit test coverage >80% for new services
- ✅ Integration tests pass without modification
- ✅ No changes to CLI or daemon user experience
- ✅ Reduced coupling metrics (measured by import analysis)

## Phase 2: Structural Layering (Clean Architecture)

### 2.1 Establish Domain Layer
```
internal/
  domain/
    session/
      - session.go (pure domain entity)
      - status.go (value objects)
      - rules.go (business rules)
    git/
      - repository.go (interface)
      - branch.go (entity)
      - diff.go (value object)
    tmux/
      - session.go (interface)
      - window.go (entity)
```

### 2.2 Create Application Services Layer
```
internal/
  app/
    session/
      - create_session.go (use case)
      - attach_session.go (use case)
      - list_sessions.go (use case)
    git/
      - analyze_diff.go (use case)
```

### 2.3 Implement Adapters
```
internal/
  adapter/
    gitexec/
      - repository.go (implements domain.git.Repository)
    tmuxexec/
      - session.go (implements domain.tmux.Session)
    storagejson/
      - repository.go (implements domain persistence)
```

## Phase 3: API-First Architecture

### 3.1 Define API Contract
```protobuf
// api/session/v1/session.proto
service SessionService {
  rpc CreateSession(CreateSessionRequest) returns (Session);
  rpc AttachSession(AttachSessionRequest) returns (stream Output);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
}
```

### 3.2 Implement Server
```go
// cmd/server/main.go
func main() {
    // Wire up dependencies
    app := wire.InitializeApp()

    // Start gRPC server
    grpcServer := grpc.NewServer()
    sessionv1.RegisterSessionServiceServer(grpcServer, app.SessionService)

    // Start HTTP gateway
    mux := runtime.NewServeMux()
    sessionv1.RegisterSessionServiceHandlerServer(ctx, mux, app.SessionService)
}
```

### 3.3 Migrate CLI to Client
```go
// cmd/claude-squad/client.go
func runCommand(cmd *cobra.Command, args []string) error {
    client := sessionv1.NewSessionServiceClient(conn)
    resp, err := client.CreateSession(ctx, &sessionv1.CreateSessionRequest{
        Title: args[0],
    })
}
```

## Implementation Timeline

### Week 1-2: Phase 1.1-1.2
- Extract Session Orchestrator
- Create Git Service interface

### Week 3-4: Phase 1.3-1.4
- Create Tmux Service interface
- Implement Storage Repository

### Week 5: Phase 1.5 + Testing
- Command Executor abstraction
- Comprehensive test suite

### Week 6: Phase 1 Validation
- Integration testing
- Performance benchmarks
- Documentation

## Risk Mitigation

1. **Incremental Changes**: Each step is independently deployable
2. **Feature Flags**: New services can be toggled on/off
3. **Parallel Running**: Old and new code can coexist during migration
4. **Rollback Plan**: Git tags at each milestone for easy reversion
5. **Test Coverage**: Minimum 80% coverage before moving to next phase

## Success Metrics

- **Code Quality**: Reduced cyclomatic complexity by 30%
- **Test Coverage**: Increase from current to >80%
- **Build Time**: Reduce by 20% through better modularity
- **Developer Velocity**: New features 40% faster to implement
- **Bug Rate**: Reduce production bugs by 50%

## Next Steps

1. Review and approve this plan
2. Create feature branch for Phase 1
3. Set up CI/CD pipeline for new test suite
4. Begin Phase 1.1 implementation
5. Weekly progress reviews

## Notes

- Each phase produces a stable, working system
- No breaking changes to external interfaces until Phase 3
- Documentation updates parallel to implementation
- Team training on new patterns during Phase 1