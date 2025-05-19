package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/elibdev/notably/db"
	"github.com/elibdev/notably/dynamo"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":8080", "HTTP listen address")
	flag.Parse()

	// Underlying DynamoDB table name must be set via environment
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		log.Fatal("environment variable DYNAMODB_TABLE_NAME is required")
	}

	ctx := context.Background()
	opts := []func(*config.LoadOptions) error{}
	if ep := os.Getenv("DYNAMODB_ENDPOINT_URL"); ep != "" {
		resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{URL: ep, SigningRegion: region}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		opts = append(opts, config.WithEndpointResolver(resolver))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Fatalf("loading AWS config: %v", err)
	}

	// Ensure the underlying DynamoDB table exists (no user context)
	baseClient := dynamo.NewClient(cfg, tableName, "")
	baseStore := db.NewStoreAdapter(db.CreateStoreFromClient(baseClient))
	if err := baseStore.CreateTable(ctx); err != nil {
		log.Fatalf("creating table: %v", err)
	}

	// Per-request factory for a StoreAdapter bound to the current user
	getStore := func(user string) *db.StoreAdapter {
		client := dynamo.NewClient(cfg, tableName, user)
		return db.NewStoreAdapter(db.CreateStoreFromClient(client))
	}

	mux := http.NewServeMux()

	// Handler middleware for user authentication
	withUser := func(h func(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, err := requireUser(r)
			if err != nil {
				writeError(w, http.StatusUnauthorized, err.Error())
				return
			}
			
			// Add context values that will be useful to handlers
			ctx := context.WithValue(r.Context(), "user", user)
			r = r.WithContext(ctx)
			
			store := getStore(user)
			h(w, r, user, store)
		}
	}

	// CRUD for user tables
	mux.HandleFunc("GET /tables", withUser(handleListTables))
	mux.HandleFunc("POST /tables", withUser(handleCreateTable))

	// Operations on specific rows
	mux.HandleFunc("GET /tables/{table}/rows", withUser(handleListRows))
	mux.HandleFunc("GET /tables/{table}/rows/{id}", withUser(handleGetRow))
	mux.HandleFunc("POST /tables/{table}/rows", withUser(handleCreateRow))
	mux.HandleFunc("PUT /tables/{table}/rows/{id}", withUser(handleUpdateRow))
	mux.HandleFunc("DELETE /tables/{table}/rows/{id}", withUser(handleDeleteRow))

	// Table snapshot and history
	mux.HandleFunc("GET /tables/{table}/snapshot", withUser(handleTableSnapshot))
	mux.HandleFunc("GET /tables/{table}/history", withUser(handleTableHistory))

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// requireUser extracts the X-User-ID header or returns an error
func requireUser(r *http.Request) (string, error) {
	user := r.Header.Get("X-User-ID")
	if user == "" {
		return "", fmt.Errorf("missing X-User-ID header")
	}
	return user, nil
}

// newID generates a simple unique ID based on the current time
func newID() string {
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

// writeError writes an error message as JSON.
func writeError(w http.ResponseWriter, status int, msg string) {
	log.Printf("API error (%d): %s", status, msg)
	writeJSON(w, status, map[string]string{"error": msg})
}

// isValidName checks if a name contains only allowed characters
func isValidName(name string) bool {
	for _, r := range name {
		if !(('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// tableExists checks if a table exists for the given user
func tableExists(ctx context.Context, store *db.StoreAdapter, user, table string) bool {
	snap, err := store.GetSnapshot(ctx, time.Now().UTC())
	if err != nil {
		return false
	}
	
	if entries, ok := snap[user]; ok {
		_, exists := entries[table]
		return exists
	}
	return false
}

// parseTime parses an RFC3339 timestamp.
func parseTime(val string) (time.Time, error) {
	return time.Parse(time.RFC3339, val)
}

// TableInfo represents metadata for a user table.
type TableInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// RowData represents a row snapshot for a table.
type RowData struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"`
}

// RowEvent represents a history event for a row.
type RowEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"`
}

// handleCreateTable records a new table under the user's namespace.
func handleCreateTable(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	var req struct {
		Name string `json:"name"`
	}
	
	// Check for valid JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	
	// Validate table name
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "table name is required")
		return
	}
	
	// Validate table name format (alphanumeric and hyphens only)
	if !isValidName(req.Name) {
		writeError(w, http.StatusBadRequest, "table name must contain only alphanumeric characters and hyphens")
		return
	}
	
	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: user,
		FieldName: req.Name,
		DataType:  "string",
		Value:     "",
	}
	
	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create table: %v", err))
		return
	}
	
	writeJSON(w, http.StatusCreated, TableInfo{Name: req.Name, CreatedAt: fact.Timestamp})
}

