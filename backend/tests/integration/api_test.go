/*
package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elibdev/notably/internal/api"
	"github.com/elibdev/notably/internal/config"
	"github.com/elibdev/notably/internal/repository"
)

func TestAPIIntegration(t *testing.T) {
	// Setup test server
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:   "test-secret",
			TokenExpiry: time.Hour,
			RequireAuth: true,
		},
	}

	userManager := &repository.MockUserManager{}
	server := api.NewServer(cfg, userManager)
	router := server.SetupRouter() // We'd need to expose this method

	// Test user registration
	t.Run("Register User", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"user_id":  "test_user",
			"password": "password123",
			"email":    "test@example.com",
		}

		resp := makeTestRequest(t, router, "POST", "/api/v1/auth/register", reqBody, "")
		assert.Equal(t, http.StatusCreated, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "token")
		assert.Contains(t, response, "user_id")
		assert.Equal(t, "test_user", response["user_id"])
	})

	// Test table creation
	t.Run("Create Table", func(t *testing.T) {
		// First register and get token
		token := registerAndGetToken(t, router)

		reqBody := map[string]interface{}{
			"id": "contacts",
			"fields": []map[string]interface{}{
				{"name": "name", "data_type": "string"},
				{"name": "email", "data_type": "string"},
				{"name": "age", "data_type": "int"},
			},
		}

		resp := makeTestRequest(t, router, "POST", "/api/v1/tables", reqBody, token)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "contacts", response["id"])
		assert.Contains(t, response, "fields")
	})

	// Test entity operations
	t.Run("Entity CRUD", func(t *testing.T) {
		token := registerAndGetToken(t, router)

		// Create table first
		createTestTable(t, router, token)

		// Create entity
		createReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"age":   30,
			},
		}

		resp := makeTestRequest(t, router, "POST", "/api/v1/tables/contacts/entities", createReq, token)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var createResp map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &createResp)
		require.NoError(t, err)

		entityID := createResp["entity_id"].(string)
		assert.NotEmpty(t, entityID)

		// Get entity
		resp = makeTestRequest(t, router, "GET", "/api/v1/tables/contacts/entities/"+entityID, nil, token)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Update entity
		updateReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"email": "john.doe@newcompany.com",
				"age":   31,
			},
		}

		resp = makeTestRequest(t, router, "PUT", "/api/v1/tables/contacts/entities/"+entityID, updateReq, token)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Delete entity
		resp = makeTestRequest(t, router, "DELETE", "/api/v1/tables/contacts/entities/"+entityID, nil, token)
		assert.Equal(t, http.StatusNoContent, resp.Code)

		// Verify deleted
		resp = makeTestRequest(t, router, "GET", "/api/v1/tables/contacts/entities/"+entityID, nil, token)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func makeTestRequest(t *testing.T, router http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	return resp
}

func registerAndGetToken(t *testing.T, router http.Handler) string {
	reqBody := map[string]interface{}{
		"user_id":  "test_user",
		"password": "password123",
		"email":    "test@example.com",
	}

	resp := makeTestRequest(t, router, "POST", "/api/v1/auth/register", reqBody, "")
	require.Equal(t, http.StatusCreated, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	require.NoError(t, err)

	return response["token"].(string)
}

func createTestTable(t *testing.T, router http.Handler, token string) {
	reqBody := map[string]interface{}{
		"id": "contacts",
		"fields": []map[string]interface{}{
			{"name": "name", "data_type": "string"},
			{"name": "email", "data_type": "string"},
			{"name": "age", "data_type": "int"},
		},
	}

	resp := makeTestRequest(t, router, "POST", "/api/v1/tables", reqBody, token)
	require.Equal(t, http.StatusCreated, resp.Code)
}
*/

func setupAWSConfig(cfg *config.Config) (aws.Config, error) {
	ctx := context.Background()

	if cfg.Database.EndpointURL != "" {
		// Local DynamoDB
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{
					URL:               cfg.Database.EndpointURL,
					HostnameImmutable: true,
				}, nil
			}
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
		})

		return config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Database.Region),
			config.WithEndpointResolverWithOptions(customResolver),
		)
	}

	// Production AWS
	return config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Database.Region))
}
