package git

import (
	"context"
	"fmt"
	"time"
)

// MockGitService is a mock implementation of GitService for testing
type MockGitService struct {
	// Function fields for overriding behavior
	IsGitRepositoryFunc              func(ctx context.Context, path string) (bool, error)
	GetRepositoryRootFunc            func(ctx context.Context, path string) (string, error)
	ListBranchesFunc                 func(ctx context.Context, repoPath string) ([]Branch, error)
	CreateBranchFunc                 func(ctx context.Context, repoPath, branchName string) error
	DeleteBranchFunc                 func(ctx context.Context, repoPath, branchName string, force bool) error
	CheckoutBranchFunc               func(ctx context.Context, repoPath, branchName string) error
	GetCurrentBranchFunc             func(ctx context.Context, repoPath string) (*Branch, error)
	CreateWorktreeFunc               func(ctx context.Context, repoPath, worktreePath, branch string) (*Worktree, error)
	ListWorktreesFunc                func(ctx context.Context, repoPath string) ([]*Worktree, error)
	RemoveWorktreeFunc               func(ctx context.Context, worktreePath string, force bool) error
	GetWorktreeInfoFunc              func(ctx context.Context, worktreePath string) (*Worktree, error)
	GetDiffStatsFunc                 func(ctx context.Context, repoPath string) (*DiffStats, error)
	GetDiffStatsStagedFunc           func(ctx context.Context, repoPath string) (*DiffStats, error)
	GetDiffStatsBetweenBranchesFunc func(ctx context.Context, repoPath, fromBranch, toBranch string) (*DiffStats, error)
	CommitFunc                       func(ctx context.Context, repoPath, message string) error
	GetLastCommitFunc                func(ctx context.Context, repoPath string) (*CommitInfo, error)
	GetCommitHistoryFunc             func(ctx context.Context, repoPath string, limit int) ([]*CommitInfo, error)
	StashFunc                        func(ctx context.Context, repoPath, message string) error
	PopStashFunc                     func(ctx context.Context, repoPath string) error
	ListStashesFunc                  func(ctx context.Context, repoPath string) ([]string, error)
	GetStatusFunc                    func(ctx context.Context, repoPath string) ([]string, error)
	HasUncommittedChangesFunc        func(ctx context.Context, repoPath string) (bool, error)
	CleanupWorktreesFunc             func(ctx context.Context, repoPath string) error
	PruneWorktreesFunc               func(ctx context.Context, repoPath string) error

	// Default responses for simple cases
	DefaultIsRepo     bool
	DefaultBranch     string
	DefaultWorktrees  []*Worktree
	DefaultDiffStats  *DiffStats
	DefaultCommitInfo *CommitInfo
}

// NewMockGitService creates a new mock with sensible defaults
func NewMockGitService() *MockGitService {
	return &MockGitService{
		DefaultIsRepo: true,
		DefaultBranch: "main",
		DefaultWorktrees: []*Worktree{
			{
				Path:   "/tmp/worktree",
				Branch: "main",
				Hash:   "abc123",
			},
		},
		DefaultDiffStats: &DiffStats{
			FilesChanged: 0,
			Insertions:   0,
			Deletions:    0,
			Files:        []FileDiff{},
		},
		DefaultCommitInfo: &CommitInfo{
			Hash:      "abc123",
			Author:    "Test User",
			Email:     "test@example.com",
			Message:   "Test commit",
			Timestamp: time.Now(),
		},
	}
}

func (m *MockGitService) IsGitRepository(ctx context.Context, path string) (bool, error) {
	if m.IsGitRepositoryFunc != nil {
		return m.IsGitRepositoryFunc(ctx, path)
	}
	return m.DefaultIsRepo, nil
}

func (m *MockGitService) GetRepositoryRoot(ctx context.Context, path string) (string, error) {
	if m.GetRepositoryRootFunc != nil {
		return m.GetRepositoryRootFunc(ctx, path)
	}
	return path, nil
}

func (m *MockGitService) ListBranches(ctx context.Context, repoPath string) ([]Branch, error) {
	if m.ListBranchesFunc != nil {
		return m.ListBranchesFunc(ctx, repoPath)
	}
	return []Branch{
		{Name: m.DefaultBranch, IsCurrent: true, Hash: "abc123"},
	}, nil
}

func (m *MockGitService) CreateBranch(ctx context.Context, repoPath, branchName string) error {
	if m.CreateBranchFunc != nil {
		return m.CreateBranchFunc(ctx, repoPath, branchName)
	}
	return nil
}

func (m *MockGitService) DeleteBranch(ctx context.Context, repoPath, branchName string, force bool) error {
	if m.DeleteBranchFunc != nil {
		return m.DeleteBranchFunc(ctx, repoPath, branchName, force)
	}
	return nil
}

func (m *MockGitService) CheckoutBranch(ctx context.Context, repoPath, branchName string) error {
	if m.CheckoutBranchFunc != nil {
		return m.CheckoutBranchFunc(ctx, repoPath, branchName)
	}
	return nil
}

