# Phase 1 Implementation Guide

## Current Status

✅ **Completed:**
- Created comprehensive refactoring plan (`REFACTOR_PLAN.md`)
- Established services directory structure
- Defined all service interfaces:
  - `SessionOrchestrator` - Coordinates session lifecycle
  - `GitService` - Git repository operations
  - `TmuxService` - Tmux session management
  - `StorageRepository` - Persistence abstraction
  - `CommandExecutor` - Command execution abstraction
- Implemented `orchestratorImpl` - concrete SessionOrchestrator

## Next Steps to Complete Phase 1

### 1. Implement Concrete Adapters

#### Git Service Adapter (`services/git/exec_adapter.go`)
- Migrate logic from `session/git/*.go`
- Use `CommandExecutor` interface instead of direct `os/exec`
- Maintain existing functionality

#### Tmux Service Adapter (`services/tmux/exec_adapter.go`)
- Consolidate `session/tmux/tmux_*.go` implementations
- Handle platform-specific differences
- Use `CommandExecutor` interface

#### Storage Repository (`services/storage/json_repository.go`)
- Adapt existing `session/storage.go` JSON logic
- Implement full `StorageRepository` interface
- Add transaction support (optional)

#### Command Executor (`services/executor/exec_impl.go`)
- Implement concrete executor using `os/exec`
- Add logging and retry capabilities
- Handle streaming output

### 2. Wire Up Dependencies

Create dependency injection setup:
```go
// app/wire.go or app/init.go
func InitializeApp() *App {
    executor := executor.NewExecutor(&executor.ExecutorOptions{})
    gitService := git.NewExecGitService(executor)
    tmuxService := tmux.NewExecTmuxService(executor)
    storage := storage.NewJSONRepository(configDir)
    orchestrator := session.NewOrchestrator(gitService, tmuxService, storage, executor)

    return &App{
        Orchestrator: orchestrator,
        // ... other dependencies
    }
}
```

### 3. Update Existing Code to Use New Services

#### Update `app/app.go`:
```go
// Instead of directly using session.Instance
orchestrator.CreateSession(ctx, req)
orchestrator.AttachSession(ctx, sessionID)
```

#### Update `main.go`:
```go
// Initialize with dependency injection
app := InitializeApp()
```

### 4. Create Comprehensive Tests

#### Unit Tests with Mocks:
```go
// services/session/orchestrator_test.go
func TestCreateSession(t *testing.T) {
    gitMock := &git.MockGitService{}
    tmuxMock := &tmux.MockTmuxService{}
    storageMock := &storage.MockRepository{}
    execMock := &executor.MockExecutor{}

    orch := session.NewOrchestrator(gitMock, tmuxMock, storageMock, execMock)
    // Test session creation
}
```

#### Integration Tests:
```go
// tests/integration/session_lifecycle_test.go
func TestFullSessionLifecycle(t *testing.T) {
    // Test with real implementations in isolated environment
}
```

### 5. Migration Strategy

To ensure smooth transition:

1. **Parallel Running Phase:**
   - Keep old code intact
   - Add feature flag to switch between old/new implementations
   - Run both in production, compare results

2. **Gradual Rollout:**
   - Start with read operations (ListSessions, GetSession)
   - Move to create/update operations
   - Finally migrate delete operations

3. **Validation:**
   - Ensure all existing tests pass
   - Add migration-specific tests
   - Monitor for performance regressions

## File Mapping Guide

| Old File | New Service | Notes |
|----------|-------------|-------|
| `session/instance.go` | `services/session/` | Split into orchestrator + data model |
| `session/git/*.go` | `services/git/` | Consolidate into GitService |
| `session/tmux/*.go` | `services/tmux/` | Consolidate into TmuxService |
| `session/storage.go` | `services/storage/` | Implement repository pattern |
| `cmd/cmd.go` | `services/executor/` | Expand into full executor service |

## Testing Checklist

- [ ] Unit tests for SessionOrchestrator
- [ ] Unit tests for GitService adapter
- [ ] Unit tests for TmuxService adapter
- [ ] Unit tests for StorageRepository
- [ ] Unit tests for CommandExecutor
- [ ] Integration tests for session lifecycle
- [ ] Integration tests for git operations
- [ ] Integration tests for tmux operations
- [ ] Performance benchmarks
- [ ] Migration validation tests

## Success Metrics

Before marking Phase 1 complete, ensure:

1. **Code Coverage**: >80% for new services
2. **Performance**: No regression in operation times
3. **Functionality**: All existing features work identically
4. **Tests**: All existing tests pass
5. **Documentation**: Service interfaces documented

## Common Patterns

### Error Handling
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Context Usage
```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

### Logging
```go
log.Debug("operation started", "sessionID", sessionID)
```

### Testing
```go
// Arrange
mock := setupMock()

// Act
result, err := service.Operation(ctx, input)

// Assert
assert.NoError(t, err)
assert.Equal(t, expected, result)
```

## Questions to Address

1. Should we use dependency injection framework (wire, dig)?
2. How to handle backwards compatibility during migration?
3. What metrics should we track during rollout?
4. Should we version the service interfaces?
5. How to handle configuration for new services?

## Risk Mitigation

- **Feature Flags**: Use environment variables to toggle new code
- **Monitoring**: Add metrics for all service operations
- **Rollback Plan**: Tag releases before each migration step
- **Data Backup**: Ensure session data is backed up before migration
- **Gradual Rollout**: Start with non-critical operations first