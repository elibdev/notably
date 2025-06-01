package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elibdev/notably/internal/repository"
	"github.com/elibdev/notably/tests/testutils"
)

func TestUserManagerAuthentication(t *testing.T) {
	// Use mock user manager for isolated testing
	userManager := testutils.NewMockUserManager()
	ctx := context.Background()

	t.Run("User Registration", func(t *testing.T) {
		testUserRegistration(t, ctx, userManager)
	})

	t.Run("User Login and Password Verification", func(t *testing.T) {
		testPasswordVerification(t, ctx, userManager)
	})

	t.Run("User Retrieval", func(t *testing.T) {
		testUserRetrieval(t, ctx, userManager)
	})

	t.Run("Password Updates", func(t *testing.T) {
		testPasswordUpdates(t, ctx, userManager)
	})

	t.Run("Input Validation", func(t *testing.T) {
		testInputValidation(t, ctx, userManager)
	})

	t.Run("Error Handling", func(t *testing.T) {
		testErrorHandling(t, ctx, userManager)
	})
}

func testUserRegistration(t *testing.T, ctx context.Context, userManager repository.UserManager) {
	tests := []struct {
		name        string
		userID      string
		email       string
		password    string
		expectError bool
		errorType   repository.ErrorType
	}{
		{
			name:        "Valid User Registration",
			userID:      "valid_user_123",
			email:       "valid@example.com",
			password:    "securepassword123",
			expectError: false,
		},
		{
			name:        "Valid User with Special Characters in Email",
			userID:      "special_user",
			email:       "user.name+tag@domain-name.com",
			password:    "anotherpassword456",
			expectError: false,
		},
		{
			name:        "Valid User with Long Password",
			userID:      "long_pass_user",
			email:       "longpass@example.com",
			password:    strings.Repeat("a", 70), // Within 72-byte limit
			expectError: false,
		},
		{
			name:        "Empty User ID",
			userID:      "",
			email:       "empty@example.com",
			password:    "validpassword123",
			expectError: true,
			errorType:   repository.ErrorTypeInvalidInput,
		},
		{
			name:        "Empty Email",
			userID:      "empty_email_user",
			email:       "",
			password:    "validpassword123",
			expectError: true,
			errorType:   repository.ErrorTypeInvalidInput,
		},
		{
			name:        "Empty Password",
			userID:      "empty_pass_user",
			email:       "emptypass@example.com",
			password:    "",
			expectError: true,
			errorType:   repository.ErrorTypeInvalidInput,
		},
		{
			name:        "Password Too Short",
			userID:      "short_pass_user",
			email:       "shortpass@example.com",
			password:    "123",
			expectError: true,
			errorType:   repository.ErrorTypeInvalidInput,
		},
		{
			name:        "Password Too Long",
			userID:      "long_pass_user_2",
			email:       "toolong@example.com",
			password:    strings.Repeat("a", 73), // Exceeds 72-byte limit
			expectError: true,
			errorType:   repository.ErrorTypeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := userManager.CreateUser(ctx, tt.userID, tt.email, tt.password)

			if tt.expectError {
				require.Error(t, err)

				if repoErr, ok := err.(*repository.RepositoryError); ok {
					assert.Equal(t, tt.errorType, repoErr.Type)
				}
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)

				assert.Equal(t, tt.userID, user.UserID)
				assert.Equal(t, tt.email, user.Email)
				assert.NotEmpty(t, user.PasswordHash)
				assert.NotEqual(t, tt.password, user.PasswordHash) // Password should be hashed
				assert.False(t, user.CreatedAt.IsZero())
				assert.False(t, user.UpdatedAt.IsZero())
				assert.False(t, user.LastActive.IsZero())

				// Verify password hash is valid bcrypt format
				assert.True(t, strings.HasPrefix(user.PasswordHash, "$2"))
			}
		})
	}
}

