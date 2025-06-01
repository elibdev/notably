package repository

import (
	"context"
	"time"

	"github.com/elibdev/notably/internal/models"
)

// UserRepository defines the data access interface for user-scoped operations
type UserRepository interface {
	// Table operations
	CreateTable(ctx context.Context, tableID string, fields []models.FieldDefinition) (*models.TableSchema, error)
	GetTable(ctx context.Context, tableID string) (*models.TableSchema, error)
	ListTables(ctx context.Context) ([]models.TableSchema, error)
	UpdateTable(ctx context.Context, table models.TableSchema) error
	DeleteTable(ctx context.Context, tableID string) error
	GetTableHistory(ctx context.Context, tableID string, opts models.QueryOptions) (*models.TableHistory, error)

	// Entity operations
	CreateEntity(ctx context.Context, tableID string, fields map[string]interface{}) (*models.EntitySnapshot, error)
	GetEntity(ctx context.Context, tableID string, entityID string, asOf *time.Time) (*models.EntitySnapshot, error)
	GetAllEntities(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error)
	GetAllEntitiesIncludingDeleted(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error)
	UpdateEntity(ctx context.Context, tableID string, entityID string, fields map[string]interface{}) (*models.EntitySnapshot, error)
	DeleteEntity(ctx context.Context, tableID string, entityID string) error
	UndeleteEntity(ctx context.Context, tableID string, entityID string) error

	// Field operations
	DeleteField(ctx context.Context, tableID string, entityID string, fieldName string) error
	GetFieldHistory(ctx context.Context, tableID string, fieldName string, opts models.QueryOptions) (*models.FieldHistory, error)

	// History and audit operations
	GetEntityHistory(ctx context.Context, tableID string, entityID string, opts models.QueryOptions) (*models.QueryResult, error)

	// Health and utility operations
	HealthCheck(ctx context.Context) error
}

// UserManager provides higher-level user management operations
type UserManager interface {
	// User context and authentication
	GetUserRepository(userID string) UserRepository
	ValidateUserAccess(ctx context.Context, userID string, tableID string) error

	// Administrative operations
	CreateUser(ctx context.Context, userID string) (*UserStats, error)
	GetUser(ctx context.Context, userID string) (*UserStats, error)
	DeleteUser(ctx context.Context, userID string) error
	GetUserStats(ctx context.Context, userID string) (*UserStats, error)

	// Health check
	Health(ctx context.Context) error
}

// UserStats represents usage statistics for a user
type UserStats struct {
	UserID      string    `json:"user_id"`
	TableCount  int       `json:"table_count"`
	EntityCount int       `json:"entity_count"`
	LastActive  time.Time `json:"last_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// RepositoryConfig contains configuration for repository implementations
type RepositoryConfig struct {
	TableName   string
	Region      string
	EndpointURL string
	Timeout     time.Duration
}

// Error types for repository operations
type RepositoryError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *RepositoryError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

type ErrorType string

const (
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeAlreadyExists ErrorType = "ALREADY_EXISTS"
	ErrorTypeInvalidInput  ErrorType = "INVALID_INPUT"
	ErrorTypeUnauthorized  ErrorType = "UNAUTHORIZED"
	ErrorTypeInternal      ErrorType = "INTERNAL_ERROR"
	ErrorTypeTimeout       ErrorType = "TIMEOUT"
)

// NewRepositoryError creates a new repository error
func NewRepositoryError(errType ErrorType, message string, cause error) *RepositoryError {
	return &RepositoryError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// IsNotFound checks if the error indicates a resource was not found
func IsNotFound(err error) bool {
	if repoErr, ok := err.(*RepositoryError); ok {
		return repoErr.Type == ErrorTypeNotFound
	}
	return false
}

// IsAlreadyExists checks if the error indicates a resource already exists
func IsAlreadyExists(err error) bool {
	if repoErr, ok := err.(*RepositoryError); ok {
		return repoErr.Type == ErrorTypeAlreadyExists
	}
	return false
}
