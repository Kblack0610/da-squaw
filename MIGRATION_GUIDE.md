# Migration Guide - Phase 1 Service Architecture

This guide explains how to migrate from the old monolithic code to the new service-oriented architecture.

## Overview

The Phase 1 refactoring introduces a clean service architecture that separates concerns and improves testability without changing external behavior.

## New Architecture Structure

```
services/
├── session/           # Session orchestration
├── git/              # Git operations service
├── tmux/             # Tmux session management
├── storage/          # Persistence layer
└── executor/         # Command execution abstraction
```

## Key Changes

### 1. Session Management

**Old Code:**
```go
instance := session.NewInstance(...)
instance.Start()
instance.Attach()
```

**New Code:**
```go
orchestrator.CreateSession(ctx, req)
orchestrator.AttachSession(ctx, sessionID)
```

### 2. Dependency Injection

**Old Code:**
```go
// Direct instantiation scattered throughout
storage := session.NewStorage(state)
tmux := tmux.NewTmuxSession(name, program)
```

**New Code:**
```go
// Centralized dependency injection
deps, err := app.InitializeDependencies()
orchestrator := deps.Orchestrator
```

### 3. Service Interfaces

All major components now have interfaces for better testing:

- `GitService` - Git repository operations
- `TmuxService` - Terminal multiplexer management
- `StorageRepository` - Data persistence
- `CommandExecutor` - Command execution
- `SessionOrchestrator` - Coordinates all services

## Migration Steps

### Step 1: Update Imports

Replace old imports:
```go
// Old
import (
    "claude-squad/session"
    "claude-squad/session/git"
    "claude-squad/session/tmux"
)

// New
import (
    "claude-squad/services/session"
    "claude-squad/services/git"
    "claude-squad/services/tmux"
    "claude-squad/services/storage"
    "claude-squad/services/executor"
)
```

### Step 2: Initialize Dependencies

In your main application:

```go
func main() {
    // Initialize all dependencies
    deps, err := app.InitializeDependencies()
    if err != nil {
        log.Fatal(err)
    }
    defer deps.Cleanup()

    // Use services through dependencies
    ctx := context.Background()
    sessions, err := deps.Orchestrator.ListSessions(ctx)
}
```

### Step 3: Update Session Operations

Replace direct instance manipulation:

```go
// Old way
instance := &session.Instance{
    Title: "My Session",
    Path:  "/path/to/repo",
}
instance.Start()

// New way
req := session.CreateSessionRequest{
    Title:   "My Session",
    Path:    "/path/to/repo",
    Program: "claude",
}
sess, err := orchestrator.CreateSession(ctx, req)
```

### Step 4: Update Git Operations

```go
// Old way
worktree := git.NewGitWorktree(repoPath, sessionName)
worktree.Setup()

// New way
worktree, err := gitService.CreateWorktree(ctx, repoPath, worktreePath, branch)
```

### Step 5: Update Tmux Operations

```go
// Old way
tmuxSession := tmux.NewTmuxSession(name, program)
tmuxSession.Start(workDir)

// New way
session, err := tmuxService.CreateSession(ctx, name, workDir, program)
```

## Running Both Old and New Code

During migration, you can run both versions in parallel:

### Using Environment Variables

```go
if os.Getenv("USE_NEW_ARCHITECTURE") == "true" {
    return app.RunNew(ctx, program, autoYes)
} else {
    return app.Run(ctx, program, autoYes)
}
```

### Using Feature Flags

```go
if config.FeatureFlags.UseNewArchitecture {
    deps, _ := app.InitializeDependencies()
    return runWithNewArchitecture(deps)
} else {
    return runWithOldCode()
}
```

## Testing

### Unit Tests with Mocks

```go
func TestSessionCreation(t *testing.T) {
    // Create mocks
    gitMock := git.NewMockGitService()
    tmuxMock := tmux.NewMockTmuxService()
    storageMock := storage.NewMockStorageRepository()
    execMock := executor.NewMockExecutor()

    // Configure mock behavior
    gitMock.IsGitRepositoryFunc = func(ctx context.Context, path string) (bool, error) {
        return true, nil
    }

    // Create orchestrator with mocks
    orch := session.NewOrchestrator(gitMock, tmuxMock, storageMock, execMock)

    // Test
    req := session.CreateSessionRequest{Title: "Test"}
    sess, err := orch.CreateSession(context.Background(), req)

    assert.NoError(t, err)
    assert.Equal(t, "Test", sess.Title)
}
```

### Integration Tests

```go
func TestIntegration(t *testing.T) {
    // Use real implementations
    deps, _ := app.InitializeDependencies()

    // Test full workflow
    ctx := context.Background()
    req := session.CreateSessionRequest{
        Title: "Integration Test",
        Path:  testRepoPath,
    }

    sess, err := deps.Orchestrator.CreateSession(ctx, req)
    require.NoError(t, err)

    defer deps.Orchestrator.StopSession(ctx, sess.ID)

    // Verify session was created
    sessions, err := deps.Orchestrator.ListSessions(ctx)
    require.NoError(t, err)
    assert.Len(t, sessions, 1)
}
```

## Rollback Plan

If issues arise, you can quickly rollback:

1. **Git Tags**: Tag before migration
   ```bash
   git tag pre-migration-v1
   ```

2. **Feature Flags**: Disable new code
   ```bash
   export USE_NEW_ARCHITECTURE=false
   ```

3. **Revert Commits**: If necessary
   ```bash
   git revert <migration-commit>
   ```

## Benefits of New Architecture

1. **Testability**: All services can be mocked for unit testing
2. **Modularity**: Services can be developed and tested independently
3. **Maintainability**: Clear separation of concerns
4. **Extensibility**: Easy to add new storage backends or Git providers
5. **Performance**: Services can be optimized independently
6. **Debugging**: Clearer stack traces and error boundaries

## Common Issues and Solutions

### Issue: Session not found after creation
**Solution**: Ensure storage path is correctly configured in dependencies

### Issue: Tmux commands failing
**Solution**: Check that CommandExecutor has proper timeout settings

### Issue: Git worktree conflicts
**Solution**: Use force flag in RemoveWorktree when cleaning up

### Issue: Tests failing with new mocks
**Solution**: Configure mock functions to match expected behavior

## Performance Considerations

- Service calls add minimal overhead (~1ms per call)
- Parallel operations are now easier with service architecture
- Consider caching frequently accessed data in orchestrator

## Next Steps

After successful Phase 1 migration:

1. **Phase 2**: Introduce clean architecture layers
2. **Phase 3**: Add API layer for remote operations
3. **Phase 4**: Enable microservice deployment

## Getting Help

- Check test files for usage examples
- Review mock implementations for testing patterns
- See `PHASE1_IMPLEMENTATION.md` for technical details

## Checklist

Before switching to production:

- [ ] All unit tests passing
- [ ] Integration tests cover main workflows
- [ ] Performance benchmarks show no regression
- [ ] Error handling tested
- [ ] Logging and monitoring updated
- [ ] Documentation updated
- [ ] Team trained on new patterns
- [ ] Rollback plan tested