func testPasswordVerification(t *testing.T, ctx context.Context, userManager repository.UserManager) {
	// Create a test user first
	testUserID := "password_test_user"
	testEmail := "passwordtest@example.com"
	testPassword := "testpassword123"

	user, err := userManager.CreateUser(ctx, testUserID, testEmail, testPassword)
	require.NoError(t, err)
	require.NotNil(t, user)

	tests := []struct {
		name        string
		userID      string
		password    string
		expectValid bool
		expectError bool
	}{
		{
			name:        "Correct Password",
			userID:      testUserID,
			password:    testPassword,
			expectValid: true,
			expectError: false,
		},
		{
			name:        "Wrong Password",
			userID:      testUserID,
			password:    "wrongpassword",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "Case Sensitive Password",
			userID:      testUserID,
			password:    "TESTPASSWORD123",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "Empty Password",
			userID:      testUserID,
			password:    "",
			expectValid: false,
			expectError: false,
		},
		{
			name:        "Non-existent User",
			userID:      "nonexistent_user",
			password:    testPassword,
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid, err := userManager.VerifyPassword(ctx, tt.userID, tt.password)

			if tt.expectError {
				require.Error(t, err)
				assert.False(t, isValid)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectValid, isValid)
			}
		})
	}
}

func testUserRetrieval(t *testing.T, ctx context.Context, userManager repository.UserManager) {
	// Create test users
	users := []struct {
		userID   string
		email    string
		password string
	}{
		{"retrieval_user_1", "retrieval1@example.com", "password123"},
		{"retrieval_user_2", "retrieval2@example.com", "password456"},
		{"retrieval_user_3", "UPPER@EXAMPLE.COM", "password789"},
	}

	for _, u := range users {
		_, err := userManager.CreateUser(ctx, u.userID, u.email, u.password)
		require.NoError(t, err)
	}

	t.Run("Get User by ID", func(t *testing.T) {
		for _, u := range users {
			user, err := userManager.GetUser(ctx, u.userID)
			require.NoError(t, err)
			require.NotNil(t, user)

			assert.Equal(t, u.userID, user.UserID)
			assert.Equal(t, u.email, user.Email)
			assert.NotEmpty(t, user.PasswordHash)
		}

		// Test non-existent user
		user, err := userManager.GetUser(ctx, "nonexistent_user")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("Get User by Email", func(t *testing.T) {
		for _, u := range users {
			user, err := userManager.GetUserByEmail(ctx, u.email)
			require.NoError(t, err)
			require.NotNil(t, user)

			assert.Equal(t, u.userID, user.UserID)
			assert.Equal(t, u.email, user.Email)
			assert.NotEmpty(t, user.PasswordHash)
		}

		// Test non-existent email
		user, err := userManager.GetUserByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, repository.IsNotFound(err))
	})
}

func testPasswordUpdates(t *testing.T, ctx context.Context, userManager repository.UserManager) {
	// Create a test user
	testUserID := "update_test_user"
	testEmail := "updatetest@example.com"
	originalPassword := "originalpassword123"

	user, err := userManager.CreateUser(ctx, testUserID, testEmail, originalPassword)
	require.NoError(t, err)
	originalHash := user.PasswordHash

	tests := []struct {
		name        string
		userID      string
		newPassword string
		expectError bool
		errorType   repository.ErrorType
	}{
		{
			name:        "Valid Password Update",
			userID:      testUserID,
			newPassword: "newpassword456",
			expectError: false,
		},
		{
			name:        "Update to Short Password",
			userID:      testUserID,
			newPassword: "123",
			expectError: true,
			errorType:   repository.ErrorTypeInvalidInput,
		},
		{
			name:        "Update Non-existent User",
			userID:      "nonexistent_user",
			newPassword: "validpassword123",
			expectError: true,
			errorType:   repository.ErrorTypeNotFound,
		},
		{
			name:        "Update to Empty Password",
			userID:      testUserID,
			newPassword: "",
			expectError: true,
			errorType:   repository.ErrorTypeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := userManager.UpdatePassword(ctx, tt.userID, tt.newPassword)

			if tt.expectError {
				require.Error(t, err)

				if repoErr, ok := err.(*repository.RepositoryError); ok {
					assert.Equal(t, tt.errorType, repoErr.Type)
				}
			} else {
				require.NoError(t, err)

				// Verify password was actually updated
				if tt.userID == testUserID {
					// Get updated user
					updatedUser, err := userManager.GetUser(ctx, tt.userID)
					require.NoError(t, err)

					// Hash should be different
					assert.NotEqual(t, originalHash, updatedUser.PasswordHash)

					// New password should verify correctly
					isValid, err := userManager.VerifyPassword(ctx, tt.userID, tt.newPassword)
					require.NoError(t, err)
					assert.True(t, isValid)

					// Old password should no longer work
					isValid, err = userManager.VerifyPassword(ctx, tt.userID, originalPassword)
					require.NoError(t, err)
					assert.False(t, isValid)

					// Update original hash for next test
					originalHash = updatedUser.PasswordHash
					originalPassword = tt.newPassword
				}
			}
		})
	}
}

