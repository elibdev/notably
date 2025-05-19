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

	// CRUD for user tables
	mux.HandleFunc("/tables", func(w http.ResponseWriter, r *http.Request) {
		user, err := requireUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		store := getStore(user)
		switch r.Method {
		case http.MethodPost:
			createTable(w, r, store)
		case http.MethodGet:
			listTables(w, r, store)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	// Operations on a specific table: rows, snapshot, history
	mux.HandleFunc("/tables/", func(w http.ResponseWriter, r *http.Request) {
		user, err := requireUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		store := getStore(user)
		// Trim prefix and split path: /tables/{table}[/{action}[/{id}]]
		path := strings.TrimPrefix(r.URL.Path, "/tables/")
		path = strings.TrimPrefix(path, "/")
		parts := strings.Split(path, "/")
		table := parts[0]
		if table == "" {
			writeError(w, http.StatusNotFound, "missing table name")
			return
		}
		// Delegate by action
		if len(parts) == 1 {
			writeError(w, http.StatusNotFound, "invalid table path")
			return
		}
		action := parts[1]
		subID := ""
		if len(parts) >= 3 {
			subID = parts[2]
		}
		switch action {
		case "rows":
			handleRows(r.Context(), w, r, store, user, table, subID)
		case "snapshot":
			handleTableSnapshot(r.Context(), w, r, store, user, table)
		case "history":
			handleTableHistory(r.Context(), w, r, store, user, table)
		default:
			writeError(w, http.StatusNotFound, "invalid table action")
		}
	})

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
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes an error message as JSON.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
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

// createTable records a new table under the user's namespace.
func createTable(w http.ResponseWriter, r *http.Request, store *db.StoreAdapter) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid JSON or missing table name")
		return
	}
	fact := dynamo.Fact{
		ID:        newID(),
		Timestamp: time.Now().UTC(),
		Namespace: r.Header.Get("X-User-ID"),
		FieldName: req.Name,
		DataType:  "string",
		Value:     "",
	}
	if err := store.PutFact(r.Context(), fact); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, TableInfo{Name: req.Name, CreatedAt: fact.Timestamp})
}

// listTables returns the list of tables for the user.
func listTables(w http.ResponseWriter, r *http.Request, store *db.StoreAdapter) {
	user := r.Header.Get("X-User-ID")
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

// handleRows handles CRUD operations on rows within a table.
func handleRows(ctx context.Context, w http.ResponseWriter, r *http.Request, store *db.StoreAdapter, user, table, rowID string) {
	switch r.Method {
	case http.MethodPost:
		// create a new row version
		var req struct {
			ID     string                 `json:"id"`
			Values map[string]interface{} `json:"values"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			writeError(w, http.StatusBadRequest, "invalid JSON or missing row id")
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
		if err := store.PutFact(ctx, fact); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, RowData{ID: req.ID, Timestamp: fact.Timestamp, Values: req.Values})

	case http.MethodGet:
		// list or get a single row snapshot
		if rowID == "" {
			// list current rows
			snap, err := store.GetSnapshot(ctx, time.Now().UTC())
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
		} else {
			// get single row
			snap, err := store.GetSnapshot(ctx, time.Now().UTC())
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			key := fmt.Sprintf("%s/%s", user, table)
			if entries, ok := snap[key]; ok {
				if fact, ok := entries[rowID]; ok && fact.DataType == "json" {
					vals, _ := fact.Value.(map[string]interface{})
					writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: fact.Timestamp, Values: vals})
					return
				}
			}
			writeError(w, http.StatusNotFound, "row not found")
		}

	case http.MethodPut:
		// update row (new version)
		if rowID == "" {
			writeError(w, http.StatusBadRequest, "missing row id")
			return
		}
		var req struct {
			Values map[string]interface{} `json:"values"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
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
		if err := store.PutFact(ctx, fact); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, RowData{ID: rowID, Timestamp: fact.Timestamp, Values: req.Values})

	case http.MethodDelete:
		// delete row (tombstone)
		if rowID == "" {
			writeError(w, http.StatusBadRequest, "missing row id")
			return
		}
		fact := dynamo.Fact{
			ID:        newID(),
			Timestamp: time.Now().UTC(),
			Namespace: fmt.Sprintf("%s/%s", user, table),
			FieldName: rowID,
			DataType:  "json",
			Value:     nil,
		}
		if err := store.PutFact(ctx, fact); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleTableSnapshot returns the state of a table at a given time.
func handleTableSnapshot(ctx context.Context, w http.ResponseWriter, r *http.Request, store *db.StoreAdapter, user, table string) {
	q := r.URL.Query()
	atParam := q.Get("at")
	var at time.Time
	var err error
	if atParam == "" {
		at = time.Now().UTC()
	} else {
		at, err = parseTime(atParam)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid at time")
			return
		}
	}
	snap, err := store.GetSnapshot(ctx, at)
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

// handleTableHistory returns all row events for a table in a time range.
func handleTableHistory(ctx context.Context, w http.ResponseWriter, r *http.Request, store *db.StoreAdapter, user, table string) {
	q := r.URL.Query()
	start, err := parseTime(q.Get("start"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid start time")
		return
	}
	end, err := parseTime(q.Get("end"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid end time")
		return
	}
	facts, err := store.QueryByTimeRange(ctx, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	events := []RowEvent{}
	prefix := fmt.Sprintf("%s/%s", user, table)
	for _, f := range facts {
		if f.Namespace == prefix && f.DataType == "json" {
			vals, _ := f.Value.(map[string]interface{})
			events = append(events, RowEvent{ID: f.FieldName, Timestamp: f.Timestamp, Values: vals})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events})
}
