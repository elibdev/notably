package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/elibdev/notably/db"
	"github.com/elibdev/notably/dynamo"
	"github.com/elibdev/notably/pkg/auth"
	"github.com/rs/cors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Config holds configuration for the server
type Config struct {
	TableName      string
	Addr           string
	DynamoEndpoint string
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		TableName:      os.Getenv("DYNAMODB_TABLE_NAME"),
		Addr:           ":8080",
		DynamoEndpoint: os.Getenv("DYNAMODB_ENDPOINT_URL"),
	}
}

// Server represents the API server
type Server struct {
	config        Config
	mux           *http.ServeMux
	authenticator *auth.Authenticator
	userStore     auth.UserStore
}

// NewServer creates a new server with the given configuration
func NewServer(config Config) (*Server, error) {
	// Initialize user store
	userStore := auth.NewInMemoryUserStore()
	authenticator := auth.NewAuthenticator(userStore)

	// Create the server
	server := &Server{
		config:        config,
		mux:           http.NewServeMux(),
		authenticator: authenticator,
		userStore:     userStore,
	}

	// Register routes
	server.registerRoutes()

	return server, nil
}

// registerRoutes sets up all the API routes
// Helper function to check if a name contains only allowed characters
func isValidName(name string) bool {
	for _, r := range name {
		if !(('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// validateValueType checks if a value matches the expected data type
func validateValueType(value interface{}, dataType string) bool {
	switch dataType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		// Check if float64 (JSON numbers are decoded as float64)
		_, isFloat := value.(float64)
		if isFloat {
			return true
		}
		// If not a float, try int
		_, isInt := value.(int)
		return isInt
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "datetime":
		// Check if string format can be parsed as time
		str, ok := value.(string)
		if !ok {
			return false
		}
		_, err := time.Parse(time.RFC3339, str)
		return err == nil
	case "object", "json":
		// For object/json, we expect a map
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		// For arrays, check if it's a slice
		_, ok := value.([]interface{})
		return ok
	default:
		// Unknown type, consider valid
		return true
	}
}

// Helper function to check if a table exists for the given user
func tableExists(ctx context.Context, store *db.StoreAdapter, userID, table string) bool {
	snap, err := store.GetSnapshot(ctx, time.Now().UTC())
	if err != nil {
		log.Printf("Error checking if table exists for user %s, table %s: %v", userID, table, err)
		return false
	}

	if entries, ok := snap[userID]; ok {
		_, exists := entries[table]
		return exists
	}
	return false
}

func (s *Server) registerRoutes() {
	// Authentication endpoints (no auth required)
	s.mux.HandleFunc("POST /auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /auth/login", s.handleLogin)

	// API Key management (requires auth)
	auth := s.authenticator.RequireAuth(http.HandlerFunc(s.handleAPIKeysList))
	s.mux.Handle("GET /auth/keys", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleAPIKeyCreate))
	s.mux.Handle("POST /auth/keys", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleAPIKeyRevoke))
	s.mux.Handle("DELETE /auth/keys/{id}", auth)

	// Tables API (all require auth)
	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleListTables))
	s.mux.Handle("GET /tables", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleCreateTable))
	s.mux.Handle("POST /tables", auth)

	// Rows API
	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleListRows))
	s.mux.Handle("GET /tables/{table}/rows", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleGetRow))
	s.mux.Handle("GET /tables/{table}/rows/{id}", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleCreateRow))
	s.mux.Handle("POST /tables/{table}/rows", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleUpdateRow))
	s.mux.Handle("PUT /tables/{table}/rows/{id}", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleDeleteRow))
	s.mux.Handle("DELETE /tables/{table}/rows/{id}", auth)

	// Snapshot and history
	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleTableSnapshot))
	s.mux.Handle("GET /tables/{table}/snapshot", auth)

	auth = s.authenticator.RequireAuth(http.HandlerFunc(s.handleTableHistory))
	s.mux.Handle("GET /tables/{table}/history", auth)
}

// Run starts the server
func (s *Server) Run() error {
	log.Printf("Starting server on %s", s.config.Addr)

	// Create a CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"}, // Add your frontend URL
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	})

	// Use the middleware
	handler := c.Handler(s.mux)

	return http.ListenAndServe(s.config.Addr, handler)
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	// Implement graceful shutdown if needed
	return nil
}

