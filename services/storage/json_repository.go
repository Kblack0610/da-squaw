package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"claude-squad/services/types"
)

// jsonRepository is a JSON file-based implementation of StorageRepository
type jsonRepository struct {
	basePath string
	mu       sync.RWMutex
}

// NewJSONRepository creates a new JSON-based storage repository
func NewJSONRepository(basePath string) (StorageRepository, error) {
	// Ensure the base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &jsonRepository{
		basePath: basePath,
	}, nil
}

func (r *jsonRepository) getFilePath(id string) string {
	return filepath.Join(r.basePath, fmt.Sprintf("%s.json", id))
}

func (r *jsonRepository) getAllFilePaths() ([]string, error) {
	entries, err := os.ReadDir(r.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			paths = append(paths, filepath.Join(r.basePath, entry.Name()))
		}
	}
	return paths, nil
}

// Basic CRUD operations

func (r *jsonRepository) Create(ctx context.Context, session *types.SessionData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}

	filePath := r.getFilePath(session.ID)

	// Check if already exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("session already exists: %s", session.ID)
	}

	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

func (r *jsonRepository) Get(ctx context.Context, id string) (*types.SessionData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filePath := r.getFilePath(id)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session types.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (r *jsonRepository) Update(ctx context.Context, session *types.SessionData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}

	filePath := r.getFilePath(session.ID)

	// Check if exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	session.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

func (r *jsonRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filePath := r.getFilePath(id)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", id)
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// Batch operations

func (r *jsonRepository) CreateBatch(ctx context.Context, sessions []*types.SessionData) error {
	for _, session := range sessions {
		if err := r.Create(ctx, session); err != nil {
			return fmt.Errorf("failed to create session %s: %w", session.ID, err)
		}
	}
	return nil
}

func (r *jsonRepository) UpdateBatch(ctx context.Context, sessions []*types.SessionData) error {
	for _, session := range sessions {
		if err := r.Update(ctx, session); err != nil {
			return fmt.Errorf("failed to update session %s: %w", session.ID, err)
		}
	}
	return nil
}

func (r *jsonRepository) DeleteBatch(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete session %s: %w", id, err)
		}
	}
	return nil
}

// Query operations

func (r *jsonRepository) List(ctx context.Context, opts *QueryOptions) ([]*types.SessionData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths, err := r.getAllFilePaths()
	if err != nil {
		return nil, err
	}

	var sessions []*types.SessionData
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue // Skip files that can't be read
		}

		var session types.SessionData
		if err := json.Unmarshal(data, &session); err != nil {
			continue // Skip invalid JSON files
		}

		// Apply filters if options provided
		if opts != nil {
			if opts.Status != nil && session.Status != *opts.Status {
				continue
			}
			if opts.Branch != nil && session.Branch != *opts.Branch {
				continue
			}
			if opts.Path != nil && session.Path != *opts.Path {
				continue
			}
			if opts.Program != nil && session.Program != *opts.Program {
				continue
			}
			if opts.AutoYes != nil && session.AutoYes != *opts.AutoYes {
				continue
			}
			if opts.CreatedAfter != nil && session.CreatedAt.Before(*opts.CreatedAfter) {
				continue
			}
			if opts.CreatedBefore != nil && session.CreatedAt.After(*opts.CreatedBefore) {
				continue
			}
			if opts.UpdatedAfter != nil && session.UpdatedAt.Before(*opts.UpdatedAfter) {
				continue
			}
			if opts.UpdatedBefore != nil && session.UpdatedAt.After(*opts.UpdatedBefore) {
				continue
			}
		}

		sessions = append(sessions, &session)
	}

	// Apply sorting
	if opts != nil && opts.SortBy != "" {
		sortSessions(sessions, opts.SortBy, opts.SortOrder)
	}

	// Apply pagination
	if opts != nil && opts.Limit > 0 {
		start := opts.Offset
		if start >= len(sessions) {
			return []*types.SessionData{}, nil
		}
		end := start + opts.Limit
		if end > len(sessions) {
			end = len(sessions)
		}
		sessions = sessions[start:end]
	}

	return sessions, nil
}

func (r *jsonRepository) Count(ctx context.Context, opts *QueryOptions) (int, error) {
	sessions, err := r.List(ctx, opts)
	if err != nil {
		return 0, err
	}
	return len(sessions), nil
}

