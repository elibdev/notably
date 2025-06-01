package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elibdev/notably/internal/api"
	"github.com/elibdev/notably/internal/api/models"
	"github.com/elibdev/notably/internal/config"
	"github.com/elibdev/notably/tests/testutils"
)

func TestAuthenticationE2E(t *testing.T) {
	// Setup test server
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Auth: config.AuthConfig{
			JWTSecret:   "test-secret-key-for-e2e-tests",
			TokenExpiry: time.Hour,
		},
		Database: config.DatabaseConfig{
			Provider:    "dynamodb",
			TableName:   "test-timedb-e2e",
			Region:      "us-east-1",
			EndpointURL: "http://localhost:8000",
		},
		Logging: config.LoggingConfig{
			Level:  "error", // Reduce noise in tests
			Format: "json",
		},
	}

	userManager := testutils.NewMockUserManager()
	server := api.NewServer(cfg, userManager)

	testServer := httptest.NewServer(server.GetRouter())
	defer testServer.Close()

	baseURL := testServer.URL

	t.Run("User Registration Tests", func(t *testing.T) {
		testUserRegistration(t, baseURL)
	})

	t.Run("User Login Tests", func(t *testing.T) {
		testUserLogin(t, baseURL)
	})

	t.Run("Password Validation Tests", func(t *testing.T) {
		testPasswordValidation(t, baseURL)
	})

	t.Run("JWT Token Tests", func(t *testing.T) {
		testJWTTokens(t, baseURL)
	})

	t.Run("Protected Endpoint Tests", func(t *testing.T) {
		testProtectedEndpoints(t, baseURL)
	})

	t.Run("Error Handling Tests", func(t *testing.T) {
		testErrorHandling(t, baseURL)
	})
}

