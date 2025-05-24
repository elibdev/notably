package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/elibdev/notably/dynamo"
	"github.com/elibdev/notably/testutil/dynamotest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableHandlers(t *testing.T) {
	// Skip if DynamoDB emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// Set up environment for local DynamoDB
	testTableName := fmt.Sprintf("TableHandlerTest_%d", time.Now().UnixNano())
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

	// Create server
	config := Config{
		TableName:      testTableName,
		Addr:           ":0",
		DynamoEndpoint: "http://localhost:8000",
	}

	srv, err := NewServer(config)
	require.NoError(t, err)

	// Create a test user
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@test.com", username)
	password := "testpassword123"

	user, err := srv.authenticator.RegisterUser(context.Background(), username, email, password)
	require.NoError(t, err)

	// Create API key for the user
	_, rawKey, err := srv.authenticator.GenerateAPIKey(context.Background(), user.ID, "test-key", 24*time.Hour)
	require.NoError(t, err)

	t.Run("CreateTable", func(t *testing.T) {
		// Test creating a table with columns
		tableName := fmt.Sprintf("TestTable_%d", time.Now().UnixNano())
		reqBody := map[string]interface{}{
			"name": tableName,
			"columns": []map[string]string{
				{"name": "id", "dataType": "string"},
				{"name": "title", "dataType": "string"},
				{"name": "completed", "dataType": "boolean"},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response struct {
			Name      string                    `json:"name"`
			CreatedAt time.Time                 `json:"createdAt"`
			Columns   []dynamo.ColumnDefinition `json:"columns"`
		}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, tableName, response.Name)
		assert.Len(t, response.Columns, 3)
		assert.Equal(t, "id", response.Columns[0].Name)
		assert.Equal(t, "string", response.Columns[0].DataType)
	})

	t.Run("CreateTableWithoutColumns", func(t *testing.T) {
		// Test creating a table without columns
		tableName := fmt.Sprintf("SimpleTable_%d", time.Now().UnixNano())
		reqBody := map[string]interface{}{
			"name": tableName,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response struct {
			Name      string                    `json:"name"`
			CreatedAt time.Time                 `json:"createdAt"`
			Columns   []dynamo.ColumnDefinition `json:"columns"`
		}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, tableName, response.Name)
		assert.Len(t, response.Columns, 0)
	})

	t.Run("ListTables", func(t *testing.T) {
		// Create a few tables first
		tableNames := []string{
			fmt.Sprintf("ListTest1_%d", time.Now().UnixNano()),
			fmt.Sprintf("ListTest2_%d", time.Now().UnixNano()),
		}

		for i, tableName := range tableNames {
			reqBody := map[string]interface{}{
				"name": tableName,
				"columns": []map[string]string{
					{"name": "field1", "dataType": "string"},
					{"name": "field2", "dataType": "number"},
				},
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+rawKey)

			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)
			require.Equal(t, http.StatusCreated, w.Code, "Failed to create table %d", i)
		}

		// Wait a moment for consistency
		time.Sleep(100 * time.Millisecond)

		// Now list tables
		req := httptest.NewRequest("GET", "/tables", nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Tables []TableInfo `json:"tables"`
		}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		// Should contain at least the two tables we just created
		assert.GreaterOrEqual(t, len(response.Tables), 2)

		// Find our created tables
		foundTables := make(map[string]bool)
		for _, table := range response.Tables {
			for _, expectedName := range tableNames {
				if table.Name == expectedName {
					foundTables[expectedName] = true
					// Verify columns are preserved
					assert.Len(t, table.Columns, 2)
					assert.Equal(t, "field1", table.Columns[0].Name)
					assert.Equal(t, "string", table.Columns[0].DataType)
				}
			}
		}

		// Verify both tables were found
		for _, tableName := range tableNames {
			assert.True(t, foundTables[tableName], "Table %s should be in the list", tableName)
		}
	})

	t.Run("CreateTableValidation", func(t *testing.T) {
		tests := []struct {
			name           string
			requestBody    map[string]interface{}
			expectedStatus int
			description    string
		}{
			{
				name:           "EmptyTableName",
				requestBody:    map[string]interface{}{"name": ""},
				expectedStatus: http.StatusBadRequest,
				description:    "Empty table name should be rejected",
			},
			{
				name:           "InvalidTableName",
				requestBody:    map[string]interface{}{"name": "invalid@name#with$special%chars"},
				expectedStatus: http.StatusBadRequest,
				description:    "Invalid table name should be rejected",
			},
			{
				name: "EmptyColumnName",
				requestBody: map[string]interface{}{
					"name": "ValidTableName",
					"columns": []map[string]string{
						{"name": "", "dataType": "string"},
					},
				},
				expectedStatus: http.StatusBadRequest,
				description:    "Empty column name should be rejected",
			},
			{
				name: "MissingColumnDataType",
				requestBody: map[string]interface{}{
					"name": "ValidTableName",
					"columns": []map[string]string{
						{"name": "validColumn", "dataType": ""},
					},
				},
				expectedStatus: http.StatusBadRequest,
				description:    "Missing column data type should be rejected",
			},
			{
				name: "InvalidColumnName",
				requestBody: map[string]interface{}{
					"name": "ValidTableName",
					"columns": []map[string]string{
						{"name": "invalid@column", "dataType": "string"},
					},
				},
				expectedStatus: http.StatusBadRequest,
				description:    "Invalid column name should be rejected",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+rawKey)

				w := httptest.NewRecorder()
				srv.Handler().ServeHTTP(w, req)

				assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
			})
		}
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		// Test creating table without auth
		reqBody := map[string]interface{}{
			"name": "TestTable",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Test listing tables without auth
		req = httptest.NewRequest("GET", "/tables", nil)

		w = httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestTableCreationFlow(t *testing.T) {
	// Skip if DynamoDB emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// This test specifically verifies the flow that would happen in the UI:
	// 1. User creates a table
	// 2. User lists tables
	// 3. The created table appears in the list

	testTableName := fmt.Sprintf("FlowTest_%d", time.Now().UnixNano())
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

	config := Config{
		TableName:      testTableName,
		Addr:           ":0",
		DynamoEndpoint: "http://localhost:8000",
	}

	srv, err := NewServer(config)
	require.NoError(t, err)

	// Create test user
	username := fmt.Sprintf("flowuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@test.com", username)
	password := "testpassword123"

	user, err := srv.authenticator.RegisterUser(context.Background(), username, email, password)
	require.NoError(t, err)

	_, rawKey, err := srv.authenticator.GenerateAPIKey(context.Background(), user.ID, "test-key", 24*time.Hour)
	require.NoError(t, err)

	// Step 1: List tables initially (should be empty or have known count)
	req := httptest.NewRequest("GET", "/tables", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var initialResponse struct {
		Tables []TableInfo `json:"tables"`
	}
	err = json.NewDecoder(w.Body).Decode(&initialResponse)
	require.NoError(t, err)

	initialCount := len(initialResponse.Tables)

	// Step 2: Create a new table
	newTableName := fmt.Sprintf("UICreatedTable_%d", time.Now().UnixNano())
	createReq := map[string]interface{}{
		"name": newTableName,
		"columns": []map[string]string{
			{"name": "id", "dataType": "string"},
			{"name": "title", "dataType": "string"},
			{"name": "priority", "dataType": "number"},
			{"name": "completed", "dataType": "boolean"},
		},
	}

	body, _ := json.Marshal(createReq)
	req = httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawKey)

	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse struct {
		Name      string                    `json:"name"`
		CreatedAt time.Time                 `json:"createdAt"`
		Columns   []dynamo.ColumnDefinition `json:"columns"`
	}
	err = json.NewDecoder(w.Body).Decode(&createResponse)
	require.NoError(t, err)

	assert.Equal(t, newTableName, createResponse.Name)
	assert.Len(t, createResponse.Columns, 4)

	// Step 3: List tables again and verify the new table appears
	// Add a small delay to account for eventual consistency
	time.Sleep(100 * time.Millisecond)

	req = httptest.NewRequest("GET", "/tables", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var finalResponse struct {
		Tables []TableInfo `json:"tables"`
	}
	err = json.NewDecoder(w.Body).Decode(&finalResponse)
	require.NoError(t, err)

	// Verify we have one more table
	assert.Equal(t, initialCount+1, len(finalResponse.Tables), "Should have one more table after creation")

	// Verify the created table is in the list
	found := false
	var foundTable TableInfo
	for _, table := range finalResponse.Tables {
		if table.Name == newTableName {
			found = true
			foundTable = table
			break
		}
	}

	require.True(t, found, "Created table should appear in the list")
	assert.Equal(t, newTableName, foundTable.Name)
	assert.Len(t, foundTable.Columns, 4)

	// Verify column details are preserved
	expectedColumns := map[string]string{
		"id":        "string",
		"title":     "string",
		"priority":  "number",
		"completed": "boolean",
	}

	for _, col := range foundTable.Columns {
		expectedType, exists := expectedColumns[col.Name]
		assert.True(t, exists, "Column %s should be expected", col.Name)
		assert.Equal(t, expectedType, col.DataType, "Column %s should have correct type", col.Name)
	}
}

func TestRowManagement(t *testing.T) {
	// Skip if DynamoDB emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// Set up environment for local DynamoDB
	testTableName := fmt.Sprintf("RowTest_%d", time.Now().UnixNano())
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

	config := Config{
		TableName:      testTableName,
		Addr:           ":0",
		DynamoEndpoint: "http://localhost:8000",
	}

	srv, err := NewServer(config)
	require.NoError(t, err)

	// Create test user
	username := fmt.Sprintf("rowuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@test.com", username)
	password := "testpassword123"

	user, err := srv.authenticator.RegisterUser(context.Background(), username, email, password)
	require.NoError(t, err)

	_, rawKey, err := srv.authenticator.GenerateAPIKey(context.Background(), user.ID, "test-key", 24*time.Hour)
	require.NoError(t, err)

	// First create a table
	tableName := fmt.Sprintf("TestTable_%d", time.Now().UnixNano())
	createTableReq := map[string]interface{}{
		"name": tableName,
		"columns": []map[string]string{
			{"name": "id", "dataType": "string"},
			{"name": "title", "dataType": "string"},
			{"name": "priority", "dataType": "number"},
			{"name": "completed", "dataType": "boolean"},
		},
	}

	body, _ := json.Marshal(createTableReq)
	req := httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rawKey)

	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	t.Run("CreateRow", func(t *testing.T) {
		// Test creating a row
		rowData := map[string]interface{}{
			"values": map[string]interface{}{
				"id":        "task-1",
				"title":     "Complete project",
				"priority":  1,
				"completed": false,
			},
		}

		body, _ := json.Marshal(rowData)
		req := httptest.NewRequest("POST", fmt.Sprintf("/tables/%s/rows", tableName), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response RowData
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("Created row response: ID=%s, Values=%+v", response.ID, response.Values)
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, "task-1", response.Values["id"])
		assert.Equal(t, "Complete project", response.Values["title"])
		assert.Equal(t, float64(1), response.Values["priority"]) // JSON numbers are float64
		assert.Equal(t, false, response.Values["completed"])
	})

	t.Run("CreateRowWithAutoGeneratedID", func(t *testing.T) {
		// Test creating a row without specifying an ID
		rowData := map[string]interface{}{
			"values": map[string]interface{}{
				"id":        "task-2",
				"title":     "Review code",
				"priority":  2,
				"completed": true,
			},
		}

		body, _ := json.Marshal(rowData)
		req := httptest.NewRequest("POST", fmt.Sprintf("/tables/%s/rows", tableName), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response RowData
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("Created second row response: ID=%s, Values=%+v", response.ID, response.Values)
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, "task-2", response.Values["id"])
		assert.Equal(t, "Review code", response.Values["title"])
	})

	t.Run("ListRows", func(t *testing.T) {
		// Wait a moment for consistency
		time.Sleep(100 * time.Millisecond)

		// Test listing rows
		req := httptest.NewRequest("GET", fmt.Sprintf("/tables/%s/rows", tableName), nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Rows []RowData `json:"rows"`
		}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("ListRows response: %d rows found", len(response.Rows))
		for i, row := range response.Rows {
			t.Logf("Row %d: ID=%s, Values=%+v", i, row.ID, row.Values)
		}

		// Should have at least 2 rows
		assert.GreaterOrEqual(t, len(response.Rows), 2)

		// Verify row data
		foundRows := make(map[string]RowData)
		for _, row := range response.Rows {
			if idVal, ok := row.Values["id"].(string); ok {
				foundRows[idVal] = row
			}
		}

		// Check first row
		if row, exists := foundRows["task-1"]; exists {
			assert.Equal(t, "Complete project", row.Values["title"])
			assert.Equal(t, float64(1), row.Values["priority"])
			assert.Equal(t, false, row.Values["completed"])
		} else {
			t.Error("Expected to find row with id 'task-1'")
		}

		// Check second row
		if row, exists := foundRows["task-2"]; exists {
			assert.Equal(t, "Review code", row.Values["title"])
			assert.Equal(t, float64(2), row.Values["priority"])
			assert.Equal(t, true, row.Values["completed"])
		} else {
			t.Error("Expected to find row with id 'task-2'")
		}
	})

	t.Run("TableSnapshot", func(t *testing.T) {
		// Test getting table snapshot
		req := httptest.NewRequest("GET", fmt.Sprintf("/tables/%s/snapshot", tableName), nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Rows []RowData `json:"rows"`
		}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("TableSnapshot response: %d rows found", len(response.Rows))
		for i, row := range response.Rows {
			t.Logf("Snapshot Row %d: ID=%s, Values=%+v", i, row.ID, row.Values)
		}

		// Should have the same rows as ListRows
		assert.GreaterOrEqual(t, len(response.Rows), 2)
	})

	t.Run("TableHistory", func(t *testing.T) {
		// Test getting table history
		start := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
		end := time.Now().UTC().Format(time.RFC3339)

		req := httptest.NewRequest("GET", fmt.Sprintf("/tables/%s/history?start=%s&end=%s", tableName, start, end), nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Events []RowEvent `json:"events"`
		}
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		t.Logf("TableHistory response: %d events found", len(response.Events))
		for i, event := range response.Events {
			t.Logf("Event %d: ID=%s, Values=%+v", i, event.ID, event.Values)
		}

		// Should have at least 2 events (the row creations)
		assert.GreaterOrEqual(t, len(response.Events), 2)

		// Verify events contain the created rows
		foundEvents := make(map[string]RowEvent)
		for _, event := range response.Events {
			if idVal, ok := event.Values["id"].(string); ok {
				foundEvents[idVal] = event
			}
		}

		assert.Contains(t, foundEvents, "task-1")
		assert.Contains(t, foundEvents, "task-2")
	})

	t.Run("CreateRowValidation", func(t *testing.T) {
		tests := []struct {
			name           string
			requestBody    map[string]interface{}
			expectedStatus int
			description    string
		}{
			{
				name:           "MissingValues",
				requestBody:    map[string]interface{}{},
				expectedStatus: http.StatusBadRequest,
				description:    "Missing values should be rejected",
			},
			{
				name: "InvalidColumnType",
				requestBody: map[string]interface{}{
					"values": map[string]interface{}{
						"id":       "task-3",
						"title":    "Test task",
						"priority": "high", // Should be number
					},
				},
				expectedStatus: http.StatusBadRequest,
				description:    "Invalid column type should be rejected",
			},
			{
				name: "UndefinedColumn",
				requestBody: map[string]interface{}{
					"values": map[string]interface{}{
						"id":            "task-4",
						"title":         "Test task",
						"priority":      1,
						"undefined_col": "value",
					},
				},
				expectedStatus: http.StatusBadRequest,
				description:    "Undefined column should be rejected",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest("POST", fmt.Sprintf("/tables/%s/rows", tableName), bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+rawKey)

				w := httptest.NewRecorder()
				srv.Handler().ServeHTTP(w, req)

				assert.Equal(t, tt.expectedStatus, w.Code, tt.description)
			})
		}
	})

	t.Run("RowOperationsOnNonexistentTable", func(t *testing.T) {
		nonexistentTable := "nonexistent_table"

		// Test creating row on nonexistent table
		rowData := map[string]interface{}{
			"values": map[string]interface{}{
				"id": "test",
			},
		}

		body, _ := json.Marshal(rowData)
		req := httptest.NewRequest("POST", fmt.Sprintf("/tables/%s/rows", nonexistentTable), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Test listing rows on nonexistent table
		req = httptest.NewRequest("GET", fmt.Sprintf("/tables/%s/rows", nonexistentTable), nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w = httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Test snapshot on nonexistent table
		req = httptest.NewRequest("GET", fmt.Sprintf("/tables/%s/snapshot", nonexistentTable), nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)

		w = httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
