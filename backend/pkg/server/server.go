package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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
	corsHandler   http.Handler
	awsCfg        aws.Config // New field
}

// NewServer creates a new server with the given configuration
func NewServer(config Config) (*Server, error) {
	// Initialize user store
	userStore := auth.NewInMemoryUserStore()
	authenticator := auth.NewAuthenticator(userStore)

	// Load AWS config once
	opts := []func(*config.LoadOptions) error{}
	if config.DynamoEndpoint != "" {
		resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{URL: config.DynamoEndpoint, SigningRegion: region}, nil
		})
		opts = append(opts, config.WithEndpointResolver(resolver))
	}
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), opts...) // Use context.TODO() or background
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Ensure table exists on startup (using a temporary client for this)
	// The userID for CreateTable doesn't strictly matter if it's just ensuring table structure.
	// However, the dynamo.NewClient requires a userID. We can pass a placeholder or an admin/system ID.
	// Let's use a placeholder "system_user_for_table_creation".
	tempDynamoClient := dynamo.NewClient(awsCfg, config.TableName, "system_user_for_table_creation")
	// It's important to use a context for CreateTable. context.Background() is suitable for startup.
	if err := tempDynamoClient.CreateTable(context.Background()); err != nil {
		// Log the error but don't necessarily fail server startup,
		// as the table might already exist and permissions might be the issue.
		// Or, decide to fail hard if table creation is critical.
		// For now, let's log and continue.
		log.Printf("Warning: Failed to ensure DynamoDB table '%s' exists on startup: %v", config.TableName, err)
	}

	server := &Server{
		config:        config,
		mux:           http.NewServeMux(),
		authenticator: authenticator,
		userStore:     userStore,
		awsCfg:        awsCfg, // Store the loaded config
	}

	server.registerRoutes()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		Debug:          true,
	})
	server.corsHandler = c.Handler(server.mux)

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
		// Unknown type, consider invalid
		return false // Changed from true to false
	}
}

func init() {
	// Seed the random number generator for ID generation
	rand.Seed(time.Now().UnixNano())
}

func (s *Server) registerRoutes() {
	// Health check endpoint (no auth required)
	s.mux.HandleFunc("GET /health", s.handleHealth)

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
	// Use the pre-configured CORS handler
	return http.ListenAndServe(s.config.Addr, s.corsHandler)
}

