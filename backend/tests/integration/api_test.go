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
	"github.com/elibdev/notably/internal/models"
	"github.com/elibdev/notably/internal/repository"
)

func TestAPIIntegration(t *testing.T) {
	// Setup test server
	cfg := &apiConfig.Config{
		Auth: apiConfig.AuthConfig{
			JWTSecret:   "test-secret",
			TokenExpiry: time.Hour,
			RequireAuth: true, // Enable auth for testing
		},
	}

	userManager := &MockUserManager{}
	server := api.NewServer(cfg, userManager)
	// Initialize the server's router
	server.SetupRouter()
	server.SetupRoutes()
	testServer := httptest.NewServer(server.GetHandler())
	defer testServer.Close()

	// Test user registration
	t.Run("Register User", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"user_id":  "test_user",
			"password": "password123",
			"email":    "test@example.com",
		}

		resp := makeTestRequest(t, testServer.URL, "POST", "/api/v1/auth/register", reqBody, "")
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		defer resp.Body.Close()
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "token")
		assert.Contains(t, response, "user_id")
		assert.Equal(t, "test_user", response["user_id"])
	})

	// Test table creation
	t.Run("Create Table", func(t *testing.T) {
		// First register and get token
		token := authenticateUser(t, testServer.URL)

		reqBody := map[string]interface{}{
			"id": "contacts",
			"fields": []map[string]interface{}{
				{"name": "name", "data_type": "string"},
				{"name": "email", "data_type": "string"},
				{"name": "age", "data_type": "int"},
			},
		}

		resp := makeTestRequest(t, testServer.URL, "POST", "/api/v1/tables", reqBody, token)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]interface{}
		defer resp.Body.Close()
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "contacts", response["id"])
		assert.Contains(t, response, "fields")
	})

	// Test entity operations
	t.Run("Entity CRUD", func(t *testing.T) {
		token := authenticateUser(t, testServer.URL)

		// Create table first
		createTableTest(t, testServer.URL, token)

		// Create entity
		createReq := map[string]interface{}{
			"fields": map[string]interface{}{
				"name":  "John Doe",
				"email": "john.doe@example.com",
				"age":   30,
			},
		}

		resp := makeTestRequest(t, testServer.URL, "POST", "/api/v1/tables/contacts/entities", createReq, token)

		var entityResp map[string]interface{}
		defer resp.Body.Close()
		err := json.NewDecoder(resp.Body).Decode(&entityResp)
		require.NoError(t, err)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		entityID, exists := entityResp["entity_id"]
		require.True(t, exists, "entity_id not found in response: %+v", entityResp)
		entityIDStr := entityID.(string)
		assert.NotEmpty(t, entityID)

		// Get entity
		resp = makeTestRequest(t, testServer.URL, "GET", fmt.Sprintf("/api/v1/tables/contacts/entities/%s", entityIDStr), nil, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Update entity
		updateReqBody := map[string]interface{}{
			"fields": map[string]interface{}{
				"name": "Jane Doe",
				"age":  31,
			},
		}

		resp = makeTestRequest(t, testServer.URL, "PUT", fmt.Sprintf("/api/v1/tables/contacts/entities/%s", entityIDStr), updateReqBody, token)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Delete entity
		resp = makeTestRequest(t, testServer.URL, "DELETE", fmt.Sprintf("/api/v1/tables/contacts/entities/%s", entityIDStr), nil, token)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify deleted
		resp = makeTestRequest(t, testServer.URL, "GET", fmt.Sprintf("/api/v1/tables/contacts/entities/%s", entityIDStr), nil, token)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// MockUserManager implements repository.UserManager for testing
type MockUserManager struct {
	repositories map[string]repository.UserRepository
}

func (m *MockUserManager) GetUserRepository(userID string) repository.UserRepository {
	if m.repositories == nil {
		m.repositories = make(map[string]repository.UserRepository)
	}
	if repo, exists := m.repositories[userID]; exists {
		return repo
	}
	repo := &MockUserRepository{userID: userID}
	m.repositories[userID] = repo
	return repo
}

func (m *MockUserManager) ValidateUserAccess(ctx context.Context, userID string, tableID string) error {
	return nil
}

func (m *MockUserManager) CreateUser(ctx context.Context, userID string, email string, password string) (*repository.User, error) {
	now := time.Now()
	return &repository.User{
		UserID:     userID,
		Email:      email,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastActive: now,
	}, nil
}

func (m *MockUserManager) GetUser(ctx context.Context, userID string) (*repository.User, error) {
	now := time.Now()
	return &repository.User{
		UserID:     userID,
		Email:      "test@example.com",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastActive: now,
	}, nil
}

func (m *MockUserManager) DeleteUser(ctx context.Context, userID string) error {
	delete(m.repositories, userID)
	return nil
}

func (m *MockUserManager) GetUserByEmail(ctx context.Context, email string) (*repository.User, error) {
	now := time.Now()
	return &repository.User{
		UserID:     "mock_user",
		Email:      email,
		CreatedAt:  now,
		UpdatedAt:  now,
		LastActive: now,
	}, nil
}

func (m *MockUserManager) VerifyPassword(ctx context.Context, userID string, password string) (bool, error) {
	return true, nil
}

func (m *MockUserManager) UpdatePassword(ctx context.Context, userID string, newPassword string) error {
	return nil
}

func (m *MockUserManager) GetUserStats(ctx context.Context, userID string) (*repository.UserStats, error) {
	now := time.Now()
	return &repository.UserStats{
		UserID:      userID,
		Email:       "test@example.com",
		TableCount:  0,
		EntityCount: 0,
		LastActive:  now,
		CreatedAt:   now,
	}, nil
}

func (m *MockUserManager) Health(ctx context.Context) error {
	return nil
}

// MockUserRepository implements repository.UserRepository for testing
type MockUserRepository struct {
	userID   string
	tables   map[string]*models.TableSchema
	entities map[string]*models.EntitySnapshot
}

func (r *MockUserRepository) CreateTable(ctx context.Context, tableID string, fields []models.FieldDefinition) (*models.TableSchema, error) {
	if r.tables == nil {
		r.tables = make(map[string]*models.TableSchema)
	}
	table := models.NewTableSchema(r.userID, tableID, fields)
	r.tables[tableID] = &table
	return &table, nil
}

func (r *MockUserRepository) GetTable(ctx context.Context, tableID string) (*models.TableSchema, error) {
	if table, exists := r.tables[tableID]; exists {
		return table, nil
	}
	return nil, fmt.Errorf("table not found")
}

func (r *MockUserRepository) ListTables(ctx context.Context) ([]models.TableSchema, error) {
	var tables []models.TableSchema
	for _, table := range r.tables {
		tables = append(tables, *table)
	}
	return tables, nil
}

func (r *MockUserRepository) UpdateTable(ctx context.Context, table models.TableSchema) error {
	r.tables[table.ID] = &table
	return nil
}

func (r *MockUserRepository) DeleteTable(ctx context.Context, tableID string) error {
	delete(r.tables, tableID)
	return nil
}

func (r *MockUserRepository) GetTableHistory(ctx context.Context, tableID string, opts models.QueryOptions) (*models.TableHistory, error) {
	return &models.TableHistory{
		TableID: tableID,
		Changes: []models.Tuple{},
	}, nil
}

func (r *MockUserRepository) CreateEntity(ctx context.Context, tableID string, fields map[string]interface{}) (*models.EntitySnapshot, error) {
	if r.entities == nil {
		r.entities = make(map[string]*models.EntitySnapshot)
	}
	entityID := models.NewEntityID()
	now := time.Now()
	entity := &models.EntitySnapshot{
		EntityID:  entityID,
		TableID:   tableID,
		UserID:    r.userID,
		Fields:    fields,
		Timestamp: now,
		CreatedAt: &now,
	}
	r.entities[entityID] = entity
	return entity, nil
}

func (r *MockUserRepository) GetEntity(ctx context.Context, tableID string, entityID string, asOf *time.Time) (*models.EntitySnapshot, error) {
	if entity, exists := r.entities[entityID]; exists {
		if entity.IsDeleted {
			return nil, nil
		}
		return entity, nil
	}
	return nil, nil
}

func (r *MockUserRepository) GetAllEntities(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error) {
	entities := make(map[string]models.EntitySnapshot)
	for _, entity := range r.entities {
		if entity.TableID == tableID && !entity.IsDeleted {
			entities[entity.EntityID] = *entity
		}
	}
	now := time.Now()
	if asOf != nil {
		now = *asOf
	}
	return &models.EntitiesSnapshot{
		TableID:   tableID,
		Entities:  entities,
		Timestamp: now,
	}, nil
}

func (r *MockUserRepository) GetAllEntitiesIncludingDeleted(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error) {
	entities := make(map[string]models.EntitySnapshot)
	for _, entity := range r.entities {
		if entity.TableID == tableID {
			entities[entity.EntityID] = *entity
		}
	}
	now := time.Now()
	if asOf != nil {
		now = *asOf
	}
	return &models.EntitiesSnapshot{
		TableID:   tableID,
		Entities:  entities,
		Timestamp: now,
	}, nil
}

func (r *MockUserRepository) UpdateEntity(ctx context.Context, tableID string, entityID string, fields map[string]interface{}) (*models.EntitySnapshot, error) {
	if entity, exists := r.entities[entityID]; exists {
		for k, v := range fields {
			entity.SetFieldValue(k, v)
		}
		entity.Timestamp = time.Now()
		return entity, nil
	}
	return nil, fmt.Errorf("entity not found")
}

func (r *MockUserRepository) DeleteEntity(ctx context.Context, tableID string, entityID string) error {
	if entity, exists := r.entities[entityID]; exists {
		entity.IsDeleted = true
		now := time.Now()
		entity.DeletedAt = &now
		entity.Timestamp = now
		return nil
	}
	return fmt.Errorf("entity not found")
}

func (r *MockUserRepository) UndeleteEntity(ctx context.Context, tableID string, entityID string) error {
	if entity, exists := r.entities[entityID]; exists {
		entity.IsDeleted = false
		entity.DeletedAt = nil
		entity.Timestamp = time.Now()
		return nil
	}
	return fmt.Errorf("entity not found")
}

func (r *MockUserRepository) DeleteField(ctx context.Context, tableID string, entityID string, fieldName string) error {
	if entity, exists := r.entities[entityID]; exists {
		entity.DeleteField(fieldName)
		entity.Timestamp = time.Now()
		return nil
	}
	return fmt.Errorf("entity not found")
}

func (r *MockUserRepository) GetFieldHistory(ctx context.Context, tableID string, fieldName string, opts models.QueryOptions) (*models.FieldHistory, error) {
	return &models.FieldHistory{
		TableID:   tableID,
		FieldName: fieldName,
		Changes:   []models.FieldChange{},
	}, nil
}

func (r *MockUserRepository) GetEntityHistory(ctx context.Context, tableID string, entityID string, opts models.QueryOptions) (*models.QueryResult, error) {
	return &models.QueryResult{
		Tuples:  []models.Tuple{},
		HasMore: false,
	}, nil
}

func (r *MockUserRepository) HealthCheck(ctx context.Context) error {
	return nil
}

func makeTestRequest(t *testing.T, serverURL, method, path string, body interface{}, token string) *http.Response {
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

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

func authenticateUser(t *testing.T, serverURL string) string {
	reqBody := map[string]interface{}{
		"user_id":  "test_user",
		"password": "password123",
	}

	resp := makeTestRequest(t, serverURL, "POST", "/api/v1/auth/login", reqBody, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	return response["token"].(string)
}

func createTableTest(t *testing.T, serverURL string, token string) {
	reqBody := map[string]interface{}{
		"id": "contacts",
		"fields": []map[string]interface{}{
			{"name": "name", "data_type": "string"},
			{"name": "email", "data_type": "string"},
			{"name": "age", "data_type": "int"},
		},
	}

	resp := makeTestRequest(t, serverURL, "POST", "/api/v1/tables", reqBody, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func setupAWSConfig(cfg *apiConfig.Config) (aws.Config, error) {
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
