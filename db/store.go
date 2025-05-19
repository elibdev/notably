package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DataType represents the type of data stored in a fact
type DataType string

const (
	DataTypeString  DataType = "string"
	DataTypeNumber  DataType = "number"
	DataTypeBoolean DataType = "boolean"
	DataTypeJSON    DataType = "json"
)

// Fact represents a single piece of data with versioning
type Fact struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Namespace string    `json:"namespace"`
	FieldName string    `json:"fieldName"`
	DataType  DataType  `json:"dataType"`
	Value     string    `json:"value"`
	UserID    string    `json:"userId"`
	IsDeleted bool      `json:"isDeleted"`
}

// QueryOptions provides filtering and pagination options for queries
type QueryOptions struct {
	StartTime     *time.Time
	EndTime       *time.Time
	Limit         *int32
	NextToken     *string
	SortAscending bool
}

// QueryResult contains the results of a query operation
type QueryResult struct {
	Facts     []Fact
	NextToken *string
}

// Store defines the interface for DynamoDB operations
type Store interface {
	// Schema operations
	CreateTable(ctx context.Context) error
	DeleteTable(ctx context.Context) error

	// Fact operations
	PutFact(ctx context.Context, fact *Fact) error
	GetFact(ctx context.Context, id string) (*Fact, error)
	DeleteFact(ctx context.Context, id string) error

	// Query operations
	QueryByField(ctx context.Context, namespace, fieldName string, opts QueryOptions) (*QueryResult, error)
	QueryByTimeRange(ctx context.Context, opts QueryOptions) (*QueryResult, error)
	QueryByNamespace(ctx context.Context, namespace string, opts QueryOptions) (*QueryResult, error)

	// Snapshot operations
	GetSnapshotAtTime(ctx context.Context, namespace string, at time.Time) (map[string]Fact, error)
}

// Config holds the configuration for the DynamoDB store
type Config struct {
	TableName    string
	UserID       string
	DynamoClient *dynamodb.Client
}

// StoreError represents errors that can occur in the Store
type StoreError struct {
	Operation string
	Err       error
}

func (e *StoreError) Error() string {
	return fmt.Sprintf("store operation %s failed: %v", e.Operation, e.Err)
}

// IsNotFound returns true if the error indicates a record was not found
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var storeErr *StoreError
	if e, ok := err.(*StoreError); ok {
		storeErr = e
	} else {
		return false
	}

	_, ok := storeErr.Err.(*types.ResourceNotFoundException)
	return ok
}