func (m *MockGitService) GetCurrentBranch(ctx context.Context, repoPath string) (*Branch, error) {
	if m.GetCurrentBranchFunc != nil {
		return m.GetCurrentBranchFunc(ctx, repoPath)
	}
	return &Branch{Name: m.DefaultBranch, IsCurrent: true, Hash: "abc123"}, nil
}

func (m *MockGitService) CreateWorktree(ctx context.Context, repoPath, worktreePath, branch string) (*Worktree, error) {
	if m.CreateWorktreeFunc != nil {
		return m.CreateWorktreeFunc(ctx, repoPath, worktreePath, branch)
	}
	return &Worktree{
		Path:   worktreePath,
		Branch: branch,
		Hash:   "abc123",
	}, nil
}

func (m *MockGitService) ListWorktrees(ctx context.Context, repoPath string) ([]*Worktree, error) {
	if m.ListWorktreesFunc != nil {
		return m.ListWorktreesFunc(ctx, repoPath)
	}
	return m.DefaultWorktrees, nil
}

func (m *MockGitService) RemoveWorktree(ctx context.Context, worktreePath string, force bool) error {
	if m.RemoveWorktreeFunc != nil {
		return m.RemoveWorktreeFunc(ctx, worktreePath, force)
	}
	return nil
}

func (m *MockGitService) GetWorktreeInfo(ctx context.Context, worktreePath string) (*Worktree, error) {
	if m.GetWorktreeInfoFunc != nil {
		return m.GetWorktreeInfoFunc(ctx, worktreePath)
	}
	if len(m.DefaultWorktrees) > 0 {
		return m.DefaultWorktrees[0], nil
	}
	return nil, fmt.Errorf("worktree not found")
}

func (m *MockGitService) GetDiffStats(ctx context.Context, repoPath string) (*DiffStats, error) {
	if m.GetDiffStatsFunc != nil {
		return m.GetDiffStatsFunc(ctx, repoPath)
	}
	return m.DefaultDiffStats, nil
}

func (m *MockGitService) GetDiffStatsStaged(ctx context.Context, repoPath string) (*DiffStats, error) {
	if m.GetDiffStatsStagedFunc != nil {
		return m.GetDiffStatsStagedFunc(ctx, repoPath)
	}
	return m.DefaultDiffStats, nil
}

func (m *MockGitService) GetDiffStatsBetweenBranches(ctx context.Context, repoPath, fromBranch, toBranch string) (*DiffStats, error) {
	if m.GetDiffStatsBetweenBranchesFunc != nil {
		return m.GetDiffStatsBetweenBranchesFunc(ctx, repoPath, fromBranch, toBranch)
	}
	return m.DefaultDiffStats, nil
}

func (m *MockGitService) Commit(ctx context.Context, repoPath, message string) error {
	if m.CommitFunc != nil {
		return m.CommitFunc(ctx, repoPath, message)
	}
	return nil
}

func (m *MockGitService) GetLastCommit(ctx context.Context, repoPath string) (*CommitInfo, error) {
	if m.GetLastCommitFunc != nil {
		return m.GetLastCommitFunc(ctx, repoPath)
	}
	return m.DefaultCommitInfo, nil
}

func (m *MockGitService) GetCommitHistory(ctx context.Context, repoPath string, limit int) ([]*CommitInfo, error) {
	if m.GetCommitHistoryFunc != nil {
		return m.GetCommitHistoryFunc(ctx, repoPath, limit)
	}
	return []*CommitInfo{m.DefaultCommitInfo}, nil
}

func (m *MockGitService) Stash(ctx context.Context, repoPath, message string) error {
	if m.StashFunc != nil {
		return m.StashFunc(ctx, repoPath, message)
	}
	return nil
}

func (m *MockGitService) PopStash(ctx context.Context, repoPath string) error {
	if m.PopStashFunc != nil {
		return m.PopStashFunc(ctx, repoPath)
	}
	return nil
}

func (m *MockGitService) ListStashes(ctx context.Context, repoPath string) ([]string, error) {
	if m.ListStashesFunc != nil {
		return m.ListStashesFunc(ctx, repoPath)
	}
	return []string{}, nil
}

func (m *MockGitService) GetStatus(ctx context.Context, repoPath string) ([]string, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(ctx, repoPath)
	}
	return []string{}, nil
}

func (m *MockGitService) HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	if m.HasUncommittedChangesFunc != nil {
		return m.HasUncommittedChangesFunc(ctx, repoPath)
	}
	return false, nil
}

func (m *MockGitService) CleanupWorktrees(ctx context.Context, repoPath string) error {
	if m.CleanupWorktreesFunc != nil {
		return m.CleanupWorktreesFunc(ctx, repoPath)
	}
	return nil
}

func (m *MockGitService) PruneWorktrees(ctx context.Context, repoPath string) error {
	if m.PruneWorktreesFunc != nil {
		return m.PruneWorktreesFunc(ctx, repoPath)
	}
	return nil
}