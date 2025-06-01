package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elibdev/notably/internal/api"
	apiConfig "github.com/elibdev/notably/internal/config"
	"github.com/elibdev/notably/internal/repository"
)

// TestRealDynamoDBIntegration tests the actual DynamoDB implementation
func TestRealDynamoDBIntegration(t *testing.T) {
	// Skip if DynamoDB local is not available
	if !isDynamoDBLocalAvailable() {
		t.Skip("DynamoDB Local not available, skipping real integration tests")
	}

	// Setup real DynamoDB connection
	cfg := &apiConfig.Config{
		Database: apiConfig.DatabaseConfig{
			Provider:    "dynamodb",
			TableName:   "timedb_test",
			Region:      "us-east-1",
			EndpointURL: "http://localhost:8000",
		},
		Auth: apiConfig.AuthConfig{
			JWTSecret:   "test-secret-key-for-testing",
			TokenExpiry: time.Hour,
			RequireAuth: false, // Disable auth for simpler testing
		},
		Logging: apiConfig.LoggingConfig{
			Level:  "error", // Reduce noise in tests
			Format: "text",
		},
	}

	// Create real DynamoDB client
	awsCfg, err := setupRealAWSConfig(cfg)
	require.NoError(t, err)
	dynamoClient := dynamodb.NewFromConfig(awsCfg)

	// Create user manager with real DynamoDB
	userManager := repository.NewDynamoUserManager(dynamoClient, cfg.Database.TableName)

	// Create API server
	server := api.NewServer(cfg, userManager)
	server.SetupRouter()
	server.SetupRoutes()
	testServer := httptest.NewServer(server.GetHandler())
	defer testServer.Close()

	// Wait for table to be ready
	ctx := context.Background()
	err = userManager.Health(ctx)
	require.NoError(t, err, "DynamoDB table should be ready")

	// Run comprehensive tests
	testUserRegistration(t, testServer.URL)
	testTableOperations(t, testServer.URL)
	testEntityOperationsWithHistory(t, testServer.URL)
	testTimeTravelFunctionality(t, testServer.URL)
	testFieldOperations(t, testServer.URL)
	testSoftDeleteAndUndelete(t, testServer.URL)
	testFieldHistoryFunctionality(t, testServer.URL)
}

func testUserRegistration(t *testing.T, serverURL string) {
	t.Run("User Registration and Authentication", func(t *testing.T) {
		// Register a new user
		registerReq := map[string]interface{}{
			"user_id":  fmt.Sprintf("test_user_%d", time.Now().Unix()),
			"password": "secure_password_123",
			"email":    fmt.Sprintf("test_%d@example.com", time.Now().Unix()),
		}

		resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/auth/register", registerReq, "")
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var authResp map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&authResp)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Contains(t, authResp, "token")
		assert.Contains(t, authResp, "user_id")
		assert.Equal(t, registerReq["user_id"], authResp["user_id"])

		// Verify token is valid JWT
		token := authResp["token"].(string)
		assert.NotEmpty(t, token)
		assert.Contains(t, token, ".") // JWT should have dots
	})
}

