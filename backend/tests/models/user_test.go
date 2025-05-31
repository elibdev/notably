package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/elibdev/notably/internal/models"
)

func TestNewUser(t *testing.T) {
	userID := "user_123"
	user := models.NewUser(userID)

	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "USER#"+userID, user.PK)
	assert.Equal(t, "USER", user.SK)
	assert.WithinDuration(t, time.Now(), user.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), user.UpdatedAt, time.Second)
}

func TestUserKeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		expected string
	}{
		{"basic user", "user_123", "USER#user_123"},
		{"email user", "user@example.com", "USER#user@example.com"},
		{"uuid user", "550e8400-e29b-41d4-a716-446655440000", "USER#550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := models.NewUser(tt.userID)
			assert.Equal(t, tt.expected, user.PK)
		})
	}
}

func TestUser_UpdateTimestamp(t *testing.T) {
	user := models.NewUser("test_user")
	originalTime := user.UpdatedAt

	// Wait a small amount to ensure time difference
	time.Sleep(10 * time.Millisecond)

	user.UpdateTimestamp()

	assert.True(t, user.UpdatedAt.After(originalTime))
	assert.WithinDuration(t, time.Now(), user.UpdatedAt, time.Second)
}

func TestUser_GetUserID(t *testing.T) {
	tests := []struct {
		name   string
		userID string
	}{
		{"simple ID", "user_123"},
		{"email ID", "test@example.com"},
		{"uuid ID", "550e8400-e29b-41d4-a716-446655440000"},
		{"complex ID", "user-with-dashes_and_underscores"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := models.NewUser(tt.userID)
			assert.Equal(t, tt.userID, user.GetUserID())
		})
	}
}

func TestUser_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		user     models.User
		expected bool
	}{
		{
			name:     "valid user",
			user:     models.NewUser("test_user"),
			expected: true,
		},
		{
			name: "missing ID",
			user: models.User{
				PK: "USER#test",
				SK: "USER",
			},
			expected: false,
		},
		{
			name: "missing PK",
			user: models.User{
				ID: "test_user",
				SK: "USER",
			},
			expected: false,
		},
		{
			name: "wrong SK",
			user: models.User{
				ID: "test_user",
				PK: "USER#test_user",
				SK: "WRONG",
			},
			expected: false,
		},
		{
			name: "empty user",
			user: models.User{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.IsValid())
		})
	}
}

func TestUser_FieldConsistency(t *testing.T) {
	userID := "consistency_test_user"
	user := models.NewUser(userID)

	// Test that all fields are consistent
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "USER#"+userID, user.PK)
	assert.Equal(t, "USER", user.SK)
	assert.Equal(t, user.CreatedAt, user.UpdatedAt) // Should be same at creation

	// Test that GetUserID returns the same as ID
	assert.Equal(t, user.ID, user.GetUserID())

	// Test that the user is valid
	assert.True(t, user.IsValid())
}

func TestUser_TimestampBehavior(t *testing.T) {
	user := models.NewUser("timestamp_test")

	// Initially, CreatedAt and UpdatedAt should be the same
	assert.Equal(t, user.CreatedAt, user.UpdatedAt)

	// After updating timestamp, UpdatedAt should change but CreatedAt should remain
	originalCreatedAt := user.CreatedAt
	time.Sleep(10 * time.Millisecond)
	user.UpdateTimestamp()

	assert.Equal(t, originalCreatedAt, user.CreatedAt) // CreatedAt shouldn't change
	assert.True(t, user.UpdatedAt.After(originalCreatedAt)) // UpdatedAt should be later
}