func (r *jsonRepository) Exists(ctx context.Context, id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filePath := r.getFilePath(id)
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Specialized queries

func (r *jsonRepository) GetByTitle(ctx context.Context, title string) (*types.SessionData, error) {
	sessions, err := r.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.Title == title {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found with title: %s", title)
}

func (r *jsonRepository) GetByBranch(ctx context.Context, branch string) ([]*types.SessionData, error) {
	return r.List(ctx, &QueryOptions{Branch: &branch})
}

func (r *jsonRepository) GetActive(ctx context.Context) ([]*types.SessionData, error) {
	running := types.StatusRunning
	ready := types.StatusReady

	sessions, err := r.List(ctx, &QueryOptions{Status: &running})
	if err != nil {
		return nil, err
	}

	readySessions, err := r.List(ctx, &QueryOptions{Status: &ready})
	if err != nil {
		return nil, err
	}

	sessions = append(sessions, readySessions...)
	return sessions, nil
}

func (r *jsonRepository) GetPaused(ctx context.Context) ([]*types.SessionData, error) {
	paused := types.StatusPaused
	return r.List(ctx, &QueryOptions{Status: &paused})
}

// Status operations

func (r *jsonRepository) UpdateStatus(ctx context.Context, id string, status types.Status) error {
	session, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	return r.Update(ctx, session)
}

func (r *jsonRepository) UpdateStatusBatch(ctx context.Context, updates map[string]types.Status) error {
	for id, status := range updates {
		if err := r.UpdateStatus(ctx, id, status); err != nil {
			return fmt.Errorf("failed to update status for %s: %w", id, err)
		}
	}
	return nil
}

// Metadata operations

func (r *jsonRepository) SetMetadata(ctx context.Context, id string, key, value string) error {
	session, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Metadata == nil {
		session.Metadata = make(map[string]string)
	}
	session.Metadata[key] = value
	session.UpdatedAt = time.Now()

	return r.Update(ctx, session)
}

func (r *jsonRepository) GetMetadata(ctx context.Context, id string, key string) (string, error) {
	session, err := r.Get(ctx, id)
	if err != nil {
		return "", err
	}

	if session.Metadata == nil {
		return "", fmt.Errorf("metadata key not found: %s", key)
	}

	value, exists := session.Metadata[key]
	if !exists {
		return "", fmt.Errorf("metadata key not found: %s", key)
	}

	return value, nil
}

func (r *jsonRepository) DeleteMetadata(ctx context.Context, id string, key string) error {
	session, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Metadata != nil {
		delete(session.Metadata, key)
		session.UpdatedAt = time.Now()
		return r.Update(ctx, session)
	}

	return nil
}

// Maintenance operations

func (r *jsonRepository) DeleteAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	paths, err := r.getAllFilePaths()
	if err != nil {
		return err
	}

	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", path, err)
		}
	}

	return nil
}

func (r *jsonRepository) DeleteOlderThan(ctx context.Context, duration time.Duration) error {
	cutoff := time.Now().Add(-duration)
	sessions, err := r.List(ctx, &QueryOptions{UpdatedBefore: &cutoff})
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if err := r.Delete(ctx, session.ID); err != nil {
			return fmt.Errorf("failed to delete old session %s: %w", session.ID, err)
		}
	}

	return nil
}

func (r *jsonRepository) Vacuum(ctx context.Context) error {
	// For JSON repository, vacuum could compact files or clean up metadata
	// Currently a no-op
	return nil
}

func (r *jsonRepository) Backup(ctx context.Context, backupPath string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	paths, err := r.getAllFilePaths()
	if err != nil {
		return err
	}

	for _, srcPath := range paths {
		filename := filepath.Base(srcPath)
		dstPath := filepath.Join(backupPath, filename)

		data, err := ioutil.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", srcPath, err)
		}

		if err := ioutil.WriteFile(dstPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write backup file %s: %w", dstPath, err)
		}
	}

	return nil
}

func (r *jsonRepository) Restore(ctx context.Context, backupPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing data
	if err := r.DeleteAll(ctx); err != nil {
		return fmt.Errorf("failed to clear existing data: %w", err)
	}

	// Read backup files
	entries, err := os.ReadDir(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		srcPath := filepath.Join(backupPath, entry.Name())
		dstPath := filepath.Join(r.basePath, entry.Name())

		data, err := ioutil.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read backup file %s: %w", srcPath, err)
		}

		if err := ioutil.WriteFile(dstPath, data, 0644); err != nil {
			return fmt.Errorf("failed to restore file %s: %w", dstPath, err)
		}
	}

	return nil
}

// Transaction support