func testTableOperations(t *testing.T, serverURL string) {
	t.Run("Table CRUD Operations", func(t *testing.T) {
		token := registerAndGetToken(t, serverURL)

		// Create table
		createTableReq := map[string]interface{}{
			"id": "test_contacts",
			"fields": []map[string]interface{}{
				{"name": "name", "data_type": "string"},
				{"name": "email", "data_type": "string"},
				{"name": "age", "data_type": "int"},
				{"name": "salary", "data_type": "float"},
				{"name": "active", "data_type": "bool"},
				{"name": "joined", "data_type": "date"},
				{"name": "metadata", "data_type": "json"},
			},
		}

		resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/tables", createTableReq, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var tableResp map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&tableResp)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, "test_contacts", tableResp["id"])
		assert.Contains(t, tableResp, "fields")

		// List tables
		resp = makeRealTestRequest(t, serverURL, "GET", "/api/v1/tables", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var listResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		resp.Body.Close()

		tables := listResp["tables"].([]interface{})
		assert.Len(t, tables, 1)

		// Get specific table
		resp = makeRealTestRequest(t, serverURL, "GET", "/api/v1/tables/test_contacts", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func testEntityOperationsWithHistory(t *testing.T, serverURL string) {
	t.Run("Entity Operations with History Tracking", func(t *testing.T) {
		token := registerAndGetToken(t, serverURL)
		createTestTable(t, serverURL, token)

		// Create entity
		createEntityReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"name":   "John Doe",
				"email":  "john@example.com",
				"age":    30,
				"salary": 75000.50,
				"active": true,
				"joined": time.Now().Format(time.RFC3339),
				"metadata": map[string]interface{}{
					"department": "engineering",
					"level":      "senior",
				},
			},
		}

		resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/tables/test_contacts/entities", createEntityReq, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var entityResp map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&entityResp)
		require.NoError(t, err)
		resp.Body.Close()

		entityID := entityResp["entity_id"].(string)
		assert.NotEmpty(t, entityID)

		// Verify initial entity state
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var getResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&getResp)
		require.NoError(t, err)
		resp.Body.Close()

		fields := getResp["fields"].(map[string]interface{})
		assert.Equal(t, "John Doe", fields["name"])
		assert.Equal(t, "john@example.com", fields["email"])
		assert.Equal(t, float64(30), fields["age"])

		// Update entity (this should create history)
		time.Sleep(100 * time.Millisecond) // Ensure timestamp difference
		updateReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"email":  "john.doe@newcompany.com",
				"salary": 85000.00,
				"metadata": map[string]interface{}{
					"department": "management",
					"level":      "director",
				},
			},
		}

		resp = makeRealTestRequest(t, serverURL, "PUT", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), updateReq, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify updated state
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&getResp)
		require.NoError(t, err)
		resp.Body.Close()

		fields = getResp["fields"].(map[string]interface{})
		assert.Equal(t, "john.doe@newcompany.com", fields["email"])
		assert.Equal(t, float64(85000), fields["salary"])

		// Check entity history (should have changes now)
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s/history", entityID), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var historyResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&historyResp)
		require.NoError(t, err)
		resp.Body.Close()

		tuples := historyResp["tuples"].([]interface{})
		assert.GreaterOrEqual(t, len(tuples), 2, "Should have at least 2 history entries (create + update)")

		// Verify history contains actual change data
		foundEmailChange := false
		foundSalaryChange := false
		for _, tupleInterface := range tuples {
			tuple := tupleInterface.(map[string]interface{})
			if fieldName, ok := tuple["field_name"]; ok {
				if fieldName == "email" {
					foundEmailChange = true
				}
				if fieldName == "salary" {
					foundSalaryChange = true
				}
			}
		}
		assert.True(t, foundEmailChange, "Should find email change in history")
		assert.True(t, foundSalaryChange, "Should find salary change in history")
	})
}

