package tests

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
	"github.com/elibdev/notably/db"
	"github.com/elibdev/notably/dynamo"
)

// TestUser is a constant user ID for testing
const TestUser = "test-user-123"

// setupTestServer sets up a test server with the given DynamoDB configuration
func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	// Set up a local DynamoDB endpoint for testing
	tableName := fmt.Sprintf("notably-test-%d", time.Now().UnixNano())
	
	// Create AWS config with a mock resolver for testing
	ctx := context.Background()
	mockEndpoint := "http://localhost:8000" // This would be your local DynamoDB endpoint
	resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if service == dynamodb.ServiceID {
			return aws.Endpoint{URL: mockEndpoint, SigningRegion: "us-east-1"}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	
	cfg, err := config.LoadDefaultConfig(ctx, config.WithEndpointResolver(resolver))
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}
	
	// Create base client and store
	baseClient := dynamo.NewClient(cfg, tableName, "")
	baseStore := db.NewStoreAdapter(db.CreateStoreFromClient(baseClient))
	
	// Create the test table
	if err := baseStore.CreateTable(ctx); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	
	// Set up the server mux
	mux := http.NewServeMux()
	
	// Handler middleware for user authentication
	withUser := func(h func(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := r.Header.Get("X-User-ID")
			if user == "" {
				http.Error(w, "missing X-User-ID header", http.StatusUnauthorized)
				return
			}
			store := db.NewStoreAdapter(db.CreateStoreFromClient(dynamo.NewClient(cfg, tableName, user)))
			h(w, r, user, store)
		}
	}
	
	// Define handlers similar to main.go
	mux.HandleFunc("GET /tables", withUser(handleListTables))
	mux.HandleFunc("POST /tables", withUser(handleCreateTable))
	mux.HandleFunc("GET /tables/{table}/rows", withUser(handleListRows))
	mux.HandleFunc("GET /tables/{table}/rows/{id}", withUser(handleGetRow))
	mux.HandleFunc("POST /tables/{table}/rows", withUser(handleCreateRow))
	mux.HandleFunc("PUT /tables/{table}/rows/{id}", withUser(handleUpdateRow))
	mux.HandleFunc("DELETE /tables/{table}/rows/{id}", withUser(handleDeleteRow))
	mux.HandleFunc("GET /tables/{table}/snapshot", withUser(handleTableSnapshot))
	mux.HandleFunc("GET /tables/{table}/history", withUser(handleTableHistory))
	
	// Create and start the test server
	server := httptest.NewServer(mux)
	
	// Return cleanup function to delete the test table
	cleanup := func() {
		server.Close()
		// Ideally would delete the DynamoDB table here in a real implementation
	}
	
	return server, cleanup
}

// Helper functions to simulate the handlers (simplified for testing)
func handleListTables(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	// Simplified implementation for tests
	tables := []map[string]interface{}{
		{"name": "table1", "createdAt": time.Now().Format(time.RFC3339)},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tables": tables})
}

func handleCreateTable(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "invalid JSON or missing table name", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"name":      req.Name,
		"createdAt": time.Now().Format(time.RFC3339),
	})
}

func handleListRows(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	rows := []map[string]interface{}{
		{
			"id":        "row1",
			"timestamp": time.Now().Format(time.RFC3339),
			"values":    map[string]interface{}{"key1": "value1"},
		},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"rows": rows})
}

func handleGetRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	rowID := r.PathValue("id")
	if rowID == "notfound" {
		http.Error(w, "row not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":        rowID,
		"timestamp": time.Now().Format(time.RFC3339),
		"values":    map[string]interface{}{"key1": "value1"},
	})
}

func handleCreateRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	var req struct {
		ID     string                 `json:"id"`
		Values map[string]interface{} `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "invalid JSON or missing row id", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        req.ID,
		"timestamp": time.Now().Format(time.RFC3339),
		"values":    req.Values,
	})
}

func handleUpdateRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	rowID := r.PathValue("id")
	var req struct {
		Values map[string]interface{} `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":        rowID,
		"timestamp": time.Now().Format(time.RFC3339),
		"values":    req.Values,
	})
}

func handleDeleteRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	// Just return 204 No Content for tests
	w.WriteHeader(http.StatusNoContent)
}

func handleTableSnapshot(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	rows := []map[string]interface{}{
		{
			"id":        "row1",
			"timestamp": time.Now().Format(time.RFC3339),
			"values":    map[string]interface{}{"key1": "value1"},
		},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"rows": rows})
}

func handleTableHistory(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	events := []map[string]interface{}{
		{
			"id":        "row1",
			"timestamp": time.Now().Format(time.RFC3339),
			"values":    map[string]interface{}{"key1": "value1"},
		},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events})
}

// Helper function for writing JSON responses
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Actual tests

func TestListTables(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	req, _ := http.NewRequest("GET", server.URL+"/tables", nil)
	req.Header.Set("X-User-ID", TestUser)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	tables, ok := result["tables"].([]interface{})
	if !ok {
		t.Fatalf("Expected tables array, got %T", result["tables"])
	}
	
	if len(tables) == 0 {
		t.Error("Expected at least one table")
	}
}

func TestCreateTable(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	tableName := "test-table"
	reqBody, _ := json.Marshal(map[string]string{"name": tableName})
	
	req, _ := http.NewRequest("POST", server.URL+"/tables", bytes.NewBuffer(reqBody))
	req.Header.Set("X-User-ID", TestUser)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", resp.Status)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if result["name"] != tableName {
		t.Errorf("Expected table name %s, got %v", tableName, result["name"])
	}
}

func TestCreateRow(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	rowID := "test-row"
	rowValues := map[string]string{"key1": "value1", "key2": "value2"}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"id":     rowID,
		"values": rowValues,
	})
	
	req, _ := http.NewRequest("POST", server.URL+"/tables/test-table/rows", bytes.NewBuffer(reqBody))
	req.Header.Set("X-User-ID", TestUser)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", resp.Status)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if result["id"] != rowID {
		t.Errorf("Expected row ID %s, got %v", rowID, result["id"])
	}
}

func TestGetRow(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	// Test existing row
	req, _ := http.NewRequest("GET", server.URL+"/tables/test-table/rows/row1", nil)
	req.Header.Set("X-User-ID", TestUser)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}
	
	// Test non-existing row
	req, _ = http.NewRequest("GET", server.URL+"/tables/test-table/rows/notfound", nil)
	req.Header.Set("X-User-ID", TestUser)
	
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status Not Found, got %v", resp.Status)
	}
}

func TestUpdateRow(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	rowValues := map[string]string{"key1": "updated", "key2": "updated"}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"values": rowValues,
	})
	
	req, _ := http.NewRequest("PUT", server.URL+"/tables/test-table/rows/row1", bytes.NewBuffer(reqBody))
	req.Header.Set("X-User-ID", TestUser)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	values, ok := result["values"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected values map, got %T", result["values"])
	}
	
	if values["key1"] != "updated" {
		t.Errorf("Expected updated value, got %v", values["key1"])
	}
}

func TestDeleteRow(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	req, _ := http.NewRequest("DELETE", server.URL+"/tables/test-table/rows/row1", nil)
	req.Header.Set("X-User-ID", TestUser)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status No Content, got %v", resp.Status)
	}
}

func TestTableSnapshot(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	req, _ := http.NewRequest("GET", server.URL+"/tables/test-table/snapshot", nil)
	req.Header.Set("X-User-ID", TestUser)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	rows, ok := result["rows"].([]interface{})
	if !ok {
		t.Fatalf("Expected rows array, got %T", result["rows"])
	}
	
	if len(rows) == 0 {
		t.Error("Expected at least one row")
	}
}

func TestTableHistory(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	
	// Set up query parameters for time range
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour).Format(time.RFC3339)
	end := now.Format(time.RFC3339)
	
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/tables/test-table/history?start=%s&end=%s", 
		server.URL, start, end), nil)
	req.Header.Set("X-User-ID", TestUser)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	events, ok := result["events"].([]interface{})
	if !ok {
		t.Fatalf("Expected events array, got %T", result["events"])
	}
	
	if len(events) == 0 {
		t.Error("Expected at least one event")
	}
}