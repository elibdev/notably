package testutils

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/elibdev/notably/internal/auth"
	"github.com/elibdev/notably/internal/models"
	"github.com/elibdev/notably/internal/repository"
)

// MockUserManager implements repository.UserManager for testing
type MockUserManager struct {
	users        map[string]*repository.User
	userStats    map[string]*repository.UserStats
	repositories map[string]repository.UserRepository
	mutex        sync.RWMutex
}

// NewMockUserManager creates a new mock user manager
func NewMockUserManager() *MockUserManager {
	return &MockUserManager{
		users:        make(map[string]*repository.User),
		userStats:    make(map[string]*repository.UserStats),
		repositories: make(map[string]repository.UserRepository),
	}
}

// GetUserRepository returns a mock user repository
func (m *MockUserManager) GetUserRepository(userID string) repository.UserRepository {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if repo, exists := m.repositories[userID]; exists {
		return repo
	}

	// Return a new mock repository
	mockRepo := &MockUserRepository{userID: userID}
	m.repositories[userID] = mockRepo
	return mockRepo
}

// ValidateUserAccess validates if a user has access to a table
func (m *MockUserManager) ValidateUserAccess(ctx context.Context, userID string, tableID string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if _, exists := m.users[userID]; !exists {
		return repository.NewRepositoryError(repository.ErrorTypeNotFound, "user not found", nil)
	}
	return nil
}

