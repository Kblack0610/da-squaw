package git

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"claude-squad/services/executor"

	"github.com/go-git/go-git/v5"
)

// execAdapter implements GitService using CommandExecutor
type execAdapter struct {
	executor executor.CommandExecutor
}

// NewGitService creates a new GitService implementation using CommandExecutor
func NewGitService(exec executor.CommandExecutor) GitService {
	return &execAdapter{
		executor: exec,
	}
}

// Repository operations

// IsGitRepository checks if the given path is within a git repository
func (g *execAdapter) IsGitRepository(ctx context.Context, path string) (bool, error) {
	// Try to find git repository using go-git first for efficiency
	for currentPath := path; currentPath != filepath.Dir(currentPath); currentPath = filepath.Dir(currentPath) {
		_, err := git.PlainOpen(currentPath)
		if err == nil {
			return true, nil
		}
	}
	return false, nil
}

// GetRepositoryRoot finds and returns the git repository root path
func (g *execAdapter) GetRepositoryRoot(ctx context.Context, path string) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Find repository root using go-git
	for currentPath := absPath; currentPath != filepath.Dir(currentPath); currentPath = filepath.Dir(currentPath) {
		_, err := git.PlainOpen(currentPath)
		if err == nil {
			return currentPath, nil
		}
	}

	return "", fmt.Errorf("failed to find Git repository root from path: %s", path)
}

// Branch operations

// ListBranches lists all branches in the repository
func (g *execAdapter) ListBranches(ctx context.Context, repoPath string) ([]Branch, error) {
	// List local branches
	localCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "branch", "-v", "--no-abbrev"},
	}

	localResult, err := g.executor.Execute(ctx, localCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list local branches: %w", err)
	}

	// List remote branches
	remoteCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "branch", "-rv", "--no-abbrev"},
	}

	remoteResult, err := g.executor.Execute(ctx, remoteCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote branches: %w", err)
	}

	var branches []Branch

	// Parse local branches
	localLines := strings.Split(string(localResult.Stdout), "\n")
	for _, line := range localLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		branch := g.parseLocalBranch(line)
		if branch != nil {
			branches = append(branches, *branch)
		}
	}

	// Parse remote branches
	remoteLines := strings.Split(string(remoteResult.Stdout), "\n")
	for _, line := range remoteLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") {
			continue
		}

		branch := g.parseRemoteBranch(line)
		if branch != nil {
			branches = append(branches, *branch)
		}
	}

	return branches, nil
}

// parseLocalBranch parses a local branch line from git branch output
func (g *execAdapter) parseLocalBranch(line string) *Branch {
	isCurrent := strings.HasPrefix(line, "*")
	if isCurrent {
		line = strings.TrimPrefix(line, "*")
	}
	line = strings.TrimSpace(line)

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	return &Branch{
		Name:      parts[0],
		IsCurrent: isCurrent,
		IsRemote:  false,
		Hash:      parts[1],
		UpdatedAt: time.Now(), // Would need git log for actual timestamp
	}
}

// parseRemoteBranch parses a remote branch line from git branch output
func (g *execAdapter) parseRemoteBranch(line string) *Branch {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	// Skip origin/HEAD entries
	if strings.Contains(parts[0], "/HEAD") {
		return nil
	}

	return &Branch{
		Name:      parts[0],
		IsCurrent: false,
		IsRemote:  true,
		Hash:      parts[1],
		UpdatedAt: time.Now(),
	}
}

// CreateBranch creates a new branch
func (g *execAdapter) CreateBranch(ctx context.Context, repoPath, branchName string) error {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "branch", branchName},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %s (%w)", branchName, result.Stderr, err)
	}

	return nil
}

// DeleteBranch deletes a branch
func (g *execAdapter) DeleteBranch(ctx context.Context, repoPath, branchName string, force bool) error {
	args := []string{"-C", repoPath, "branch"}
	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branchName)

	cmd := executor.Command{
		Program: "git",
		Args:    args,
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %s (%w)", branchName, result.Stderr, err)
	}

	return nil
}