// handleHealth returns a simple health check response
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"notably-api"}`))
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	// Implement graceful shutdown if needed
	return nil
}

// Helper methods

// getStoreForUser returns a store adapter for the given user ID
func (s *Server) getStoreForUser(ctx context.Context, userID string) (*db.StoreAdapter, error) {
	// Use the stored AWS config
	// Create client and store
	// The dynamo.NewClient is light enough to be created per call if userID specific behavior is needed.
	// If dynamo.Client could be shared and userID passed to its methods, that would be even better.
	// For now, this is a good improvement.
	client := dynamo.NewClient(s.awsCfg, s.config.TableName, userID)

	// The CreateTable call has been moved to NewServer.

	// Create adapter for the store
	// db.CreateStoreFromClient uses the old dynamo.Client which expects the dynamo.Fact
	// db.NewStoreAdapter wraps a db.Store (like DynamoDBStore)
	// The existing line is: store := db.NewStoreAdapter(db.CreateStoreFromClient(client))
	// This implies db.CreateStoreFromClient(client) returns a db.Store compatible thing.
	// Let's assume this structure is correct and db.CreateStoreFromClient adapts the dynamo.Client to db.Store interface.
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
	// Create a more robust ID format (similar to ULID)
	// Format: timestamp + random component
	now := time.Now().UTC()
	timestamp := now.Format("20060102150405.000")
	randomPart := make([]byte, 8)
	for i := range randomPart {
		randomPart[i] = byte(rand.Intn(256))
	}
	return fmt.Sprintf("%s_%x", timestamp, randomPart)
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
type TableInfo struct {
	Name      string                    `json:"name"`
	CreatedAt time.Time                 `json:"createdAt"`
	Columns   []dynamo.ColumnDefinition `json:"columns,omitempty"`
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
		Name    string                    `json:"name"`
		Columns []dynamo.ColumnDefinition `json:"columns,omitempty"`
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

	// Query all facts for the user and filter for table definitions
	facts, err := store.QueryByTimeRange(r.Context(), time.Time{}, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get tables: %v", err))
		return
	}

	tables := []TableInfo{}
	for _, fact := range facts {
		// Only include facts that are table definitions
		if fact.Namespace == user.ID && fact.DataType == "table" {
			tables = append(tables, TableInfo{
				Name:      fact.FieldName,
				CreatedAt: fact.Timestamp,
				Columns:   fact.Columns,
			})
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
	var columns []dynamo.ColumnDefinition
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

	// Always auto-generate ID if not provided
	if req.ID == "" {
		req.ID = newID()
		log.Printf("Auto-generated row ID: %s", req.ID)
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
			var colDef dynamo.ColumnDefinition

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

	// Validate table exists and get column definitions
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	// We found the table definition, now get the snapshot
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
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	// Read the request body
	var req struct {
		Values map[string]interface{} `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}
	if req.Values == nil {
		writeError(w, http.StatusBadRequest, "Row values are required for update")
		return
	}

	// Check if the row exists
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get snapshot: %v", err))
		return
	}

	key := fmt.Sprintf("%s/%s", user.ID, table)
	rowExists := false
	if entries, ok := snap[key]; ok {
		if fact, ok := entries[rowID]; ok && fact.DataType == "json" {
			rowExists = true
		}
	}

	if !rowExists {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Row '%s' not found in table '%s'", rowID, table))
		return
	}

	// Validate against column definitions (similar to handleCreateRow)
	tableFacts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(tableFacts) == 0 {
		// This should ideally not happen if the table check passed earlier, but good to have
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' definition not found", table))
		return
	}

	if len(tableFacts[0].Columns) > 0 {
		for colName, value := range req.Values {
			found := false
			var colDef dynamo.ColumnDefinition
			for _, col := range tableFacts[0].Columns {
				if col.Name == colName {
					found = true
					colDef = col
					break
				}
			}

			if !found {
				// Depending on requirements, you might allow adding new columns
				// or enforce that only defined columns can be updated.
				// For now, let's assume we only update existing, defined columns.
				// If you want to allow adding new columns, remove this check.
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Column '%s' is not defined in table schema. Updates are restricted to defined columns.", colName))
				return
			}

			if !validateValueType(value, colDef.DataType) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Value for column '%s' does not match expected type '%s'", colName, colDef.DataType))
				return
			}
		}
	}

	// Create the updated fact
	updatedFact := dynamo.Fact{
		ID:        newID(), // This creates a new version of the fact (event sourcing)
		Timestamp: time.Now().UTC(),
		Namespace: key,   // Namespace for the row data (user.ID/table)
		FieldName: rowID, // FieldName stores the actual Row ID
		DataType:  "json",
		Value:     req.Values, // The new values for the row
	}

	// Save the updated fact to the store
	if err := store.PutFact(r.Context(), updatedFact); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update row: %v", err))
		return
	}

	// Return the updated row data
	writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: updatedFact.Timestamp, Values: req.Values})
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
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Table '%s' not found", table))
		return
	}

	// Get current snapshot for the table
	snap, err := store.GetSnapshot(r.Context(), time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get snapshot: %v", err))
		return
	}

	key := fmt.Sprintf("%s/%s", user.ID, table)

	// Look for the row in the snapshot
	if entries, ok := snap[key]; ok {
		if fact, ok := entries[rowID]; ok && fact.DataType == "json" {
			vals, dataOk := fact.Value.(map[string]interface{})
			if !dataOk {
				writeError(w, http.StatusInternalServerError, "Invalid row data format")
				return
			}
			writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: fact.Timestamp, Values: vals})
			return
		}
	}

	// If the row is not found after these checks, return a 404 error
	writeError(w, http.StatusNotFound, fmt.Sprintf("Row '%s' not found in table '%s'", rowID, table))
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

	// Validate table exists and get column definitions
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
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

	// Validate table exists
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
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

// handleTableSnapshot returns a snapshot of a table at a given point in time

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
	facts, err := store.QueryByField(r.Context(), user.ID, table, time.Time{}, time.Now().UTC())
	if err != nil || len(facts) == 0 {
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

	facts, err = store.QueryByTimeRange(r.Context(), start, end)
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