// handleListTables returns the list of tables for the user.
func handleListTables(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tbls := []TableInfo{}
	if entries, ok := snap[user]; ok {
		for name, fact := range entries {
			tbls = append(tbls, TableInfo{Name: name, CreatedAt: fact.Timestamp})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tables": tbls})
}

// handleCreateRow creates a new row in a table
func handleCreateRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	
	// Validate table exists first
	if !tableExists(r.Context(), store, user, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("table '%s' not found", table))
		return
	}
	
	var req struct {
		ID     string                 `json:"id"`
		Values map[string]interface{} `json:"values"`
	}
	
	// Check for valid JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	
	// Validate row ID
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "row id is required")
		return
	}
	
	// Validate that values is not nil
	if req.Values == nil {
		writeError(w, http.StatusBadRequest, "row values are required")
		return
	}
	
	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: fmt.Sprintf("%s/%s", user, table),
		FieldName: req.ID,
		DataType:  "json",
		Value:     req.Values,
	}
	
	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create row: %v", err))
		return
	}
	
	writeJSON(w, http.StatusCreated, RowData{ID: req.ID, Timestamp: fact.Timestamp, Values: req.Values})
}

// handleListRows lists all rows in a table
func handleListRows(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	key := fmt.Sprintf("%s/%s", user, table)
	rows := []RowData{}
	
	if entries, ok := snap[key]; ok {
		for id, fact := range entries {
			if fact.DataType == "json" {
				vals, _ := fact.Value.(map[string]interface{})
				rows = append(rows, RowData{ID: id, Timestamp: fact.Timestamp, Values: vals})
			}
		}
	}
	
	writeJSON(w, http.StatusOK, map[string]interface{}{"rows": rows})
}

// handleGetRow gets a single row by ID
func handleGetRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	rowID := r.PathValue("id")
	
	// Validate table exists
	if !tableExists(r.Context(), store, user, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("table '%s' not found", table))
		return
	}
	
	// Validate row ID is not empty
	if rowID == "" {
		writeError(w, http.StatusBadRequest, "row id is required")
		return
	}
	
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get snapshot: %v", err))
		return
	}
	
	key := fmt.Sprintf("%s/%s", user, table)
	if entries, ok := snap[key]; ok {
		if fact, ok := entries[rowID]; ok && fact.DataType == "json" {
			vals, ok := fact.Value.(map[string]interface{})
			if !ok {
				writeError(w, http.StatusInternalServerError, "invalid row data format")
				return
			}
			writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: fact.Timestamp, Values: vals})
			return
		}
	}
	
	writeError(w, http.StatusNotFound, fmt.Sprintf("row '%s' not found in table '%s'", rowID, table))
}