// CheckoutBranch checks out a branch
func (g *execAdapter) CheckoutBranch(ctx context.Context, repoPath, branchName string) error {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "checkout", branchName},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %s (%w)", branchName, result.Stderr, err)
	}

	return nil
}

// GetCurrentBranch gets the current branch information
func (g *execAdapter) GetCurrentBranch(ctx context.Context, repoPath string) (*Branch, error) {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "branch", "--show-current"},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	branchName := strings.TrimSpace(string(result.Stdout))
	if branchName == "" {
		return nil, fmt.Errorf("not on any branch (detached HEAD)")
	}

	// Get hash for current branch
	hashCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "rev-parse", "HEAD"},
	}

	hashResult, err := g.executor.Execute(ctx, hashCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch hash: %w", err)
	}

	return &Branch{
		Name:      branchName,
		IsCurrent: true,
		IsRemote:  false,
		Hash:      strings.TrimSpace(string(hashResult.Stdout)),
		UpdatedAt: time.Now(),
	}, nil
}

// Worktree operations

// CreateWorktree creates a new worktree
func (g *execAdapter) CreateWorktree(ctx context.Context, repoPath, worktreePath, branch string) (*Worktree, error) {
	// Check if branch exists
	branchExistsCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "rev-parse", "--verify", branch},
	}

	_, err := g.executor.Execute(ctx, branchExistsCmd)
	branchExists := err == nil

	var args []string
	if branchExists {
		// Create worktree from existing branch
		args = []string{"-C", repoPath, "worktree", "add", worktreePath, branch}
	} else {
		// Create worktree with new branch from HEAD
		args = []string{"-C", repoPath, "worktree", "add", "-b", branch, worktreePath}
	}

	cmd := executor.Command{
		Program: "git",
		Args:    args,
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %s (%w)", result.Stderr, err)
	}

	// Get worktree info
	return g.GetWorktreeInfo(ctx, worktreePath)
}

// ListWorktrees lists all worktrees
func (g *execAdapter) ListWorktrees(ctx context.Context, repoPath string) ([]*Worktree, error) {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "worktree", "list", "--porcelain"},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return g.parseWorktrees(string(result.Stdout)), nil
}

// parseWorktrees parses the output of git worktree list --porcelain
func (g *execAdapter) parseWorktrees(output string) []*Worktree {
	var worktrees []*Worktree
	var current *Worktree

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, current)
			}
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") && current != nil {
			current.Hash = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			branch := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
		} else if line == "detached" && current != nil {
			current.IsDetached = true
		} else if line == "locked" && current != nil {
			current.IsLocked = true
		}
	}

	if current != nil {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// RemoveWorktree removes a worktree
func (g *execAdapter) RemoveWorktree(ctx context.Context, worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, worktreePath)

	cmd := executor.Command{
		Program: "git",
		Args:    args,
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to remove worktree %s: %s (%w)", worktreePath, result.Stderr, err)
	}

	return nil
}

// GetWorktreeInfo gets information about a specific worktree
func (g *execAdapter) GetWorktreeInfo(ctx context.Context, worktreePath string) (*Worktree, error) {
	// Get current branch
	branchCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", worktreePath, "branch", "--show-current"},
		Dir:     worktreePath,
	}

	branchResult, err := g.executor.Execute(ctx, branchCmd)
	branch := strings.TrimSpace(string(branchResult.Stdout))
	isDetached := err != nil || branch == ""

	// Get HEAD hash
	hashCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", worktreePath, "rev-parse", "HEAD"},
		Dir:     worktreePath,
	}

	hashResult, err := g.executor.Execute(ctx, hashCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD hash: %w", err)
	}

	return &Worktree{
		Path:       worktreePath,
		Branch:     branch,
		Hash:       strings.TrimSpace(string(hashResult.Stdout)),
		IsDetached: isDetached,
		IsLocked:   false, // Would need to check .git/worktrees/<name>/locked
	}, nil
}

// Diff operations