// Helper methods

// getStoreForUser returns a store adapter for the given user ID
func (s *Server) getStoreForUser(ctx context.Context, userID string) (*db.StoreAdapter, error) {
	// Create AWS config
	opts := []func(*config.LoadOptions) error{}
	if s.config.DynamoEndpoint != "" {
		resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{URL: s.config.DynamoEndpoint, SigningRegion: region}, nil
		})
		opts = append(opts, config.WithEndpointResolver(resolver))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Printf("Error loading AWS config: %v", err)
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	// Create client and store
	client := dynamo.NewClient(cfg, s.config.TableName, userID)

	// Ensure the table exists (this is idempotent and safe to call every time)
	if err := client.CreateTable(ctx); err != nil {
		log.Printf("Error ensuring DynamoDB table exists: %v", err)
		return nil, fmt.Errorf("ensuring table exists: %w", err)
	}

	// Create adapter for the store
	store := db.NewStoreAdapter(db.CreateStoreFromClient(client))

	return store, nil
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

// writeError writes an error response in JSON format
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// newID generates a unique ID
func newID() string {
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}

// Auth handlers

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Username, email, and password are required")
		return
	}

	// Register user
	user, err := s.authenticator.RegisterUser(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if err == auth.ErrUserAlreadyExists {
			writeError(w, http.StatusConflict, "Username or email already exists")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to create user")
		}
		return
	}

	// Generate an API key for the new user
	_, rawKey, err := s.authenticator.GenerateAPIKey(r.Context(), user.ID, "default", 0)
	if err != nil {
		log.Printf("Error generating API key: %v", err)
		// Continue anyway, user was created
	}

	// Return user info
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"apiKey":   rawKey,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"` // Can be username or email
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Authenticate user
	user, err := s.authenticator.LoginUser(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate a new API key
	_, rawKey, err := s.authenticator.GenerateAPIKey(r.Context(), user.ID, "login-"+time.Now().Format(time.RFC3339), 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	// Return user info and API key
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"apiKey":   rawKey,
	})
}

func (s *Server) handleAPIKeysList(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// List API keys
	keys, err := s.authenticator.ListAPIKeys(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list API keys")
		return
	}

	// Return keys (without sensitive data)
	type keyInfo struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"createdAt"`
		ExpiresAt time.Time `json:"expiresAt"`
		LastUsed  time.Time `json:"lastUsed"`
		Revoked   bool      `json:"revoked"`
	}

	response := make([]keyInfo, 0, len(keys))
	for _, key := range keys {
		response = append(response, keyInfo{
			ID:        key.ID,
			Name:      key.Name,
			CreatedAt: key.CreatedAt,
			ExpiresAt: key.ExpiresAt,
			LastUsed:  key.LastUsed,
			Revoked:   key.Revoked,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"keys": response,
	})
}

func (s *Server) handleAPIKeyCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	var req struct {
		Name     string        `json:"name"`
		Duration time.Duration `json:"duration"` // In seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.Name == "" {
		req.Name = "api-key-" + time.Now().Format(time.RFC3339)
	}

	duration := req.Duration * time.Second
	if duration == 0 {
		duration = auth.DefaultAPIKeyExpiration
	}

	// Create new API key
	apiKey, rawKey, err := s.authenticator.GenerateAPIKey(r.Context(), user.ID, req.Name, duration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create API key")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        apiKey.ID,
		"name":      apiKey.Name,
		"apiKey":    rawKey,
		"createdAt": apiKey.CreatedAt,
		"expiresAt": apiKey.ExpiresAt,
	})
}

func (s *Server) handleAPIKeyRevoke(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	keyID := r.PathValue("id")
	if keyID == "" {
		writeError(w, http.StatusBadRequest, "Key ID is required")
		return
	}

	// Revoke API key
	err := s.authenticator.RevokeAPIKey(r.Context(), user.ID, keyID)
	if err != nil {
		if err == auth.ErrInsufficientPrivilege {
			writeError(w, http.StatusForbidden, "You do not have permission to revoke this key")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to revoke API key")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "API key revoked",
	})
}

// Table and row data types

// TableInfo represents metadata for a user table
type ColumnDefinition struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"`
}

