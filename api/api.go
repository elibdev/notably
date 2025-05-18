// Package api provides a unified, higher-level API for managing versioned facts in DynamoDB.
// It wraps the lower-level dynamo.Client and offers methods for schema creation,
// fact operations (add, update, delete), and time-travel views (snapshots).
package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/elibdev/notably/dynamo"
)

const envEndpointVar = "DYNAMODB_ENDPOINT_URL"

// FieldSnapshot is the value and metadata of a field at a specific timestamp.
type FieldSnapshot struct {
	Value     interface{} `json:"value"`
	DataType  string      `json:"dataType"`
	Timestamp time.Time   `json:"timestamp"`
}

// Snapshot maps namespace -> fieldName -> FieldSnapshot representing the latest
// version of each field as of a given point in time.
type Snapshot map[string]map[string]FieldSnapshot

// Fact is an alias for dynamo.Fact for convenience.
type Fact = dynamo.Fact

// FactStore provides a unified API for schema management and fact operations.
type FactStore struct {
	client *dynamo.Client
}

// NewFactStore initializes a FactStore for the given table and user. It loads AWS
// configuration from the environment (credentials, region), and if the
// DYNAMODB_ENDPOINT_URL environment variable is set, it configures the client
// to use the specified endpoint (e.g., a local DynamoDB emulator).
func NewFactStore(ctx context.Context, tableName, userID string) (*FactStore, error) {
	opts := []func(*config.LoadOptions) error{}
	if ep := os.Getenv(envEndpointVar); ep != "" {
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
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	client := dynamo.NewClient(cfg, tableName, userID)
	return &FactStore{client: client}, nil
}

// CreateSchema creates the DynamoDB table and the GSI for field-based queries.
func (fs *FactStore) CreateSchema(ctx context.Context) error {
	return fs.client.CreateTable(ctx)
}

// AddFact writes a new fact or a new version of an existing fact.
func (fs *FactStore) AddFact(ctx context.Context, fact Fact) error {
	return fs.client.PutFact(ctx, fact)
}

// UpdateFact is an alias for AddFact, allowing semantic clarity.
func (fs *FactStore) UpdateFact(ctx context.Context, fact Fact) error {
	return fs.AddFact(ctx, fact)
}

// DeleteFact records a tombstone fact for the given namespace, field, and ID at the specified timestamp.
// Consumers can interpret a DataType of "deleted" (and a nil Value) as a deletion marker.
func (fs *FactStore) DeleteFact(ctx context.Context, namespace, fieldName, id string, timestamp time.Time) error {
	fact := Fact{
		ID:        id,
		Timestamp: timestamp,
		Namespace: namespace,
		FieldName: fieldName,
		DataType:  "deleted",
		Value:     nil,
	}
	return fs.client.PutFact(ctx, fact)
}

// QueryByField returns all versions of a specific field within the time range [start, end].
func (fs *FactStore) QueryByField(ctx context.Context, namespace, fieldName string, start, end time.Time) ([]Fact, error) {
	return fs.client.QueryByField(ctx, namespace, fieldName, start, end)
}

// QueryByTimeRange returns all facts for the user within the time range [start, end].
func (fs *FactStore) QueryByTimeRange(ctx context.Context, start, end time.Time) ([]Fact, error) {
	return fs.client.QueryByTimeRange(ctx, start, end)
}

// SnapshotAt returns the latest value for each field as of the specified timestamp.
// It does so by querying all facts up to that time and retaining the most recent entry
// per namespace/fieldName.
func (fs *FactStore) SnapshotAt(ctx context.Context, ts time.Time) (Snapshot, error) {
	facts, err := fs.client.QueryByTimeRange(ctx, time.Time{}, ts)
	if err != nil {
		return nil, err
	}
	snap := make(Snapshot)
	for _, f := range facts {
		ns := f.Namespace
		if _, ok := snap[ns]; !ok {
			snap[ns] = make(map[string]FieldSnapshot)
		}
		prev, exists := snap[ns][f.FieldName]
		if !exists || f.Timestamp.After(prev.Timestamp) {
			snap[ns][f.FieldName] = FieldSnapshot{
				Value:     f.Value,
				DataType:  f.DataType,
				Timestamp: f.Timestamp,
			}
		}
	}
	return snap, nil
}