// GetDiffStats gets diff statistics for the working directory vs HEAD
func (g *execAdapter) GetDiffStats(ctx context.Context, repoPath string) (*DiffStats, error) {
	return g.getDiffStats(ctx, repoPath, []string{"HEAD"})
}

// GetDiffStatsStaged gets diff statistics for staged changes
func (g *execAdapter) GetDiffStatsStaged(ctx context.Context, repoPath string) (*DiffStats, error) {
	return g.getDiffStats(ctx, repoPath, []string{"--cached", "HEAD"})
}

// GetDiffStatsBetweenBranches gets diff statistics between two branches
func (g *execAdapter) GetDiffStatsBetweenBranches(ctx context.Context, repoPath, fromBranch, toBranch string) (*DiffStats, error) {
	return g.getDiffStats(ctx, repoPath, []string{fromBranch + ".." + toBranch})
}

// getDiffStats executes git diff with given arguments and parses the statistics
func (g *execAdapter) getDiffStats(ctx context.Context, repoPath string, diffArgs []string) (*DiffStats, error) {
	// First get the numstat for file-level statistics
	numstatArgs := append([]string{"-C", repoPath, "diff", "--numstat"}, diffArgs...)
	numstatCmd := executor.Command{
		Program: "git",
		Args:    numstatArgs,
	}

	numstatResult, err := g.executor.Execute(ctx, numstatCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff numstat: %w", err)
	}

	// Parse file-level statistics
	files := g.parseNumstat(string(numstatResult.Stdout))

	// Calculate totals
	totalInsertions := 0
	totalDeletions := 0
	for _, file := range files {
		totalInsertions += file.Insertions
		totalDeletions += file.Deletions
	}

	return &DiffStats{
		FilesChanged: len(files),
		Insertions:   totalInsertions,
		Deletions:    totalDeletions,
		Files:        files,
	}, nil
}

// parseNumstat parses the output of git diff --numstat
func (g *execAdapter) parseNumstat(output string) []FileDiff {
	var files []FileDiff
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		file := FileDiff{
			Path:   parts[2],
			Status: "modified", // Default status
		}

		// Parse insertions
		if parts[0] == "-" {
			file.Binary = true
		} else {
			if insertions, err := strconv.Atoi(parts[0]); err == nil {
				file.Insertions = insertions
			}
		}

		// Parse deletions
		if parts[1] == "-" {
			file.Binary = true
		} else {
			if deletions, err := strconv.Atoi(parts[1]); err == nil {
				file.Deletions = deletions
			}
		}

		// Determine status based on insertions/deletions
		if file.Insertions > 0 && file.Deletions == 0 {
			file.Status = "added"
		} else if file.Insertions == 0 && file.Deletions > 0 {
			file.Status = "deleted"
		}

		files = append(files, file)
	}

	return files
}

// Commit operations

// Commit creates a commit with the given message
func (g *execAdapter) Commit(ctx context.Context, repoPath, message string) error {
	// Stage all changes first
	addCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "add", "."},
	}

	_, err := g.executor.Execute(ctx, addCmd)
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create commit
	commitCmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "commit", "-m", message, "--no-verify"},
	}

	result, err := g.executor.Execute(ctx, commitCmd)
	if err != nil {
		return fmt.Errorf("failed to commit: %s (%w)", result.Stderr, err)
	}

	return nil
}

// GetLastCommit gets information about the last commit
func (g *execAdapter) GetLastCommit(ctx context.Context, repoPath string) (*CommitInfo, error) {
	cmd := executor.Command{
		Program: "git",
		Args: []string{
			"-C", repoPath,
			"log", "-1",
			"--pretty=format:%H|%an|%ae|%ct|%s",
		},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get last commit: %w", err)
	}

	return g.parseCommitInfo(string(result.Stdout))
}

