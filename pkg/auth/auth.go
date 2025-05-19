package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// APIKeyLength is the number of bytes in a raw API key
	APIKeyLength = 32

	// APIKeyPrefix is the prefix for all API keys
	APIKeyPrefix = "nb_"

	// DefaultAPIKeyExpiration is the default expiration for API keys
	DefaultAPIKeyExpiration = 90 * 24 * time.Hour // 90 days
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrInvalidAPIKey         = errors.New("invalid API key")
	ErrAPIKeyExpired         = errors.New("API key expired")
	ErrAPIKeyRevoked         = errors.New("API key revoked")
	ErrInsufficientPrivilege = errors.New("insufficient privilege")
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	APIKeys      []*APIKey `json:"-"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Key       string    `json:"-"`
	KeyHash   string    `json:"-"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	LastUsed  time.Time `json:"lastUsed"`
	Revoked   bool      `json:"revoked"`
}

// UserStore is an interface for user data storage
type UserStore interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error

	// API Key methods
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKey(ctx context.Context, id string) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	UpdateAPIKey(ctx context.Context, key *APIKey) error
	DeleteAPIKey(ctx context.Context, id string) error
	ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error)
}

// InMemoryUserStore implements UserStore with in-memory storage
type InMemoryUserStore struct {
	mu        sync.RWMutex
	users     map[string]*User
	usernames map[string]string  // username -> userID
	emails    map[string]string  // email -> userID
	apiKeys   map[string]*APIKey // key hash -> APIKey
	apiKeyIDs map[string]*APIKey // key ID -> APIKey
}

// NewInMemoryUserStore creates a new in-memory user store
func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users:     make(map[string]*User),
		usernames: make(map[string]string),
		emails:    make(map[string]string),
		apiKeys:   make(map[string]*APIKey),
		apiKeyIDs: make(map[string]*APIKey),
	}
}

