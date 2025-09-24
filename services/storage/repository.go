package storage

import (
	"context"
	"time"

	"claude-squad/services/session"
)

// SessionData represents the persistent data of a session
type SessionData struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Path      string            `json:"path"`
	Branch    string            `json:"branch"`
	Status    session.Status    `json:"status"`
	Program   string            `json:"program"`
	Height    int               `json:"height"`
	Width     int               `json:"width"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	AutoYes   bool              `json:"auto_yes"`
	Prompt    string            `json:"prompt"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// QueryOptions provides filtering and pagination for queries
type QueryOptions struct {
	// Filtering
	Status   *session.Status
	Branch   *string
	Path     *string
	Program  *string
	AutoYes  *bool

	// Sorting
	SortBy    string // "created_at", "updated_at", "title"
	SortOrder string // "asc", "desc"

	// Pagination
	Limit  int
	Offset int

	// Time range
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time
}

// StorageRepository provides persistence operations for sessions
type StorageRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, session *SessionData) error
	Get(ctx context.Context, id string) (*SessionData, error)
	Update(ctx context.Context, session *SessionData) error
	Delete(ctx context.Context, id string) error

	// Batch operations
	CreateBatch(ctx context.Context, sessions []*SessionData) error
	UpdateBatch(ctx context.Context, sessions []*SessionData) error
	DeleteBatch(ctx context.Context, ids []string) error

	// Query operations
	List(ctx context.Context, opts *QueryOptions) ([]*SessionData, error)
	Count(ctx context.Context, opts *QueryOptions) (int, error)
	Exists(ctx context.Context, id string) (bool, error)

	// Specialized queries
	GetByTitle(ctx context.Context, title string) (*SessionData, error)
	GetByBranch(ctx context.Context, branch string) ([]*SessionData, error)
	GetActive(ctx context.Context) ([]*SessionData, error)
	GetPaused(ctx context.Context) ([]*SessionData, error)

	// Status operations
	UpdateStatus(ctx context.Context, id string, status session.Status) error
	UpdateStatusBatch(ctx context.Context, updates map[string]session.Status) error

	// Metadata operations
	SetMetadata(ctx context.Context, id string, key, value string) error
	GetMetadata(ctx context.Context, id string, key string) (string, error)
	DeleteMetadata(ctx context.Context, id string, key string) error

	// Maintenance operations
	DeleteAll(ctx context.Context) error
	DeleteOlderThan(ctx context.Context, duration time.Duration) error
	Vacuum(ctx context.Context) error
	Backup(ctx context.Context, path string) error
	Restore(ctx context.Context, path string) error

	// Transaction support (optional - implementations may return ErrNotSupported)
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction provides transactional operations
type Transaction interface {
	StorageRepository
	Commit() error
	Rollback() error
}