// CreateUser creates a new user
func (m *MockUserManager) CreateUser(ctx context.Context, userID string, email string, password string) (*repository.User, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if user already exists
	if _, exists := m.users[userID]; exists {
		return nil, repository.NewRepositoryError(repository.ErrorTypeAlreadyExists, "user already exists", nil)
	}

	// Validate input
	if userID == "" {
		return nil, repository.NewRepositoryError(repository.ErrorTypeInvalidInput, "user ID cannot be empty", auth.ErrEmptyUserID)
	}
	
	// Validate user ID format (alphanumeric and underscores only, 3-50 characters)
	userIDRegex := regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	if !userIDRegex.MatchString(userID) {
		return nil, repository.NewRepositoryError(repository.ErrorTypeInvalidInput, "invalid user ID format", auth.ErrInvalidUserID)
	}
	
	if email == "" {
		return nil, repository.NewRepositoryError(repository.ErrorTypeInvalidInput, "email cannot be empty", auth.ErrEmptyEmail)
	}
	if password == "" {
		return nil, repository.NewRepositoryError(repository.ErrorTypeInvalidInput, "password cannot be empty", auth.ErrEmptyPassword)
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(password); err != nil {
		return nil, repository.NewRepositoryError(repository.ErrorTypeInvalidInput, "invalid password", err)
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, repository.NewRepositoryError(repository.ErrorTypeInternal, "failed to hash password", err)
	}

	now := time.Now()
	user := &repository.User{
		UserID:       userID,
		Email:        email,
		PasswordHash: hashedPassword,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastActive:   now,
	}

	// Store user
	m.users[userID] = user

	// Create user stats
	stats := &repository.UserStats{
		UserID:      userID,
		Email:       email,
		TableCount:  0,
		EntityCount: 0,
		LastActive:  now,
		CreatedAt:   now,
	}
	m.userStats[userID] = stats

	return user, nil
}

// GetUser retrieves a user by userID
func (m *MockUserManager) GetUser(ctx context.Context, userID string) (*repository.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, repository.NewRepositoryError(repository.ErrorTypeNotFound, "user not found", nil)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (m *MockUserManager) GetUserByEmail(ctx context.Context, email string) (*repository.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, repository.NewRepositoryError(repository.ErrorTypeNotFound, "user not found", nil)
}

// VerifyPassword verifies a password for a user
func (m *MockUserManager) VerifyPassword(ctx context.Context, userID string, password string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return false, repository.NewRepositoryError(repository.ErrorTypeNotFound, "user not found", nil)
	}

	return auth.VerifyPassword(password, user.PasswordHash), nil
}

// UpdatePassword updates a user's password
func (m *MockUserManager) UpdatePassword(ctx context.Context, userID string, newPassword string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return repository.NewRepositoryError(repository.ErrorTypeNotFound, "user not found", nil)
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(newPassword); err != nil {
		return repository.NewRepositoryError(repository.ErrorTypeInvalidInput, "invalid password", err)
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(newPassword)
	if err != nil {
		return repository.NewRepositoryError(repository.ErrorTypeInternal, "failed to hash password", err)
	}

	// Update user
	user.PasswordHash = hashedPassword
	user.UpdatedAt = time.Now()
	m.users[userID] = user

	return nil
}

// DeleteUser deletes a user
func (m *MockUserManager) DeleteUser(ctx context.Context, userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.users[userID]; !exists {
		return repository.NewRepositoryError(repository.ErrorTypeNotFound, "user not found", nil)
	}

	delete(m.users, userID)
	delete(m.userStats, userID)
	delete(m.repositories, userID)

	return nil
}

// GetUserStats retrieves user statistics
func (m *MockUserManager) GetUserStats(ctx context.Context, userID string) (*repository.UserStats, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats, exists := m.userStats[userID]
	if !exists {
		return nil, repository.NewRepositoryError(repository.ErrorTypeNotFound, "user stats not found", nil)
	}

	return stats, nil
}

// Health performs a health check
func (m *MockUserManager) Health(ctx context.Context) error {
	return nil
}

// MockUserRepository implements repository.UserRepository for testing
type MockUserRepository struct {
	userID string
}

// For testing purposes, we'll implement a minimal mock that returns empty results
// In a real test, you might want to implement these methods more fully

func (r *MockUserRepository) CreateTable(ctx context.Context, tableID string, fields []models.FieldDefinition) (*models.TableSchema, error) {
	// Return a basic table schema for testing
	return &models.TableSchema{
		ID:        tableID,
		UserID:    r.userID,
		Fields:    fields,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (r *MockUserRepository) GetTable(ctx context.Context, tableID string) (*models.TableSchema, error) {
	return nil, repository.NewRepositoryError(repository.ErrorTypeNotFound, "table not found", nil)
}

func (r *MockUserRepository) ListTables(ctx context.Context) ([]models.TableSchema, error) {
	return []models.TableSchema{}, nil
}

func (r *MockUserRepository) UpdateTable(ctx context.Context, table models.TableSchema) error {
	return nil
}

func (r *MockUserRepository) DeleteTable(ctx context.Context, tableID string) error {
	return nil
}

func (r *MockUserRepository) GetTableHistory(ctx context.Context, tableID string, opts models.QueryOptions) (*models.TableHistory, error) {
	return &models.TableHistory{}, nil
}

func (r *MockUserRepository) CreateEntity(ctx context.Context, tableID string, fields map[string]interface{}) (*models.EntitySnapshot, error) {
	return &models.EntitySnapshot{}, nil
}

func (r *MockUserRepository) GetEntity(ctx context.Context, tableID string, entityID string, asOf *time.Time) (*models.EntitySnapshot, error) {
	return nil, repository.NewRepositoryError(repository.ErrorTypeNotFound, "entity not found", nil)
}

func (r *MockUserRepository) GetAllEntities(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error) {
	return &models.EntitiesSnapshot{
		Entities: map[string]models.EntitySnapshot{},
	}, nil
}

func (r *MockUserRepository) GetAllEntitiesIncludingDeleted(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error) {
	return &models.EntitiesSnapshot{
		Entities: map[string]models.EntitySnapshot{},
	}, nil
}

func (r *MockUserRepository) UpdateEntity(ctx context.Context, tableID string, entityID string, fields map[string]interface{}) (*models.EntitySnapshot, error) {
	return &models.EntitySnapshot{}, nil
}

func (r *MockUserRepository) DeleteEntity(ctx context.Context, tableID string, entityID string) error {
	return nil
}

func (r *MockUserRepository) UndeleteEntity(ctx context.Context, tableID string, entityID string) error {
	return nil
}

func (r *MockUserRepository) DeleteField(ctx context.Context, tableID string, entityID string, fieldName string) error {
	return nil
}

func (r *MockUserRepository) GetFieldHistory(ctx context.Context, tableID string, fieldName string, opts models.QueryOptions) (*models.FieldHistory, error) {
	return &models.FieldHistory{}, nil
}

func (r *MockUserRepository) GetEntityHistory(ctx context.Context, tableID string, entityID string, opts models.QueryOptions) (*models.QueryResult, error) {
	return &models.QueryResult{}, nil
}

func (r *MockUserRepository) HealthCheck(ctx context.Context) error {
	return nil
}