func testTimeTravelFunctionality(t *testing.T, serverURL string) {
	t.Run("Time Travel Functionality", func(t *testing.T) {
		token := registerAndGetToken(t, serverURL)
		createTestTable(t, serverURL, token)

		// Create entity
		createEntityReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"name":  "Alice Smith",
				"email": "alice@original.com",
				"age":   25,
			},
		}

		resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/tables/test_contacts/entities", createEntityReq, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var entityResp map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&entityResp)
		require.NoError(t, err)
		resp.Body.Close()

		entityID := entityResp["entity_id"].(string)

		// Record time before update for time travel
		beforeUpdate := time.Now()
		time.Sleep(1 * time.Second) // Ensure clear time separation

		// Update entity
		updateReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"email": "alice@updated.com",
				"age":   26,
			},
		}

		resp = makeRealTestRequest(t, serverURL, "PUT", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), updateReq, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Test individual entity time travel
		asOfParam := beforeUpdate.Format(time.RFC3339)
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s?asOf=%s", entityID, asOfParam), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var pastEntityResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&pastEntityResp)
		require.NoError(t, err)
		resp.Body.Close()

		pastFields := pastEntityResp["fields"].(map[string]interface{})
		assert.Equal(t, "alice@original.com", pastFields["email"], "Time travel should show original email")
		assert.Equal(t, float64(25), pastFields["age"], "Time travel should show original age")

		// Test entity listing time travel
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities?asOf=%s", asOfParam), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var pastEntitiesResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&pastEntitiesResp)
		require.NoError(t, err)
		resp.Body.Close()

		pastEntities := pastEntitiesResp["entities"].([]interface{})
		assert.Len(t, pastEntities, 1, "Should find entity in past state")

		pastEntity := pastEntities[0].(map[string]interface{})
		pastEntityFields := pastEntity["fields"].(map[string]interface{})
		assert.Equal(t, "alice@original.com", pastEntityFields["email"], "Past entity listing should show original email")
	})
}

func testFieldOperations(t *testing.T, serverURL string) {
	t.Run("Field Operations", func(t *testing.T) {
		token := registerAndGetToken(t, serverURL)
		createTestTable(t, serverURL, token)

		// Create entity
		createEntityReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"name":  "Bob Wilson",
				"email": "bob@example.com",
				"age":   35,
			},
		}

		resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/tables/test_contacts/entities", createEntityReq, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var entityResp map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&entityResp)
		require.NoError(t, err)
		resp.Body.Close()

		entityID := entityResp["entity_id"].(string)

		// Delete a field
		resp = makeRealTestRequest(t, serverURL, "DELETE", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s/fields/age", entityID), nil, token)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify field is deleted
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var getResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&getResp)
		require.NoError(t, err)
		resp.Body.Close()

		fields := getResp["fields"].(map[string]interface{})
		assert.NotContains(t, fields, "age", "Age field should be deleted")
		assert.Contains(t, fields, "name", "Name field should still exist")
		assert.Contains(t, fields, "email", "Email field should still exist")
	})
}

func testSoftDeleteAndUndelete(t *testing.T, serverURL string) {
	t.Run("Soft Delete and Undelete", func(t *testing.T) {
		token := registerAndGetToken(t, serverURL)
		createTestTable(t, serverURL, token)

		// Create entity
		createEntityReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"name":  "Charlie Brown",
				"email": "charlie@example.com",
			},
		}

		resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/tables/test_contacts/entities", createEntityReq, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var entityResp map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&entityResp)
		require.NoError(t, err)
		resp.Body.Close()

		entityID := entityResp["entity_id"].(string)

		// Soft delete entity
		resp = makeRealTestRequest(t, serverURL, "DELETE", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), nil, token)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify entity is not found in normal queries
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), nil, token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// Verify entity is not in normal entity listing
		resp = makeRealTestRequest(t, serverURL, "GET", "/api/v1/tables/test_contacts/entities", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var entitiesResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&entitiesResp)
		require.NoError(t, err)
		resp.Body.Close()

		entities := entitiesResp["entities"].([]interface{})
		for _, entityInterface := range entities {
			entity := entityInterface.(map[string]interface{})
			assert.NotEqual(t, entityID, entity["entity_id"], "Deleted entity should not appear in normal listing")
		}

		// Verify entity appears in admin view (including deleted)
		resp = makeRealTestRequest(t, serverURL, "GET", "/api/v1/admin/tables/test_contacts/entities", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Undelete entity
		resp = makeRealTestRequest(t, serverURL, "POST", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s/undelete", entityID), nil, token)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify entity is now accessible again
		resp = makeRealTestRequest(t, serverURL, "GET", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var undeleteResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&undeleteResp)
		require.NoError(t, err)
		resp.Body.Close()

		fields := undeleteResp["fields"].(map[string]interface{})
		assert.Equal(t, "Charlie Brown", fields["name"])
		assert.Equal(t, "charlie@example.com", fields["email"])
	})
}