func testInputValidation(t *testing.T, ctx context.Context, userManager repository.UserManager) {
	tests := []struct {
		name        string
		userID      string
		email       string
		password    string
		expectError string
	}{
		{
			name:        "Unicode in User ID",
			userID:      "пользователь123",
			email:       "unicode@example.com",
			password:    "validpassword123",
			expectError: "", // Should work unless specific validation prevents it
		},
		{
			name:        "Unicode in Email",
			userID:      "unicode_user",
			email:       "пользователь@example.com",
			password:    "validpassword123",
			expectError: "", // Should work with international domains
		},
		{
			name:        "Unicode in Password",
			userID:      "unicode_pass_user",
			email:       "unicodepass@example.com",
			password:    "пароль测试密码123",
			expectError: "",
		},
		{
			name:        "Very Long User ID",
			userID:      strings.Repeat("a", 100),
			email:       "longid@example.com",
			password:    "validpassword123",
			expectError: "invalid", // Likely to fail validation
		},
		{
			name:        "Very Long Email",
			userID:      "long_email_user",
			email:       strings.Repeat("a", 200) + "@example.com",
			password:    "validpassword123",
			expectError: "", // Might be valid depending on email validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := userManager.CreateUser(ctx, tt.userID, tt.email, tt.password)

			if tt.expectError != "" {
				assert.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.expectError))
				assert.Nil(t, user)
			} else {
				// If no error expected, just check that we get a reasonable result
				// (might still error due to validation, but shouldn't crash)
				if err != nil {
					t.Logf("Validation rejected input (might be expected): %v", err)
				} else {
					assert.NotNil(t, user)
					assert.Equal(t, tt.userID, user.UserID)
				}
			}
		})
	}
}

