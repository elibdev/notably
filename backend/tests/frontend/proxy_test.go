package frontend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	FrontendURL = "http://localhost:3000"
	BackendURL  = "http://localhost:8080"
)

func TestFrontendProxyIntegration(t *testing.T) {
	// Skip if services aren't running
	if !isServiceRunning(FrontendURL) {
		t.Skip("Frontend not running on", FrontendURL)
	}
	if !isServiceRunning(BackendURL) {
		t.Skip("Backend not running on", BackendURL)
	}

	t.Run("Proxy Configuration", func(t *testing.T) {
		testProxyConfiguration(t)
	})

	t.Run("Authentication Flow", func(t *testing.T) {
		testAuthenticationFlow(t)
	})

	t.Run("Error Handling", func(t *testing.T) {
		testErrorHandling(t)
	})

	t.Run("Response Consistency", func(t *testing.T) {
		testResponseConsistency(t)
	})
}

func testProxyConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		method   string
		status   int
	}{
		{
			name:     "Health Check via Proxy",
			endpoint: "/api/v1/health",
			method:   "GET",
			status:   http.StatusOK,
		},
		{
			name:     "API Docs via Proxy",
			endpoint: "/api/v1/docs/",
			method:   "GET",
			status:   http.StatusOK,
		},
		{
			name:     "Non-existent Endpoint",
			endpoint: "/api/v1/nonexistent",
			method:   "GET",
			status:   http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, FrontendURL+tt.endpoint, nil)
			require.NoError(t, err)

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func testAuthenticationFlow(t *testing.T) {
	// Generate unique test data
	timestamp := time.Now().Unix()
	userID := fmt.Sprintf("proxy_test_%d", timestamp)
	email := fmt.Sprintf("proxy%d@example.com", timestamp)
	password := "proxypassword123"

	t.Run("Registration via Proxy", func(t *testing.T) {
		// Test registration through frontend proxy
		registerData := map[string]string{
			"user_id":  userID,
			"email":    email,
			"password": password,
		}

		proxyToken := testRegistration(t, FrontendURL, registerData)
		directToken := testRegistration(t, BackendURL, map[string]string{
			"user_id":  userID + "_direct",
			"email":    "direct" + email,
			"password": password,
		})

		// Both should return valid tokens
		assert.NotEmpty(t, proxyToken)
		assert.NotEmpty(t, directToken)

		// Tokens should be JWT format
		assert.Contains(t, proxyToken, ".")
		assert.Contains(t, directToken, ".")
	})

	t.Run("Login via Proxy", func(t *testing.T) {
		loginData := map[string]string{
			"user_id":  userID,
			"password": password,
		}

		proxyToken := testLogin(t, FrontendURL, loginData)

		// Should return valid token
		assert.NotEmpty(t, proxyToken)
		assert.Contains(t, proxyToken, ".")
	})

	t.Run("Invalid Login via Proxy", func(t *testing.T) {
		loginData := map[string]string{
			"user_id":  userID,
			"password": "wrongpassword",
		}

		body, _ := json.Marshal(loginData)
		resp, err := http.Post(FrontendURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var errorResp map[string]string
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp["error"], "credentials")
	})
}

func testErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		method         string
		body           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "Invalid JSON",
			endpoint:       "/api/v1/auth/register",
			method:         "POST",
			body:           `{"user_id": "test", "email": `,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing Content-Type",
			endpoint:       "/api/v1/auth/register",
			method:         "POST",
			body:           `{"user_id": "test", "email": "test@example.com", "password": "testpass123"}`,
			contentType:    "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty Body",
			endpoint:       "/api/v1/auth/register",
			method:         "POST",
			body:           "",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, FrontendURL+tt.endpoint, bytes.NewBufferString(tt.body))
			require.NoError(t, err)

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func testResponseConsistency(t *testing.T) {
	// Test that responses from proxy match direct backend responses
	timestamp := time.Now().Unix()

	// Test data for comparison
	testData := map[string]string{
		"user_id":  fmt.Sprintf("consistency_test_%d", timestamp),
		"email":    fmt.Sprintf("consistency%d@example.com", timestamp),
		"password": "consistencypass123",
	}

	// Make same request to both proxy and direct backend
	proxyResp := makeRegistrationRequest(t, FrontendURL, testData)
	directResp := makeRegistrationRequest(t, BackendURL, map[string]string{
		"user_id":  testData["user_id"] + "_direct",
		"email":    "direct" + testData["email"],
		"password": testData["password"],
	})

	// Compare response structure
	assert.Equal(t, proxyResp.StatusCode, directResp.StatusCode)

	if proxyResp.StatusCode == http.StatusCreated {
		var proxyData, directData map[string]interface{}

		err := json.NewDecoder(proxyResp.Body).Decode(&proxyData)
		require.NoError(t, err)

		err = json.NewDecoder(directResp.Body).Decode(&directData)
		require.NoError(t, err)

		// Both should have same structure
		assert.Contains(t, proxyData, "token")
		assert.Contains(t, proxyData, "user_id")
		assert.Contains(t, proxyData, "expires_at")

		assert.Contains(t, directData, "token")
		assert.Contains(t, directData, "user_id")
		assert.Contains(t, directData, "expires_at")
	}
}

// Helper functions

func isServiceRunning(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func testRegistration(t *testing.T, baseURL string, data map[string]string) string {
	body, _ := json.Marshal(data)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	token, ok := result["token"].(string)
	require.True(t, ok)
	return token
}

func testLogin(t *testing.T, baseURL string, data map[string]string) string {
	body, _ := json.Marshal(data)
	resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	token, ok := result["token"].(string)
	require.True(t, ok)
	return token
}

func makeRegistrationRequest(t *testing.T, baseURL string, data map[string]string) *http.Response {
	body, _ := json.Marshal(data)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	return resp
}

func TestFrontendStaticAssets(t *testing.T) {
	if !isServiceRunning(FrontendURL) {
		t.Skip("Frontend not running on", FrontendURL)
	}

	t.Run("Frontend Root Loads", func(t *testing.T) {
		resp, err := http.Get(FrontendURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "<!doctype html>")
		assert.Contains(t, bodyStr, "<div id=\"root\">")
	})
}

func TestPortConfiguration(t *testing.T) {
	t.Run("Expected Ports", func(t *testing.T) {
		// Test that services are running on expected ports
		services := map[string]string{
			"Frontend": "http://localhost:3000",
			"Backend":  "http://localhost:8080",
		}

		for name, url := range services {
			t.Run(name, func(t *testing.T) {
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(url)

				if err != nil {
					t.Logf("%s not running on %s: %v", name, url, err)
					t.Skipf("%s service not available", name)
				}

				defer resp.Body.Close()
				assert.True(t, resp.StatusCode < 500,
					"%s should be responding (got %d)", name, resp.StatusCode)
			})
		}
	})
}