// GetCommitHistory gets commit history with a limit
func (g *execAdapter) GetCommitHistory(ctx context.Context, repoPath string, limit int) ([]*CommitInfo, error) {
	cmd := executor.Command{
		Program: "git",
		Args: []string{
			"-C", repoPath,
			"log", fmt.Sprintf("-%d", limit),
			"--pretty=format:%H|%an|%ae|%ct|%s",
		},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	var commits []*CommitInfo
	lines := strings.Split(string(result.Stdout), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		commit, err := g.parseCommitInfo(line)
		if err != nil {
			continue // Skip malformed commit entries
		}
		commits = append(commits, commit)
	}

	return commits, nil
}

// parseCommitInfo parses a commit info line in format: hash|author|email|timestamp|message
func (g *execAdapter) parseCommitInfo(line string) (*CommitInfo, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid commit info format")
	}

	timestamp, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	return &CommitInfo{
		Hash:      parts[0],
		Author:    parts[1],
		Email:     parts[2],
		Timestamp: time.Unix(timestamp, 0),
		Message:   strings.Join(parts[4:], "|"), // In case message contains |
	}, nil
}

// Stash operations

// Stash creates a stash with the given message
func (g *execAdapter) Stash(ctx context.Context, repoPath, message string) error {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "stash", "save", message},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to stash changes: %s (%w)", result.Stderr, err)
	}

	return nil
}

// PopStash pops the latest stash
func (g *execAdapter) PopStash(ctx context.Context, repoPath string) error {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "stash", "pop"},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to pop stash: %s (%w)", result.Stderr, err)
	}

	return nil
}

// ListStashes lists all stashes
func (g *execAdapter) ListStashes(ctx context.Context, repoPath string) ([]string, error) {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "stash", "list"},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list stashes: %w", err)
	}

	var stashes []string
	lines := strings.Split(string(result.Stdout), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			stashes = append(stashes, line)
		}
	}

	return stashes, nil
}

// Status operations

// GetStatus gets the repository status
func (g *execAdapter) GetStatus(ctx context.Context, repoPath string) ([]string, error) {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "status", "--porcelain"},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var statusLines []string
	lines := strings.Split(string(result.Stdout), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			statusLines = append(statusLines, line)
		}
	}

	return statusLines, nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func (g *execAdapter) HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	status, err := g.GetStatus(ctx, repoPath)
	if err != nil {
		return false, err
	}
	return len(status) > 0, nil
}

// Cleanup operations

// CleanupWorktrees removes all worktrees and prunes
func (g *execAdapter) CleanupWorktrees(ctx context.Context, repoPath string) error {
	// Get list of all worktrees first
	worktrees, err := g.ListWorktrees(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var errors []error

	// Remove each worktree (except main repository)
	for _, wt := range worktrees {
		if wt.Path == repoPath {
			continue // Skip main repository
		}

		if err := g.RemoveWorktree(ctx, wt.Path, true); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove worktree %s: %w", wt.Path, err))
		}
	}

	// Prune after removing worktrees
	if err := g.PruneWorktrees(ctx, repoPath); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// PruneWorktrees removes stale worktree administrative files
func (g *execAdapter) PruneWorktrees(ctx context.Context, repoPath string) error {
	cmd := executor.Command{
		Program: "git",
		Args:    []string{"-C", repoPath, "worktree", "prune"},
	}

	result, err := g.executor.Execute(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %s (%w)", result.Stderr, err)
	}

	return nil
}

// sanitizeBranchName transforms an arbitrary string into a Git branch name friendly string
func sanitizeBranchName(s string) string {
	// Convert to lower-case
	s = strings.ToLower(s)

	// Replace spaces with a dash
	s = strings.ReplaceAll(s, " ", "-")

	// Remove any characters not allowed in our safe subset.
	// Here we allow: letters, digits, dash, underscore, slash, and dot.
	re := regexp.MustCompile(`[^a-z0-9\-_/.]+`)
	s = re.ReplaceAllString(s, "")

	// Replace multiple dashes with a single dash (optional cleanup)
	reDash := regexp.MustCompile(`-+`)
	s = reDash.ReplaceAllString(s, "-")

	// Trim leading and trailing dashes or slashes to avoid issues
	s = strings.Trim(s, "-/")

	return s
}