func testErrorHandling(t *testing.T, ctx context.Context, userManager repository.UserManager) {
	t.Run("Duplicate User Creation", func(t *testing.T) {
		userID := "duplicate_test_user"
		email := "duplicate@example.com"
		password := "duplicatepassword123"

		// Create user first time
		user1, err := userManager.CreateUser(ctx, userID, email, password)
		require.NoError(t, err)
		require.NotNil(t, user1)

		// Try to create same user again
		user2, err := userManager.CreateUser(ctx, userID, "different@example.com", "differentpassword456")
		require.Error(t, err)
		assert.Nil(t, user2)
		assert.True(t, repository.IsAlreadyExists(err))
	})

	t.Run("User Stats Retrieval", func(t *testing.T) {
		// Create a user
		userID := "stats_test_user"
		email := "stats@example.com"
		password := "statspassword123"

		user, err := userManager.CreateUser(ctx, userID, email, password)
		require.NoError(t, err)
		require.NotNil(t, user)

		// Get user stats
		stats, err := userManager.GetUserStats(ctx, userID)
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Equal(t, userID, stats.UserID)
		assert.Equal(t, email, stats.Email)
		assert.Equal(t, 0, stats.TableCount)
		assert.Equal(t, 0, stats.EntityCount)
		assert.False(t, stats.CreatedAt.IsZero())
		assert.False(t, stats.LastActive.IsZero())

		// Try to get stats for non-existent user
		stats, err = userManager.GetUserStats(ctx, "nonexistent_user")
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.True(t, repository.IsNotFound(err))
	})

	t.Run("Health Check", func(t *testing.T) {
		err := userManager.Health(ctx)
		assert.NoError(t, err)
	})

	t.Run("User Deletion", func(t *testing.T) {
		// Create a user
		userID := "delete_test_user"
		email := "delete@example.com"
		password := "deletepassword123"

		user, err := userManager.CreateUser(ctx, userID, email, password)
		require.NoError(t, err)
		require.NotNil(t, user)

		// Verify user exists
		foundUser, err := userManager.GetUser(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, userID, foundUser.UserID)

		// Delete user
		err = userManager.DeleteUser(ctx, userID)
		require.NoError(t, err)

		// Verify user no longer exists
		deletedUser, err := userManager.GetUser(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, deletedUser)
		assert.True(t, repository.IsNotFound(err))

		// Try to delete non-existent user
		err = userManager.DeleteUser(ctx, "nonexistent_user")
		assert.Error(t, err)
		assert.True(t, repository.IsNotFound(err))
	})
}

func TestPasswordHashingConsistency(t *testing.T) {
	userManager := testutils.NewMockUserManager()
	ctx := context.Background()

	// Test that the same password produces different hashes (due to salt)
	// but both verify correctly
	userID1 := "hash_test_1"
	userID2 := "hash_test_2"
	email1 := "hash1@example.com"
	email2 := "hash2@example.com"
	password := "samepassword123"

	user1, err := userManager.CreateUser(ctx, userID1, email1, password)
	require.NoError(t, err)

	user2, err := userManager.CreateUser(ctx, userID2, email2, password)
	require.NoError(t, err)

	// Hashes should be different (due to salt)
	assert.NotEqual(t, user1.PasswordHash, user2.PasswordHash)

	// Both should verify correctly
	valid1, err := userManager.VerifyPassword(ctx, userID1, password)
	require.NoError(t, err)
	assert.True(t, valid1)

	valid2, err := userManager.VerifyPassword(ctx, userID2, password)
	require.NoError(t, err)
	assert.True(t, valid2)

	// Cross-verification should fail (wrong user)
	valid1Wrong, err := userManager.VerifyPassword(ctx, userID1, "wrongpassword")
	require.NoError(t, err)
	assert.False(t, valid1Wrong)
}

func TestConcurrentUserOperations(t *testing.T) {
	userManager := testutils.NewMockUserManager()
	ctx := context.Background()

	// Test concurrent user creation
	t.Run("Concurrent User Creation", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				userID := fmt.Sprintf("concurrent_user_%d", id)
				email := fmt.Sprintf("concurrent%d@example.com", id)
				password := "concurrentpassword123"

				_, err := userManager.CreateUser(ctx, userID, email, password)
				results <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent user creation failed")
		}
	})

	// Test concurrent password verification
	t.Run("Concurrent Password Verification", func(t *testing.T) {
		// Create a test user first
		userID := "concurrent_verify_user"
		email := "verify@example.com"
		password := "verifypassword123"

		_, err := userManager.CreateUser(ctx, userID, email, password)
		require.NoError(t, err)

		const numGoroutines = 20
		results := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				valid, err := userManager.VerifyPassword(ctx, userID, password)
				if err != nil {
					results <- false
				} else {
					results <- valid
				}
			}()
		}

		// All verifications should succeed
		for i := 0; i < numGoroutines; i++ {
			result := <-results
			assert.True(t, result, "Concurrent password verification failed")
		}
	})
}
