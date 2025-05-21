package dynamotest

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// ColumnDefinition represents a column in a table with its type
type ColumnDefinition struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"`
}

// Fact represents a single versioned value for a field.
type Fact struct {
	ID        string
	Timestamp time.Time
	Namespace string
	FieldName string
	DataType  string
	Value     interface{}
	// For table definitions, this will contain column definitions
	Columns []ColumnDefinition `json:"columns,omitempty"`
}

// DynamoClient defines the interface for dynamoDB operations needed for testing
type DynamoClient interface {
	// CreateTable creates the DynamoDB table and necessary indexes
	CreateTable(ctx context.Context) error

	// PutFact writes a fact to DynamoDB
	PutFact(ctx context.Context, fact Fact) error

	// QueryByField returns facts in a namespace/fieldName for the user in a time range
	QueryByField(ctx context.Context, namespace, fieldName string, start, end time.Time) ([]Fact, error)

	// QueryByTimeRange returns all facts for the user in the time range
	QueryByTimeRange(ctx context.Context, start, end time.Time) ([]Fact, error)
}

// RawDynamoDBAPI defines the interface for DynamoDB operations needed for the client
type RawDynamoDBAPI interface {
	CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
}

// ClientConfig holds the configuration for creating a new DynamoDB client
type ClientConfig struct {
	Config    aws.Config
	TableName string
	UserID    string
}

// NewClientFunc is a function type that creates a new DynamoDB client
type NewClientFunc func(cfg aws.Config, tableName, userID string) DynamoClient