// handleUpdateRow updates an existing row
func handleUpdateRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	rowID := r.PathValue("id")
	
	// Validate table exists
	if !tableExists(r.Context(), store, user, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("table '%s' not found", table))
		return
	}
	
	// Validate the row exists
	key := fmt.Sprintf("%s/%s", user, table)
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get snapshot: %v", err))
		return
	}
	
	// Check if row exists
	rowExists := false
	if entries, ok := snap[key]; ok {
		_, rowExists = entries[rowID]
	}
	
	if !rowExists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("row '%s' not found in table '%s'", rowID, table))
		return
	}
	
	var req struct {
		Values map[string]interface{} `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	
	if req.Values == nil {
		writeError(w, http.StatusBadRequest, "row values are required")
		return
	}
	
	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: fmt.Sprintf("%s/%s", user, table),
		FieldName: rowID,
		DataType:  "json",
		Value:     req.Values,
	}
	
	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update row: %v", err))
		return
	}
	
	writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: fact.Timestamp, Values: req.Values})
}

// handleDeleteRow deletes a row (tombstone)
func handleDeleteRow(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	rowID := r.PathValue("id")
	
	// Validate table exists
	if !tableExists(r.Context(), store, user, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("table '%s' not found", table))
		return
	}
	
	// Check if row exists
	key := fmt.Sprintf("%s/%s", user, table)
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get snapshot: %v", err))
		return
	}
	
	rowExists := false
	if entries, ok := snap[key]; ok {
		_, rowExists = entries[rowID]
	}
	
	// Even if row doesn't exist, we'll create a tombstone for it,
	// but we'll log this unusual situation
	if !rowExists {
		log.Printf("Warning: creating tombstone for non-existent row '%s' in table '%s'", rowID, table)
	}
	
	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: fmt.Sprintf("%s/%s", user, table),
		FieldName: rowID,
		DataType:  "json",
		Value:     nil,
	}
	
	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete row: %v", err))
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// handleTableSnapshot returns the state of a table at a given time.
func handleTableSnapshot(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	
	// Validate table exists
	if !tableExists(r.Context(), store, user, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("table '%s' not found", table))
		return
	}
	
	q := r.URL.Query()
	atParam := q.Get("at")
	var at time.Time
	var err error
	if atParam == "" {
		at = time.Now().UTC()
	} else {
		at, err = parseTime(atParam)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid 'at' time format: %v (expected RFC3339)", err))
			return
		}
	}
	
	snap, err := store.GetSnapshot(r.Context(), at)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get snapshot: %v", err))
		return
	}
	
	key := fmt.Sprintf("%s/%s", user, table)
	rows := []RowData{}
	if entries, ok := snap[key]; ok {
		for id, fact := range entries {
			if fact.DataType == "json" {
				vals, ok := fact.Value.(map[string]interface{})
				if !ok {
					log.Printf("Warning: invalid data format for row '%s' in snapshot", id)
					continue
				}
				rows = append(rows, RowData{ID: id, Timestamp: fact.Timestamp, Values: vals})
			}
		}
	}
	
	writeJSON(w, http.StatusOK, map[string]interface{}{"rows": rows})
}

// handleTableHistory returns all row events for a table in a time range.
func handleTableHistory(w http.ResponseWriter, r *http.Request, user string, store *db.StoreAdapter) {
	table := r.PathValue("table")
	
	// Validate table exists
	if !tableExists(r.Context(), store, user, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("table '%s' not found", table))
		return
	}
	
	q := r.URL.Query()
	startParam := q.Get("start")
	if startParam == "" {
		writeError(w, http.StatusBadRequest, "missing required 'start' parameter")
		return
	}
	
	endParam := q.Get("end")
	if endParam == "" {
		writeError(w, http.StatusBadRequest, "missing required 'end' parameter")
		return
	}
	
	start, err := parseTime(startParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid 'start' time format: %v (expected RFC3339)", err))
		return
	}
	
	end, err := parseTime(endParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid 'end' time format: %v (expected RFC3339)", err))
		return
	}
	
	// Validate time range
	if start.After(end) {
		writeError(w, http.StatusBadRequest, "'start' time must be before 'end' time")
		return
	}
	
	facts, err := store.QueryByTimeRange(r.Context(), start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to query time range: %v", err))
		return
	}
	
	events := []RowEvent{}
	prefix := fmt.Sprintf("%s/%s", user, table)
	for _, f := range facts {
		if f.Namespace == prefix && f.DataType == "json" {
			vals, ok := f.Value.(map[string]interface{})
			if !ok && f.Value != nil {
				log.Printf("Warning: invalid data format for row '%s' in history", f.FieldName)
				continue
			}
			events = append(events, RowEvent{ID: f.FieldName, Timestamp: f.Timestamp, Values: vals})
		}
	}
	
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events})
}