// Authenticator manages user authentication
type Authenticator struct {
	store UserStore
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(store UserStore) *Authenticator {
	return &Authenticator{store: store}
}

// RegisterUser registers a new user
func (a *Authenticator) RegisterUser(ctx context.Context, username, email, password string) (*User, error) {
	// Check if user already exists
	_, err := a.store.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	_, err = a.store.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &User{
		ID:           generateID(),
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    now,
		UpdatedAt:    now,
		APIKeys:      []*APIKey{},
	}

	if err := a.store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// LoginUser authenticates a user by username/email and password
func (a *Authenticator) LoginUser(ctx context.Context, usernameOrEmail, password string) (*User, error) {
	var user *User
	var err error

	// Try to find by username
	user, err = a.store.GetUserByUsername(ctx, usernameOrEmail)
	if err != nil {
		// Try to find by email
		user, err = a.store.GetUserByEmail(ctx, usernameOrEmail)
		if err != nil {
			return nil, ErrInvalidCredentials
		}
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// GenerateAPIKey creates a new API key for a user
func (a *Authenticator) GenerateAPIKey(ctx context.Context, userID, name string, duration time.Duration) (*APIKey, string, error) {
	// Verify user exists
	user, err := a.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	// Generate random key
	keyBytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Format the key with prefix and encode
	rawKey := fmt.Sprintf("%s%s", APIKeyPrefix, hex.EncodeToString(keyBytes))

	// Hash the key for storage
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash key: %w", err)
	}

	now := time.Now().UTC()
	if duration == 0 {
		duration = DefaultAPIKeyExpiration
	}

	apiKey := &APIKey{
		ID:        generateID(),
		UserID:    user.ID,
		KeyHash:   string(hashedKey),
		Name:      name,
		CreatedAt: now,
		ExpiresAt: now.Add(duration),
		LastUsed:  now,
		Revoked:   false,
	}

	if err := a.store.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("failed to save API key: %w", err)
	}

	return apiKey, rawKey, nil
}

// VerifyAPIKey verifies an API key and returns the associated user
func (a *Authenticator) VerifyAPIKey(ctx context.Context, apiKeyStr string) (*User, *APIKey, error) {
	// Validate format
	if !strings.HasPrefix(apiKeyStr, APIKeyPrefix) {
		return nil, nil, ErrInvalidAPIKey
	}

	// Get all API keys and check each one
	// This is inefficient but needed since we can't query by the raw key directly
	// In a real system, we'd use a more efficient lookup mechanism

	keys, err := a.GetAllAPIKeys(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	now := time.Now().UTC()

	for _, key := range keys {
		// Compare API key hash (slow but secure)
		if err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(apiKeyStr)); err == nil {
			// Key found, check if valid
			if key.Revoked {
				return nil, nil, ErrAPIKeyRevoked
			}

			if now.After(key.ExpiresAt) {
				return nil, nil, ErrAPIKeyExpired
			}

			// Update last used time
			key.LastUsed = now
			if err := a.store.UpdateAPIKey(ctx, key); err != nil {
				// Non-fatal error, just log it in a real implementation
			}

			// Get associated user
			user, err := a.store.GetUserByID(ctx, key.UserID)
			if err != nil {
				return nil, nil, fmt.Errorf("API key valid but user not found: %w", err)
			}

			return user, key, nil
		}
	}

	return nil, nil, ErrInvalidAPIKey
}

// GetAllAPIKeys returns all API keys (for internal use)
func (a *Authenticator) GetAllAPIKeys(ctx context.Context) ([]*APIKey, error) {
	// In a real system, this would be more efficient
	allUsers, err := a.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	var allKeys []*APIKey
	for _, user := range allUsers {
		keys, err := a.store.ListAPIKeys(ctx, user.ID)
		if err != nil {
			continue
		}
		allKeys = append(allKeys, keys...)
	}

	return allKeys, nil
}

// GetAllUsers returns all users (for internal use)
func (a *Authenticator) GetAllUsers(ctx context.Context) ([]*User, error) {
	// Simplified implementation for InMemoryUserStore
	if store, ok := a.store.(*InMemoryUserStore); ok {
		store.mu.RLock()
		defer store.mu.RUnlock()

		users := make([]*User, 0, len(store.users))
		for _, user := range store.users {
			users = append(users, user)
		}
		return users, nil
	}

	return nil, errors.New("operation not supported by this store implementation")
}

// RevokeAPIKey revokes an API key
func (a *Authenticator) RevokeAPIKey(ctx context.Context, userID, keyID string) error {
	key, err := a.store.GetAPIKey(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	// Verify ownership
	if key.UserID != userID {
		return ErrInsufficientPrivilege
	}

	key.Revoked = true
	return a.store.UpdateAPIKey(ctx, key)
}

// RefreshAPIKey extends the expiration of an API key
func (a *Authenticator) RefreshAPIKey(ctx context.Context, userID, keyID string, duration time.Duration) error {
	key, err := a.store.GetAPIKey(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	// Verify ownership
	if key.UserID != userID {
		return ErrInsufficientPrivilege
	}

	// Check if revoked
	if key.Revoked {
		return ErrAPIKeyRevoked
	}

	// Extend expiration
	if duration == 0 {
		duration = DefaultAPIKeyExpiration
	}
	key.ExpiresAt = time.Now().UTC().Add(duration)

	return a.store.UpdateAPIKey(ctx, key)
}

// ListAPIKeys lists all API keys for a user
func (a *Authenticator) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	return a.store.ListAPIKeys(ctx, userID)
}

// RequireAuth is a middleware that requires authentication via API key
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "unauthorized: missing API key", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer API_KEY"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "unauthorized: invalid authorization format", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]
		user, key, err := a.VerifyAPIKey(r.Context(), apiKey)
		if err != nil {
			switch err {
			case ErrAPIKeyExpired:
				http.Error(w, "unauthorized: API key expired", http.StatusUnauthorized)
			case ErrAPIKeyRevoked:
				http.Error(w, "unauthorized: API key revoked", http.StatusUnauthorized)
			default:
				http.Error(w, "unauthorized: invalid API key", http.StatusUnauthorized)
			}
			return
		}

		// Add user and key to context
		ctx := context.WithValue(r.Context(), contextKeyUser, user)
		ctx = context.WithValue(ctx, contextKeyAPIKey, key)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext extracts the user from the context
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(contextKeyUser).(*User)
	return user, ok
}

// APIKeyFromContext extracts the API key from the context
func APIKeyFromContext(ctx context.Context) (*APIKey, bool) {
	key, ok := ctx.Value(contextKeyAPIKey).(*APIKey)
	return key, ok
}

// Private helper functions and types

type contextKey string

const (
	contextKeyUser   contextKey = "user"
	contextKeyAPIKey contextKey = "apiKey"
)

// generateID generates a unique ID
func generateID() string {
	id := make([]byte, 16)
	_, err := rand.Read(id)
	if err != nil {
		panic(err) // This should never happen
	}
	return base64.RawURLEncoding.EncodeToString(id)
}

// InMemoryUserStore implementation

func (s *InMemoryUserStore) CreateUser(ctx context.Context, user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if username or email already exists
	if _, exists := s.usernames[user.Username]; exists {
		return ErrUserAlreadyExists
	}
	if _, exists := s.emails[user.Email]; exists {
		return ErrUserAlreadyExists
	}

	// Store the user
	s.users[user.ID] = user
	s.usernames[user.Username] = user.ID
	s.emails[user.Email] = user.ID

	return nil
}

func (s *InMemoryUserStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *InMemoryUserStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.usernames[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return s.users[id], nil
}

func (s *InMemoryUserStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.emails[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	return s.users[id], nil
}

func (s *InMemoryUserStore) UpdateUser(ctx context.Context, user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existingUser, exists := s.users[user.ID]
	if !exists {
		return ErrUserNotFound
	}

	// Update username/email maps if they changed
	if existingUser.Username != user.Username {
		delete(s.usernames, existingUser.Username)
		s.usernames[user.Username] = user.ID
	}

	if existingUser.Email != user.Email {
		delete(s.emails, existingUser.Email)
		s.emails[user.Email] = user.ID
	}

	// Update the user
	s.users[user.ID] = user

	return nil
}

func (s *InMemoryUserStore) DeleteUser(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return ErrUserNotFound
	}

	// Remove user from maps
	delete(s.usernames, user.Username)
	delete(s.emails, user.Email)
	delete(s.users, id)

	// Delete all associated API keys
	for _, key := range s.apiKeyIDs {
		if key.UserID == id {
			delete(s.apiKeys, key.KeyHash)
			delete(s.apiKeyIDs, key.ID)
		}
	}

	return nil
}

func (s *InMemoryUserStore) CreateAPIKey(ctx context.Context, key *APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify user exists
	if _, exists := s.users[key.UserID]; !exists {
		return ErrUserNotFound
	}

	s.apiKeys[key.KeyHash] = key
	s.apiKeyIDs[key.ID] = key

	return nil
}

func (s *InMemoryUserStore) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.apiKeyIDs[id]
	if !exists {
		return nil, errors.New("API key not found")
	}

	return key, nil
}

func (s *InMemoryUserStore) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.apiKeys[keyHash]
	if !exists {
		return nil, errors.New("API key not found")
	}

	return key, nil
}

func (s *InMemoryUserStore) UpdateAPIKey(ctx context.Context, key *APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apiKeyIDs[key.ID]; !exists {
		return errors.New("API key not found")
	}

	s.apiKeyIDs[key.ID] = key
	// No need to update the hash map since the hash doesn't change

	return nil
}

func (s *InMemoryUserStore) DeleteAPIKey(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.apiKeyIDs[id]
	if !exists {
		return errors.New("API key not found")
	}

	delete(s.apiKeyIDs, id)
	delete(s.apiKeys, key.KeyHash)

	return nil
}

func (s *InMemoryUserStore) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.users[userID]; !exists {
		return nil, ErrUserNotFound
	}

	keys := make([]*APIKey, 0)
	for _, key := range s.apiKeyIDs {
		if key.UserID == userID {
			keys = append(keys, key)
		}
	}

	return keys, nil
}
