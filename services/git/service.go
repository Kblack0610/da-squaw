package git

import (
	"context"
	"time"
)

// Branch represents a git branch
type Branch struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
	Hash      string
	UpdatedAt time.Time
}

// Worktree represents a git worktree
type Worktree struct {
	Path       string
	Branch     string
	Hash       string
	IsDetached bool
	IsLocked   bool
}

// DiffStats represents statistics about git diff
type DiffStats struct {
	FilesChanged int
	Insertions   int
	Deletions    int
	Files        []FileDiff
}

// FileDiff represents changes to a single file
type FileDiff struct {
	Path       string
	Insertions int
	Deletions  int
	Binary     bool
	Status     string // "modified", "added", "deleted", "renamed"
}

// CommitInfo represents git commit information
type CommitInfo struct {
	Hash      string
	Author    string
	Email     string
	Message   string
	Timestamp time.Time
}

// GitService provides git repository operations
type GitService interface {
	// Repository operations
	IsGitRepository(ctx context.Context, path string) (bool, error)
	GetRepositoryRoot(ctx context.Context, path string) (string, error)

	// Branch operations
	ListBranches(ctx context.Context, repoPath string) ([]Branch, error)
	CreateBranch(ctx context.Context, repoPath, branchName string) error
	DeleteBranch(ctx context.Context, repoPath, branchName string, force bool) error
	CheckoutBranch(ctx context.Context, repoPath, branchName string) error
	GetCurrentBranch(ctx context.Context, repoPath string) (*Branch, error)

	// Worktree operations
	CreateWorktree(ctx context.Context, repoPath, worktreePath, branch string) (*Worktree, error)
	ListWorktrees(ctx context.Context, repoPath string) ([]*Worktree, error)
	RemoveWorktree(ctx context.Context, worktreePath string, force bool) error
	GetWorktreeInfo(ctx context.Context, worktreePath string) (*Worktree, error)

	// Diff operations
	GetDiffStats(ctx context.Context, repoPath string) (*DiffStats, error)
	GetDiffStatsStaged(ctx context.Context, repoPath string) (*DiffStats, error)
	GetDiffStatsBetweenBranches(ctx context.Context, repoPath, fromBranch, toBranch string) (*DiffStats, error)

	// Commit operations
	Commit(ctx context.Context, repoPath, message string) error
	GetLastCommit(ctx context.Context, repoPath string) (*CommitInfo, error)
	GetCommitHistory(ctx context.Context, repoPath string, limit int) ([]*CommitInfo, error)

	// Stash operations
	Stash(ctx context.Context, repoPath, message string) error
	PopStash(ctx context.Context, repoPath string) error
	ListStashes(ctx context.Context, repoPath string) ([]string, error)

	// Status operations
	GetStatus(ctx context.Context, repoPath string) ([]string, error)
	HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error)

	// Cleanup operations
	CleanupWorktrees(ctx context.Context, repoPath string) error
	PruneWorktrees(ctx context.Context, repoPath string) error
}