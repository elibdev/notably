package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/elibdev/notably/dynamo"
	"github.com/elibdev/notably/pkg/server"
	"github.com/elibdev/notably/testutil/dynamotest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTableName = "NotablyIntegrationTest"

func TestTableCreationAndListingIntegration(t *testing.T) {
	// Skip if DynamoDB emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// Set up environment for local DynamoDB
	oldTableName := os.Getenv("DYNAMODB_TABLE_NAME")
	oldEndpoint := os.Getenv("DYNAMODB_ENDPOINT_URL")

	os.Setenv("DYNAMODB_TABLE_NAME", testTableName)
	os.Setenv("DYNAMODB_ENDPOINT_URL", "http://localhost:8000")

	defer func() {
		if oldTableName == "" {
			os.Unsetenv("DYNAMODB_TABLE_NAME")
		} else {
			os.Setenv("DYNAMODB_TABLE_NAME", oldTableName)
		}
		if oldEndpoint == "" {
			os.Unsetenv("DYNAMODB_ENDPOINT_URL")
		} else {
			os.Setenv("DYNAMODB_ENDPOINT_URL", oldEndpoint)
		}
	}()

	// Create server with real configuration
	config := server.Config{
		TableName:      testTableName,
		Addr:           ":0", // Use any available port
		DynamoEndpoint: "http://localhost:8000",
	}

	srv, err := server.NewServer(config)
	require.NoError(t, err, "Failed to create server")

	// Create test server
	testServer := httptest.NewServer(srv.Handler())
	defer testServer.Close()

	// Create a test user and get API key
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@test.com", username)
	password := "testpassword123"

	// Register user
	registerReq := map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerReq)

	resp, err := http.Post(testServer.URL+"/auth/register", "application/json", bytes.NewBuffer(registerBody))
	require.NoError(t, err, "Failed to register user")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "User registration failed")

	var registerResponse struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		APIKey   string `json:"apiKey"`
	}
	err = json.NewDecoder(resp.Body).Decode(&registerResponse)
	resp.Body.Close()
	require.NoError(t, err, "Failed to decode register response")
	require.NotEmpty(t, registerResponse.APIKey, "API key should not be empty")

	apiKey := registerResponse.APIKey

	// Test 1: List tables initially (should be empty)
	req, err := http.NewRequest("GET", testServer.URL+"/tables", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err, "Failed to list tables")
	require.Equal(t, http.StatusOK, resp.StatusCode, "List tables should succeed")

	var listResponse struct {
		Tables []map[string]interface{} `json:"tables"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listResponse)
	resp.Body.Close()
	require.NoError(t, err, "Failed to decode list response")

	initialTableCount := len(listResponse.Tables)

	// Test 2: Create a table
	tableName := fmt.Sprintf("TestTable_%d", time.Now().UnixNano())
	createTableReq := map[string]interface{}{
		"name": tableName,
		"columns": []map[string]string{
			{"name": "id", "dataType": "string"},
			{"name": "name", "dataType": "string"},
			{"name": "age", "dataType": "number"},
			{"name": "active", "dataType": "boolean"},
		},
	}
	createTableBody, _ := json.Marshal(createTableReq)

	req, err = http.NewRequest("POST", testServer.URL+"/tables", bytes.NewBuffer(createTableBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err, "Failed to create table")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Table creation should succeed")

	var createResponse struct {
		Name      string                    `json:"name"`
		CreatedAt time.Time                 `json:"createdAt"`
		Columns   []dynamo.ColumnDefinition `json:"columns"`
	}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	resp.Body.Close()
	require.NoError(t, err, "Failed to decode create response")
	assert.Equal(t, tableName, createResponse.Name, "Created table name should match")
	assert.Len(t, createResponse.Columns, 4, "Should have 4 columns")

	// Wait a moment to ensure DynamoDB consistency
	time.Sleep(100 * time.Millisecond)

	// Test 3: List tables again (should now include the created table)
	req, err = http.NewRequest("GET", testServer.URL+"/tables", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err = client.Do(req)
	require.NoError(t, err, "Failed to list tables after creation")
	require.Equal(t, http.StatusOK, resp.StatusCode, "List tables should succeed")

	err = json.NewDecoder(resp.Body).Decode(&listResponse)
	resp.Body.Close()
	require.NoError(t, err, "Failed to decode list response")

	// Verify the table appears in the list
	assert.Len(t, listResponse.Tables, initialTableCount+1, "Should have one more table")

	// Find our created table
	var foundTable map[string]interface{}
	for _, table := range listResponse.Tables {
		if table["name"] == tableName {
			foundTable = table
			break
		}
	}
	require.NotNil(t, foundTable, "Created table should appear in list")
	assert.Equal(t, tableName, foundTable["name"], "Table name should match")

	// Verify columns are preserved
	columns, ok := foundTable["columns"].([]interface{})
	require.True(t, ok, "Columns should be present and be an array")
	assert.Len(t, columns, 4, "Should have 4 columns")

	// Test 4: Create another table to verify multiple tables work
	tableName2 := fmt.Sprintf("TestTable2_%d", time.Now().UnixNano())
	createTableReq2 := map[string]interface{}{
		"name": tableName2,
		"columns": []map[string]string{
			{"name": "title", "dataType": "string"},
			{"name": "completed", "dataType": "boolean"},
		},
	}
	createTableBody2, _ := json.Marshal(createTableReq2)

	req, err = http.NewRequest("POST", testServer.URL+"/tables", bytes.NewBuffer(createTableBody2))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err, "Failed to create second table")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Second table creation should succeed")
	resp.Body.Close()

	// Wait for consistency
	time.Sleep(100 * time.Millisecond)

	// Test 5: List tables and verify both tables are present
	req, err = http.NewRequest("GET", testServer.URL+"/tables", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err = client.Do(req)
	require.NoError(t, err, "Failed to list tables after second creation")
	require.Equal(t, http.StatusOK, resp.StatusCode, "List tables should succeed")

	err = json.NewDecoder(resp.Body).Decode(&listResponse)
	resp.Body.Close()
	require.NoError(t, err, "Failed to decode final list response")

	assert.Len(t, listResponse.Tables, initialTableCount+2, "Should have two more tables")

	// Verify both tables are present
	tableNames := make([]string, 0, len(listResponse.Tables))
	for _, table := range listResponse.Tables {
		if name, ok := table["name"].(string); ok {
			tableNames = append(tableNames, name)
		}
	}
	assert.Contains(t, tableNames, tableName, "First table should be in list")
	assert.Contains(t, tableNames, tableName2, "Second table should be in list")
}

func TestTableCreationValidation(t *testing.T) {
	// Skip if DynamoDB emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// Set up environment for local DynamoDB
	oldTableName := os.Getenv("DYNAMODB_TABLE_NAME")
	oldEndpoint := os.Getenv("DYNAMODB_ENDPOINT_URL")

	os.Setenv("DYNAMODB_TABLE_NAME", testTableName+"_validation")
	os.Setenv("DYNAMODB_ENDPOINT_URL", "http://localhost:8000")

	defer func() {
		if oldTableName == "" {
			os.Unsetenv("DYNAMODB_TABLE_NAME")
		} else {
			os.Setenv("DYNAMODB_TABLE_NAME", oldTableName)
		}
		if oldEndpoint == "" {
			os.Unsetenv("DYNAMODB_ENDPOINT_URL")
		} else {
			os.Setenv("DYNAMODB_ENDPOINT_URL", oldEndpoint)
		}
	}()

	// Create server
	config := server.Config{
		TableName:      testTableName + "_validation",
		Addr:           ":0",
		DynamoEndpoint: "http://localhost:8000",
	}

	srv, err := server.NewServer(config)
	require.NoError(t, err)

	testServer := httptest.NewServer(srv.Handler())
	defer testServer.Close()

	// Create test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@test.com", username)
	password := "testpassword123"

	registerReq := map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}
	registerBody, _ := json.Marshal(registerReq)

	resp, err := http.Post(testServer.URL+"/auth/register", "application/json", bytes.NewBuffer(registerBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var registerResponse struct {
		APIKey string `json:"apiKey"`
	}
	err = json.NewDecoder(resp.Body).Decode(&registerResponse)
	resp.Body.Close()
	require.NoError(t, err)

	apiKey := registerResponse.APIKey

	// Test invalid table name (empty)
	createTableReq := map[string]interface{}{
		"name": "",
	}
	createTableBody, _ := json.Marshal(createTableReq)

	req, err := http.NewRequest("POST", testServer.URL+"/tables", bytes.NewBuffer(createTableBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Empty table name should be rejected")
	resp.Body.Close()

	// Test invalid table name (special characters)
	createTableReq["name"] = "invalid@table#name"
	createTableBody, _ = json.Marshal(createTableReq)

	req, err = http.NewRequest("POST", testServer.URL+"/tables", bytes.NewBuffer(createTableBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Invalid table name should be rejected")
	resp.Body.Close()

	// Test invalid column definition (empty column name)
	createTableReq = map[string]interface{}{
		"name": "ValidTableName",
		"columns": []map[string]string{
			{"name": "", "dataType": "string"},
		},
	}
	createTableBody, _ = json.Marshal(createTableReq)

	req, err = http.NewRequest("POST", testServer.URL+"/tables", bytes.NewBuffer(createTableBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Empty column name should be rejected")
	resp.Body.Close()

	// Test invalid column definition (missing data type)
	createTableReq = map[string]interface{}{
		"name": "ValidTableName",
		"columns": []map[string]string{
			{"name": "validColumn", "dataType": ""},
		},
	}
	createTableBody, _ = json.Marshal(createTableReq)

	req, err = http.NewRequest("POST", testServer.URL+"/tables", bytes.NewBuffer(createTableBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Missing column data type should be rejected")
	resp.Body.Close()
}
