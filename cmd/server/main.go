package main

import (
	"context"
	"encoding/json"
	"flag"
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
	var (
		tableName string
		userID    string
		addr      string
	)
	flag.StringVar(&tableName, "table", "", "DynamoDB table name (required)")
	flag.StringVar(&userID, "user", "", "User ID (required)")
	flag.StringVar(&addr, "addr", ":8080", "HTTP listen address")
	flag.Parse()
	if tableName == "" || userID == "" {
		flag.Usage()
		log.Fatal("table and user flags are required")
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

	legacyClient := dynamo.NewClient(cfg, tableName, userID)
	store := db.NewStoreAdapter(db.CreateStoreFromClient(legacyClient))

	if err := store.CreateTable(ctx); err != nil {
		log.Fatalf("creating table: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/facts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleQueryByTimeRange(ctx, store, w, r)
		case http.MethodPost:
			handleAddFact(ctx, store, w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/facts/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/facts/")
		if id == "" {
			writeError(w, http.StatusNotFound, "missing fact id")
			return
		}
		switch r.Method {
		case http.MethodGet:
			handleGetFact(ctx, store, w, r, id)
		case http.MethodDelete:
			handleDeleteFact(ctx, store, w, r, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/namespaces/", func(w http.ResponseWriter, r *http.Request) {
		segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/namespaces/"), "/")
		if len(segments) < 2 {
			writeError(w, http.StatusNotFound, "invalid namespace path")
			return
		}
		namespace := segments[0]
		switch {
		case len(segments) >= 4 && segments[1] == "fields" && segments[3] == "facts":
			fieldName := segments[2]
			handleQueryByField(ctx, store, w, r, namespace, fieldName)
		case len(segments) == 2 && segments[1] == "facts":
			handleQueryByNamespace(ctx, store, w, r, namespace)
		default:
			writeError(w, http.StatusNotFound, "invalid namespace path")
		}
	})
	mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleGetSnapshot(ctx, store, w, r)
	})

	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// FactResponse represents a fact in JSON responses.
type FactResponse struct {
	ID        string      `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	Namespace string      `json:"namespace"`
	FieldName string      `json:"fieldName"`
	DataType  string      `json:"dataType"`
	Value     interface{} `json:"value"`
}

// QueryResponse is the JSON structure for list of facts.
type QueryResponse struct {
	Facts []FactResponse `json:"facts"`
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

// parseTime parses an RFC3339 timestamp from query parameter.
func parseTime(val string) (time.Time, error) {
	return time.Parse(time.RFC3339, val)
}

func toFactResponse(f dynamo.Fact) FactResponse {
	return FactResponse{
		ID:        f.ID,
		Timestamp: f.Timestamp,
		Namespace: f.Namespace,
		FieldName: f.FieldName,
		DataType:  f.DataType,
		Value:     f.Value,
	}
}

func handleAddFact(ctx context.Context, store *db.StoreAdapter, w http.ResponseWriter, r *http.Request) {
	var fact dynamo.Fact
	if err := json.NewDecoder(r.Body).Decode(&fact); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := store.PutFact(ctx, fact); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toFactResponse(fact))
}

func handleGetFact(ctx context.Context, store *db.StoreAdapter, w http.ResponseWriter, r *http.Request, id string) {
	fact, err := store.GetFactByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toFactResponse(*fact))
}

func handleDeleteFact(ctx context.Context, store *db.StoreAdapter, w http.ResponseWriter, r *http.Request, id string) {
	if err := store.DeleteFactByID(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleQueryByField(ctx context.Context, store *db.StoreAdapter, w http.ResponseWriter, r *http.Request, namespace, fieldName string) {
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
	facts, err := store.QueryByField(ctx, namespace, fieldName, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := QueryResponse{Facts: make([]FactResponse, len(facts))}
	for i, f := range facts {
		resp.Facts[i] = toFactResponse(f)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleQueryByNamespace(ctx context.Context, store *db.StoreAdapter, w http.ResponseWriter, r *http.Request, namespace string) {
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
	filtered := make([]FactResponse, 0, len(facts))
	for _, f := range facts {
		if f.Namespace == namespace {
			filtered = append(filtered, toFactResponse(f))
		}
	}
	writeJSON(w, http.StatusOK, QueryResponse{Facts: filtered})
}

func handleQueryByTimeRange(ctx context.Context, store *db.StoreAdapter, w http.ResponseWriter, r *http.Request) {
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
	resp := QueryResponse{Facts: make([]FactResponse, len(facts))}
	for i, f := range facts {
		resp.Facts[i] = toFactResponse(f)
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleGetSnapshot(ctx context.Context, store *db.StoreAdapter, w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	atParam := q.Get("at")
	at, err := parseTime(atParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid at time")
		return
	}
	snap, err := store.GetSnapshot(ctx, at)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make(map[string]map[string]FactResponse, len(snap))
	for ns, fields := range snap {
		grp := make(map[string]FactResponse, len(fields))
		for name, f := range fields {
			grp[name] = toFactResponse(f)
		}
		result[ns] = grp
	}
	writeJSON(w, http.StatusOK, result)
}