func testUserRegistration(t *testing.T, baseURL string) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectToken    bool
		expectError    bool
		errorContains  string
	}{
		{
			name: "Valid Registration",
			requestBody: map[string]interface{}{
				"user_id":  "valid_user_123",
				"email":    "valid@example.com",
				"password": "securepassword123",
			},
			expectedStatus: http.StatusCreated,
			expectToken:    true,
			expectError:    false,
		},
		{
			name: "Duplicate User Registration",
			requestBody: map[string]interface{}{
				"user_id":  "valid_user_123", // Same as above
				"email":    "another@example.com",
				"password": "anotherpassword123",
			},
			expectedStatus: http.StatusConflict,
			expectToken:    false,
			expectError:    true,
			errorContains:  "already exists",
		},
		{
			name: "Invalid Email Format",
			requestBody: map[string]interface{}{
				"user_id":  "email_test_user",
				"email":    "invalid-email-format",
				"password": "validpassword123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
			errorContains:  "email",
		},
		{
			name: "Password Too Short",
			requestBody: map[string]interface{}{
				"user_id":  "short_pass_user",
				"email":    "shortpass@example.com",
				"password": "123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
			errorContains:  "Password",
		},
		{
			name: "Missing User ID",
			requestBody: map[string]interface{}{
				"email":    "missing@example.com",
				"password": "validpassword123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
		{
			name: "Missing Email",
			requestBody: map[string]interface{}{
				"user_id":  "missing_email_user",
				"password": "validpassword123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
		{
			name: "Missing Password",
			requestBody: map[string]interface{}{
				"user_id": "missing_pass_user",
				"email":   "missingpass@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
		{
			name: "Invalid User ID Format",
			requestBody: map[string]interface{}{
				"user_id":  "invalid-user-id!", // Contains invalid characters
				"email":    "invaliduserid@example.com",
				"password": "validpassword123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
		{
			name: "User ID Too Short",
			requestBody: map[string]interface{}{
				"user_id":  "ab", // Less than 3 characters
				"email":    "shortid@example.com",
				"password": "validpassword123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			resp, err := http.Post(
				baseURL+"/api/v1/auth/register",
				"application/json",
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.expectToken {
				var authResp map[string]interface{}
				err := json.Unmarshal(respBody, &authResp)
				require.NoError(t, err)

				assert.Contains(t, authResp, "token")
				assert.Contains(t, authResp, "user_id")
				assert.Contains(t, authResp, "expires_at")

				token, ok := authResp["token"].(string)
				assert.True(t, ok)
				assert.NotEmpty(t, token)
				assert.True(t, strings.HasPrefix(token, "eyJ")) // JWT starts with eyJ
			}

			if tt.expectError {
				var errorResp models.ErrorResponse
				err := json.Unmarshal(respBody, &errorResp)
				require.NoError(t, err)

				assert.NotEmpty(t, errorResp.Error)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(errorResp.Error), strings.ToLower(tt.errorContains))
				}
			}
		})
	}
}

func testUserLogin(t *testing.T, baseURL string) {
	// First register a test user
	registerBody := map[string]interface{}{
		"user_id":  "login_test_user",
		"email":    "logintest@example.com",
		"password": "loginpassword123",
	}
	body, _ := json.Marshal(registerBody)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectToken    bool
		expectError    bool
		errorContains  string
	}{
		{
			name: "Valid Login",
			requestBody: map[string]interface{}{
				"user_id":  "login_test_user",
				"password": "loginpassword123",
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
			expectError:    false,
		},
		{
			name: "Wrong Password",
			requestBody: map[string]interface{}{
				"user_id":  "login_test_user",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
			expectError:    true,
			errorContains:  "credentials",
		},
		{
			name: "Non-existent User",
			requestBody: map[string]interface{}{
				"user_id":  "nonexistent_user",
				"password": "anypassword123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
			expectError:    true,
			errorContains:  "credentials",
		},
		{
			name: "Missing User ID",
			requestBody: map[string]interface{}{
				"password": "loginpassword123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
		{
			name: "Missing Password",
			requestBody: map[string]interface{}{
				"user_id": "login_test_user",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
		{
			name: "Empty User ID",
			requestBody: map[string]interface{}{
				"user_id":  "",
				"password": "loginpassword123",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
		{
			name: "Empty Password",
			requestBody: map[string]interface{}{
				"user_id":  "login_test_user",
				"password": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			resp, err := http.Post(
				baseURL+"/api/v1/auth/login",
				"application/json",
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.expectToken {
				var authResp map[string]interface{}
				err := json.Unmarshal(respBody, &authResp)
				require.NoError(t, err)

				assert.Contains(t, authResp, "token")
				assert.Contains(t, authResp, "user_id")
				assert.Contains(t, authResp, "expires_at")

				token, ok := authResp["token"].(string)
				assert.True(t, ok)
				assert.NotEmpty(t, token)
			}

			if tt.expectError {
				var errorResp models.ErrorResponse
				err := json.Unmarshal(respBody, &errorResp)
				require.NoError(t, err)

				assert.NotEmpty(t, errorResp.Error)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(errorResp.Error), strings.ToLower(tt.errorContains))
				}
			}
		})
	}
}

func testPasswordValidation(t *testing.T, baseURL string) {
	tests := []struct {
		name           string
		password       string
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "Exactly 8 Characters",
			password:       "12345678",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Strong Password",
			password:       "StrongP@ssw0rd!",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Unicode Password",
			password:       "пароль测试密码123",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "7 Characters (Too Short)",
			password:       "1234567",
			expectedStatus: http.StatusBadRequest,
			errorContains:  "min",
		},
		{
			name:           "Empty Password",
			password:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Very Long Valid Password",
			password:       strings.Repeat("a", 70), // Within 72-byte limit
			expectedStatus: http.StatusCreated,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := map[string]interface{}{
				"user_id":  fmt.Sprintf("password_test_%d", i),
				"email":    fmt.Sprintf("passwordtest%d@example.com", i),
				"password": tt.password,
			}

			body, _ := json.Marshal(requestBody)
			resp, err := http.Post(
				baseURL+"/api/v1/auth/register",
				"application/json",
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus != http.StatusCreated {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var errorResp models.ErrorResponse
				err = json.Unmarshal(respBody, &errorResp)
				require.NoError(t, err)

				assert.NotEmpty(t, errorResp.Error)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(errorResp.Error), strings.ToLower(tt.errorContains))
				}
			}
		})
	}
}

func testJWTTokens(t *testing.T, baseURL string) {
	// Register and login to get a token
	registerBody := map[string]interface{}{
		"user_id":  "jwt_test_user",
		"email":    "jwttest@example.com",
		"password": "jwtpassword123",
	}

	body, _ := json.Marshal(registerBody)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var authResp map[string]interface{}
	err = json.Unmarshal(respBody, &authResp)
	require.NoError(t, err)

	token, ok := authResp["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, token)

	t.Run("Token Format Validation", func(t *testing.T) {
		// JWT should have 3 parts separated by dots
		parts := strings.Split(token, ".")
		assert.Len(t, parts, 3, "JWT should have 3 parts")

		// Each part should be base64 encoded (non-empty)
		for i, part := range parts {
			assert.NotEmpty(t, part, fmt.Sprintf("JWT part %d should not be empty", i+1))
		}
	})

	t.Run("Token Expiration", func(t *testing.T) {
		expiresAtStr, ok := authResp["expires_at"].(string)
		require.True(t, ok)

		expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
		require.NoError(t, err)

		// Token should expire in the future
		assert.True(t, expiresAt.After(time.Now()), "Token should expire in the future")

		// Token should expire within reasonable time (1 hour in our config)
		assert.True(t, expiresAt.Before(time.Now().Add(2*time.Hour)), "Token should expire within 2 hours")
	})

	t.Run("User ID in Response", func(t *testing.T) {
		userID, ok := authResp["user_id"].(string)
		require.True(t, ok)
		assert.Equal(t, "jwt_test_user", userID)
	})
}

func testProtectedEndpoints(t *testing.T, baseURL string) {
	// Register and login to get a token
	registerBody := map[string]interface{}{
		"user_id":  "protected_test_user",
		"email":    "protectedtest@example.com",
		"password": "protectedpassword123",
	}

	body, _ := json.Marshal(registerBody)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var authResp map[string]interface{}
	err = json.Unmarshal(respBody, &authResp)
	require.NoError(t, err)

	validToken, ok := authResp["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, validToken)

	tests := []struct {
		name           string
		endpoint       string
		method         string
		token          string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Access with Valid Token",
			endpoint:       "/api/v1/tables",
			method:         "GET",
			token:          validToken,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Access without Token",
			endpoint:       "/api/v1/tables",
			method:         "GET",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "Access with Invalid Token",
			endpoint:       "/api/v1/tables",
			method:         "GET",
			token:          "invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "Access with Malformed Token",
			endpoint:       "/api/v1/tables",
			method:         "GET",
			token:          "notajwttoken",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, baseURL+tt.endpoint, nil)
			require.NoError(t, err)

			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectError {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var errorResp models.ErrorResponse
				err = json.Unmarshal(respBody, &errorResp)
				require.NoError(t, err)

				assert.NotEmpty(t, errorResp.Error)
			}
		})
	}
}

func testErrorHandling(t *testing.T, baseURL string) {
	tests := []struct {
		name           string
		endpoint       string
		method         string
		contentType    string
		body           string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Invalid JSON in Registration",
			endpoint:       "/api/v1/auth/register",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"user_id": "test", "email": "test@example.com", "password": `,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Invalid Content Type",
			endpoint:       "/api/v1/auth/register",
			method:         "POST",
			contentType:    "text/plain",
			body:           `user_id=test&email=test@example.com&password=testpass123`,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Empty Request Body",
			endpoint:       "/api/v1/auth/register",
			method:         "POST",
			contentType:    "application/json",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Non-existent Endpoint",
			endpoint:       "/api/v1/auth/nonexistent",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"test": "data"}`,
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, baseURL+tt.endpoint, strings.NewReader(tt.body))
			require.NoError(t, err)

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectError {
				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				// For some errors, we might not get JSON response
				if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
					var errorResp models.ErrorResponse
					err = json.Unmarshal(respBody, &errorResp)
					if err == nil {
						assert.NotEmpty(t, errorResp.Error)
					}
				}
			}
		})
	}
}

// Helper function to create a unique user for testing
func createTestUser(t *testing.T, baseURL string, userID string) (string, string) {
	registerBody := map[string]interface{}{
		"user_id":  userID,
		"email":    fmt.Sprintf("%s@example.com", userID),
		"password": "testpassword123",
	}

	body, _ := json.Marshal(registerBody)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var authResp map[string]interface{}
	err = json.Unmarshal(respBody, &authResp)
	require.NoError(t, err)

	token, ok := authResp["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, token)

	return token, "testpassword123"
}