// TableInfo represents metadata for a user table
type TableInfo struct {
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"createdAt"`
	Columns   []ColumnDefinition `json:"columns,omitempty"`
}

// RowData represents a row snapshot for a table
type RowData struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"`
}

// RowEvent represents a history event for a row
type RowEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"`
}

// Table handlers

func (s *Server) handleCreateTable(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	var req struct {
		Name    string            `json:"name"`
		Columns []ColumnDefinition `json:"columns,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Table name is required")
		return
	}

	// Validate table name format
	if !isValidName(req.Name) {
		writeError(w, http.StatusBadRequest, "Table name must contain only alphanumeric characters, hyphens, and underscores")
		return
	}

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		log.Printf("User %s: Failed to initialize storage: %v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage: "+err.Error())
		return
	}

	// Validate column definitions if provided
	if len(req.Columns) > 0 {
		for _, col := range req.Columns {
			if col.Name == "" {
				writeError(w, http.StatusBadRequest, "Column name is required")
				return
			}
			if !isValidName(col.Name) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Column name '%s' must contain only alphanumeric characters, hyphens, and underscores", col.Name))
				return
			}
			if col.DataType == "" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Data type is required for column '%s'", col.Name))
				return
			}
		}
	}

	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: user.ID,
		FieldName: req.Name,
		DataType:  "table",
		Value:     "",
		Columns:   req.Columns,
	}

	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create table: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, TableInfo{Name: req.Name, CreatedAt: fact.Timestamp, Columns: req.Columns})
}

func (s *Server) handleListTables(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		log.Printf("Failed to get tables for user %s: %v", user.ID, err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get tables: %v", err))
		return
	}

	tables := []TableInfo{}
	if entries, ok := snap[user.ID]; ok {
		for name, fact := range entries {
			tables = append(tables, TableInfo{Name: name, CreatedAt: fact.Timestamp})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"tables": tables})
}

// Row handlers

func (s *Server) handleCreateRow(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	table := r.PathValue("table")

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	// Validate table exists and get column definitions
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}
	
	tableDefinition := facts[0]
	var columns []ColumnDefinition
	if len(tableDefinition.Columns) > 0 {
		columns = tableDefinition.Columns
	}

	var req struct {
		ID     string                 `json:"id"`
		Values map[string]interface{} `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "Row ID is required")
		return
	}

	if req.Values == nil {
		writeError(w, http.StatusBadRequest, "Row values are required")
		return
	}

	// Validate values against column definitions if available
	if len(columns) > 0 {
		for colName, value := range req.Values {
			// Check if column is defined
			found := false
			var colDef ColumnDefinition
			
			for _, col := range columns {
				if col.Name == colName {
					found = true
					colDef = col
					break
				}
			}
			
			if !found {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Column '%s' is not defined in table schema", colName))
				return
			}
			
			// Validate type according to column definition
			valid := validateValueType(value, colDef.DataType)
			if !valid {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Value for column '%s' does not match expected type '%s'", colName, colDef.DataType))
				return
			}
		}
	}

	// Validate values against column definitions if available
	if len(columns) > 0 {
		for colName, value := range req.Values {
			// Check if column is defined
			found := false
			var colDef ColumnDefinition
			
			for _, col := range columns {
				if col.Name == colName {
					found = true
					colDef = col
					break
				}
			}
			
			if !found {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Column '%s' is not defined in table schema", colName))
				return
			}
			
			// Validate type according to column definition
			valid := validateValueType(value, colDef.DataType)
			if !valid {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Value for column '%s' does not match expected type '%s'", colName, colDef.DataType))
				return
			}
		}
	}

	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: fmt.Sprintf("%s/%s", user.ID, table),
		FieldName: req.ID,
		DataType:  "json",
		Value:     req.Values,
	}

	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create row: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, RowData{ID: req.ID, Timestamp: fact.Timestamp, Values: req.Values})
}

func (s *Server) handleListRows(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	table := r.PathValue("table")

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	// Validate table exists and get column definitions
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}
	
	tableDefinition := facts[0]
	var columns []ColumnDefinition
	if len(tableDefinition.Columns) > 0 {
		columns = tableDefinition.Columns
	}

	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get rows: %v", err))
		return
	}

	key := fmt.Sprintf("%s/%s", user.ID, table)
	rows := []RowData{}

	if entries, ok := snap[key]; ok {
		for id, fact := range entries {
			if fact.DataType == "json" {
				vals, ok := fact.Value.(map[string]interface{})
				if !ok {
					log.Printf("Warning: invalid data format for row '%s'", id)
					continue
				}
				rows = append(rows, RowData{ID: id, Timestamp: fact.Timestamp, Values: vals})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"rows": rows})
}