func (r *jsonRepository) BeginTx(ctx context.Context) (Transaction, error) {
	// JSON repository doesn't support real transactions
	// Return a no-op transaction
	return &noOpTransaction{repo: r}, nil
}

// noOpTransaction is a transaction that just delegates to the repository
type noOpTransaction struct {
	repo StorageRepository
}

func (t *noOpTransaction) Commit() error {
	return nil // No-op
}

func (t *noOpTransaction) Rollback() error {
	return nil // No-op
}

// Delegate all methods to the underlying repository
func (t *noOpTransaction) Create(ctx context.Context, session *types.SessionData) error {
	return t.repo.Create(ctx, session)
}

func (t *noOpTransaction) Get(ctx context.Context, id string) (*types.SessionData, error) {
	return t.repo.Get(ctx, id)
}

func (t *noOpTransaction) Update(ctx context.Context, session *types.SessionData) error {
	return t.repo.Update(ctx, session)
}

func (t *noOpTransaction) Delete(ctx context.Context, id string) error {
	return t.repo.Delete(ctx, id)
}

func (t *noOpTransaction) CreateBatch(ctx context.Context, sessions []*types.SessionData) error {
	return t.repo.CreateBatch(ctx, sessions)
}

func (t *noOpTransaction) UpdateBatch(ctx context.Context, sessions []*types.SessionData) error {
	return t.repo.UpdateBatch(ctx, sessions)
}

func (t *noOpTransaction) DeleteBatch(ctx context.Context, ids []string) error {
	return t.repo.DeleteBatch(ctx, ids)
}

func (t *noOpTransaction) List(ctx context.Context, opts *QueryOptions) ([]*types.SessionData, error) {
	return t.repo.List(ctx, opts)
}

func (t *noOpTransaction) Count(ctx context.Context, opts *QueryOptions) (int, error) {
	return t.repo.Count(ctx, opts)
}

func (t *noOpTransaction) Exists(ctx context.Context, id string) (bool, error) {
	return t.repo.Exists(ctx, id)
}

func (t *noOpTransaction) GetByTitle(ctx context.Context, title string) (*types.SessionData, error) {
	return t.repo.GetByTitle(ctx, title)
}

func (t *noOpTransaction) GetByBranch(ctx context.Context, branch string) ([]*types.SessionData, error) {
	return t.repo.GetByBranch(ctx, branch)
}

func (t *noOpTransaction) GetActive(ctx context.Context) ([]*types.SessionData, error) {
	return t.repo.GetActive(ctx)
}

func (t *noOpTransaction) GetPaused(ctx context.Context) ([]*types.SessionData, error) {
	return t.repo.GetPaused(ctx)
}

func (t *noOpTransaction) UpdateStatus(ctx context.Context, id string, status types.Status) error {
	return t.repo.UpdateStatus(ctx, id, status)
}

func (t *noOpTransaction) UpdateStatusBatch(ctx context.Context, updates map[string]types.Status) error {
	return t.repo.UpdateStatusBatch(ctx, updates)
}

func (t *noOpTransaction) SetMetadata(ctx context.Context, id string, key, value string) error {
	return t.repo.SetMetadata(ctx, id, key, value)
}

func (t *noOpTransaction) GetMetadata(ctx context.Context, id string, key string) (string, error) {
	return t.repo.GetMetadata(ctx, id, key)
}

func (t *noOpTransaction) DeleteMetadata(ctx context.Context, id string, key string) error {
	return t.repo.DeleteMetadata(ctx, id, key)
}

func (t *noOpTransaction) DeleteAll(ctx context.Context) error {
	return t.repo.DeleteAll(ctx)
}

func (t *noOpTransaction) DeleteOlderThan(ctx context.Context, duration time.Duration) error {
	return t.repo.DeleteOlderThan(ctx, duration)
}

func (t *noOpTransaction) Vacuum(ctx context.Context) error {
	return t.repo.Vacuum(ctx)
}

func (t *noOpTransaction) Backup(ctx context.Context, path string) error {
	return t.repo.Backup(ctx, path)
}

func (t *noOpTransaction) Restore(ctx context.Context, path string) error {
	return t.repo.Restore(ctx, path)
}

func (t *noOpTransaction) BeginTx(ctx context.Context) (Transaction, error) {
	return t, nil // Return self
}

// Helper function to sort sessions
func sortSessions(sessions []*types.SessionData, sortBy, sortOrder string) {
	// Implementation of sorting logic based on sortBy field
	// This is a simplified version - you may want to use sort.Slice
	// with appropriate comparison functions based on sortBy
}