# Notably Database Package

This package provides a clean abstraction layer for DynamoDB operations in the Notably application. It follows an interface-based design pattern to enable easy testing and flexibility for different storage backends.

## Overview

The `db` package consists of:

- A `Store` interface that defines all DynamoDB operations
- A concrete `DynamoDBStore` implementation of the interface
- A `MockStore` implementation for testing
- Comprehensive test suites

## Interface-Based Design

The core of this package is the `Store` interface, which defines all operations needed for fact storage:

```go
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
```

## Usage

### Creating a DynamoDB Store

```go
// Option 1: Using environment variables for configuration
store, err := db.NewDynamoDBStoreFromEnv(ctx, "MyTableName", "user123")
if err != nil {
    log.Fatalf("Failed to create store: %v", err)
}

// Option 2: With explicit configuration
cfg, err := config.LoadDefaultConfig(ctx)
if err != nil {
    log.Fatalf("Failed to load AWS config: %v", err)
}

store := db.NewDynamoDBStore(&db.Config{
    TableName:    "MyTableName",
    UserID:       "user123",
    DynamoClient: dynamodb.NewFromConfig(cfg),
})
```

### Basic Operations

```go
// Create the DynamoDB table (if it doesn't exist)
err := store.CreateTable(ctx)
if err != nil {
    log.Fatalf("Failed to create table: %v", err)
}

// Add a fact
fact := &db.Fact{
    ID:        "unique-id-1",
    Timestamp: time.Now(),
    Namespace: "user-profile",
    FieldName: "display-name",
    DataType:  db.DataTypeString,
    Value:     "John Doe",
}
err = store.PutFact(ctx, fact)

// Retrieve a fact
retrievedFact, err := store.GetFact(ctx, "unique-id-1")

// Update a fact (adds a new version)
updatedFact := &db.Fact{
    ID:        "unique-id-1",
    Timestamp: time.Now(),
    Namespace: "user-profile",
    FieldName: "display-name",
    DataType:  db.DataTypeString,
    Value:     "John A. Doe",
}
err = store.PutFact(ctx, updatedFact)

// Delete a fact (adds a tombstone)
err = store.DeleteFact(ctx, "unique-id-1")
```

### Querying

```go
// Set up query options
startTime := time.Now().Add(-24 * time.Hour)
endTime := time.Now()
opts := db.QueryOptions{
    StartTime:     &startTime,
    EndTime:       &endTime,
    SortAscending: true,
}

// Query by field
results, err := store.QueryByField(ctx, "user-profile", "display-name", opts)
for _, fact := range results.Facts {
    fmt.Printf("Version at %s: %s\n", fact.Timestamp, fact.Value)
}

// Query by namespace
results, err = store.QueryByNamespace(ctx, "user-profile", opts)

// Query by time range (across all namespaces)
results, err = store.QueryByTimeRange(ctx, opts)
```

### Snapshots

```go
// Get all fields in a namespace as of a specific time
snapshotTime := time.Now().Add(-1 * time.Hour)
snapshot, err := store.GetSnapshotAtTime(ctx, "user-profile", snapshotTime)

// Print snapshot data
for key, fact := range snapshot {
    fmt.Printf("%s: %s (as of %s)\n", fact.FieldName, fact.Value, fact.Timestamp)
}
```

## Testing

### Using the Mock Store

The package includes a `MockStore` implementation for testing:

```go
// Create a mock store
mockStore := db.NewMockStore()

// Set up mock behavior for testing
mockStore.ExpectCall("CreateTable", 1)
mockStore.ExpectCall("PutFact", 2)

// Testing code that uses the store...

// Verify that expected calls were made
err := mockStore.VerifyExpectations()
if err != nil {
    t.Errorf("Mock expectations not met: %v", err)
}

// Simulate failures for error handling tests
mockStore.SimulateFailure("GetFact", errors.New("simulated error"))
```

### Integration Tests

To run integration tests against a real DynamoDB or local emulator:

```bash
# Run with local DynamoDB
export DYNAMODB_ENDPOINT_URL=http://localhost:8000
export DYNAMODB_INTEGRATION_TEST=true
go test ./db/tests -v
```

## Error Handling

The package provides consistent error handling through the `StoreError` type and helper functions:

```go
// Check if an error represents a "not found" condition
if db.IsNotFound(err) {
    // Handle not found case
}
```

## Local Development

For development, you can use a local DynamoDB emulator:

```bash
# Run DynamoDB Local with Docker
docker run --name dynamodb-local -p 8000:8000 amazon/dynamodb-local

# Configure your application to use it
export DYNAMODB_ENDPOINT_URL=http://localhost:8000
export AWS_REGION=us-west-2
export AWS_ACCESS_KEY_ID=dummy
export AWS_SECRET_ACCESS_KEY=dummy
```