func (s *Server) handleGetRow(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	table := r.PathValue("table")
	rowID := r.PathValue("id")

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	// Validate table exists
	if !tableExists(r.Context(), store, user.ID, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get snapshot: %v", err))
		return
	}

	key := fmt.Sprintf("%s/%s", user.ID, table)
	if entries, ok := snap[key]; ok {
		if fact, ok := entries[rowID]; ok && fact.DataType == "json" {
			vals, ok := fact.Value.(map[string]interface{})
			if !ok {
				writeError(w, http.StatusInternalServerError, "Invalid row data format")
				return
			}
			writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: fact.Timestamp, Values: vals})
			return
		}
	}

	writeError(w, http.StatusNotFound, fmt.Sprintf("Row '%s' not found in table '%s'", rowID, table))
}

func (s *Server) handleUpdateRow(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	table := r.PathValue("table")
	rowID := r.PathValue("id")

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	// Validate table exists
	if !tableExists(r.Context(), store, user.ID, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	// Validate row exists
	key := fmt.Sprintf("%s/%s", user.ID, table)
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get snapshot: %v", err))
		return
	}

	rowExists := false
	if entries, ok := snap[key]; ok {
		_, rowExists = entries[rowID]
	}

	if !rowExists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Row '%s' not found in table '%s'", rowID, table))
		return
	}

	var req struct {
		Values map[string]interface{} `json:"values"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Values == nil {
		writeError(w, http.StatusBadRequest, "Row values are required")
		return
	}

	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: fmt.Sprintf("%s/%s", user.ID, table),
		FieldName: rowID,
		DataType:  "json",
		Value:     req.Values,
	}

	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update row: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: fact.Timestamp, Values: req.Values})
}

func (s *Server) handleDeleteRow(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	table := r.PathValue("table")
	rowID := r.PathValue("id")

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	// Validate table exists
	if !tableExists(r.Context(), store, user.ID, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: fmt.Sprintf("%s/%s", user.ID, table),
		FieldName: rowID,
		DataType:  "json",
		Value:     nil,
	}

	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete row: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTableSnapshot(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	table := r.PathValue("table")

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	// Validate table exists
	if !tableExists(r.Context(), store, user.ID, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	q := r.URL.Query()
	atParam := q.Get("at")
	var at time.Time
	if atParam == "" {
		at = time.Now().UTC()
	} else {
		var err error
		at, err = time.Parse(time.RFC3339, atParam)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid 'at' time format: %v (expected RFC3339)", err))
			return
		}
	}

	snap, err := store.GetSnapshot(r.Context(), at)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get snapshot: %v", err))
		return
	}

	key := fmt.Sprintf("%s/%s", user.ID, table)
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

func (s *Server) handleTableHistory(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	table := r.PathValue("table")

	// Get store for user
	store, err := s.getStoreForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize storage")
		return
	}

	// Validate table exists
	if !tableExists(r.Context(), store, user.ID, table) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	q := r.URL.Query()
	startParam := q.Get("start")
	if startParam == "" {
		writeError(w, http.StatusBadRequest, "Missing required 'start' parameter")
		return
	}

	endParam := q.Get("end")
	if endParam == "" {
		writeError(w, http.StatusBadRequest, "Missing required 'end' parameter")
		return
	}

	start, err := time.Parse(time.RFC3339, startParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid 'start' time format: %v (expected RFC3339)", err))
		return
	}

	end, err := time.Parse(time.RFC3339, endParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid 'end' time format: %v (expected RFC3339)", err))
		return
	}

	// Validate time range
	if start.After(end) {
		writeError(w, http.StatusBadRequest, "'start' time must be before 'end' time")
		return
	}

	facts, err := store.QueryByTimeRange(r.Context(), start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to query time range: %v", err))
		return
	}

	events := []RowEvent{}
	prefix := fmt.Sprintf("%s/%s", user.ID, table)
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