func testFieldHistoryFunctionality(t *testing.T, serverURL string) {
	t.Run("Field History Functionality", func(t *testing.T) {
		token := registerAndGetToken(t, serverURL)
		createTestTable(t, serverURL, token)

		// Create entity with specific field
		createEntityReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"name":  "Dave Davis",
				"email": "dave@v1.com",
			},
		}

		resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/tables/test_contacts/entities", createEntityReq, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var entityResp map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&entityResp)
		require.NoError(t, err)
		resp.Body.Close()

		entityID := entityResp["entity_id"].(string)

		// Update the email field multiple times to create history
		time.Sleep(100 * time.Millisecond)
		updateReq1 := map[string]interface{}{
			"fields": map[string]interface{}{
				"email": "dave@v2.com",
			},
		}

		resp = makeRealTestRequest(t, serverURL, "PUT", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), updateReq1, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		time.Sleep(100 * time.Millisecond)
		updateReq2 := map[string]interface{}{
			"fields": map[string]interface{}{
				"email": "dave@v3.com",
			},
		}

		resp = makeRealTestRequest(t, serverURL, "PUT", fmt.Sprintf("/api/v1/tables/test_contacts/entities/%s", entityID), updateReq2, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Test field history endpoint
		resp = makeRealTestRequest(t, serverURL, "GET", "/api/v1/tables/test_contacts/history/fields/email", nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var fieldHistoryResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&fieldHistoryResp)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, "test_contacts", fieldHistoryResp["table_id"])
		assert.Equal(t, "email", fieldHistoryResp["field_name"])

		changes := fieldHistoryResp["changes"].([]interface{})
		assert.GreaterOrEqual(t, len(changes), 2, "Should have multiple email changes")

		// Verify changes contain the expected values
		foundV2 := false
		foundV3 := false
		for _, changeInterface := range changes {
			change := changeInterface.(map[string]interface{})
			if newValue, ok := change["new_value"]; ok {
				if newValue == "dave@v2.com" {
					foundV2 = true
				}
				if newValue == "dave@v3.com" {
					foundV3 = true
				}
			}
		}
		assert.True(t, foundV2, "Should find v2 email change")
		assert.True(t, foundV3, "Should find v3 email change")
	})
}

// Helper functions

func isDynamoDBLocalAvailable() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://localhost:8000")
	return err == nil
}

func setupRealAWSConfig(cfg *apiConfig.Config) (aws.Config, error) {
	ctx := context.Background()

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

func makeRealTestRequest(t *testing.T, serverURL, method, path string, body interface{}, token string) *http.Response {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, serverURL+path, bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func registerAndGetToken(t *testing.T, serverURL string) string {
	registerReq := map[string]interface{}{
		"user_id":  fmt.Sprintf("test_user_%d", time.Now().UnixNano()),
		"password": "secure_password_123",
		"email":    fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
	}

	resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/auth/register", registerReq, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var authResp map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&authResp)
	require.NoError(t, err)
	resp.Body.Close()

	return authResp["token"].(string)
}

func createTestTable(t *testing.T, serverURL string, token string) {
	createTableReq := map[string]interface{}{
		"id": "test_contacts",
		"fields": []map[string]interface{}{
			{"name": "name", "data_type": "string"},
			{"name": "email", "data_type": "string"},
			{"name": "age", "data_type": "int"},
			{"name": "salary", "data_type": "float"},
			{"name": "active", "data_type": "bool"},
			{"name": "joined", "data_type": "date"},
			{"name": "metadata", "data_type": "json"},
		},
	}

	resp := makeRealTestRequest(t, serverURL, "POST", "/api/v1/tables", createTableReq, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
}
