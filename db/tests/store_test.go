package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elibdev/notably/db"
)

// testStore runs a standard suite of tests against any Store implementation
func testStore(t *testing.T, store db.Store) {
	ctx := context.Background()

	// Setup - create the table
	err := store.CreateTable(ctx)
	require.NoError(t, err, "CreateTable should succeed")

	// Test basic operations
	t.Run("CRUD operations", func(t *testing.T) {
		testCRUDOperations(t, ctx, store)
	})

	t.Run("Query operations", func(t *testing.T) {
		testQueryOperations(t, ctx, store)
	})

	t.Run("Snapshot operations", func(t *testing.T) {
		testSnapshotOperations(t, ctx, store)
	})

	// Cleanup - delete the table
	err = store.DeleteTable(ctx)
	assert.NoError(t, err, "DeleteTable should succeed")
}

func testCRUDOperations(t *testing.T, ctx context.Context, store db.Store) {
	// Create a fact
	now := time.Now().UTC()
	testFact := &db.Fact{
		ID:        "test-fact-1",
		Timestamp: now,
		Namespace: "test-namespace",
		FieldName: "test-field",
		DataType:  db.DataTypeString,
		Value:     "test-value",
		UserID:    "test-user",
	}

	// Test PutFact
	err := store.PutFact(ctx, testFact)
	require.NoError(t, err, "PutFact should succeed")

	// Test GetFact
	retrievedFact, err := store.GetFact(ctx, testFact.ID)
	require.NoError(t, err, "GetFact should succeed")
	assert.Equal(t, testFact.ID, retrievedFact.ID)
	assert.Equal(t, testFact.Namespace, retrievedFact.Namespace)
	assert.Equal(t, testFact.FieldName, retrievedFact.FieldName)
	assert.Equal(t, testFact.Value, retrievedFact.Value)
	assert.Equal(t, testFact.DataType, retrievedFact.DataType)

	// Test updating a fact
	updatedFact := &db.Fact{
		ID:        testFact.ID,
		Timestamp: now.Add(time.Second),
		Namespace: testFact.Namespace,
		FieldName: testFact.FieldName,
		DataType:  db.DataTypeString,
		Value:     "updated-value",
		UserID:    testFact.UserID,
	}

	err = store.PutFact(ctx, updatedFact)
	require.NoError(t, err, "Updating a fact should succeed")

	retrievedUpdatedFact, err := store.GetFact(ctx, testFact.ID)
	require.NoError(t, err, "GetFact after update should succeed")
	assert.Equal(t, updatedFact.Value, retrievedUpdatedFact.Value)
	assert.NotEqual(t, testFact.Value, retrievedUpdatedFact.Value)

	// Test deleting a fact
	err = store.DeleteFact(ctx, testFact.ID)
	require.NoError(t, err, "DeleteFact should succeed")

	// Verify deletion
	result, err := store.QueryByField(ctx, testFact.Namespace, testFact.FieldName, db.QueryOptions{
		StartTime:     &now,
		EndTime:       &time.Time{Add: now, Duration: time.Hour},
		SortAscending: false,
	})
	require.NoError(t, err, "Query after delete should succeed")

	// Find the latest fact in results which should be a deletion marker
	var found bool
	for _, f := range result.Facts {
		if f.ID == testFact.ID && f.IsDeleted {
			found = true
			break
		}
	}
	assert.True(t, found, "DeleteFact should create a deletion marker")
}

func testQueryOperations(t *testing.T, ctx context.Context, store db.Store) {
	// Create multiple facts with various attributes for querying
	baseTime := time.Now().UTC()
	facts := []*db.Fact{
		{
			ID:        "query-fact-1",
			Timestamp: baseTime,
			Namespace: "query-ns",
			FieldName: "field1",
			DataType:  db.DataTypeString,
			Value:     "value1",
			UserID:    "test-user",
		},
		{
			ID:        "query-fact-2",
			Timestamp: baseTime.Add(time.Minute),
			Namespace: "query-ns",
			FieldName: "field1",
			DataType:  db.DataTypeString,
			Value:     "value2",
			UserID:    "test-user",
		},
		{
			ID:        "query-fact-3",
			Timestamp: baseTime.Add(2 * time.Minute),
			Namespace: "query-ns",
			FieldName: "field2",
			DataType:  db.DataTypeNumber,
			Value:     "42",
			UserID:    "test-user",
		},
		{
			ID:        "query-fact-4",
			Timestamp: baseTime.Add(3 * time.Minute),
			Namespace: "other-ns",
			FieldName: "field3",
			DataType:  db.DataTypeBoolean,
			Value:     "true",
			UserID:    "test-user",
		},
	}

	// Insert facts
	for _, fact := range facts {
		err := store.PutFact(ctx, fact)
		require.NoError(t, err, "PutFact should succeed")
	}

	// Test QueryByField
	t.Run("QueryByField", func(t *testing.T) {
		startTime := baseTime.Add(-time.Minute)
		endTime := baseTime.Add(5 * time.Minute)
		result, err := store.QueryByField(ctx, "query-ns", "field1", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			SortAscending: true,
		})
		require.NoError(t, err, "QueryByField should succeed")
		assert.Len(t, result.Facts, 2, "Should return 2 facts for field1")
		assert.Equal(t, "value1", result.Facts[0].Value, "First fact should be value1")
		assert.Equal(t, "value2", result.Facts[1].Value, "Second fact should be value2")
	})

	// Test QueryByTimeRange
	t.Run("QueryByTimeRange", func(t *testing.T) {
		startTime := baseTime.Add(1 * time.Minute)
		endTime := baseTime.Add(4 * time.Minute)
		result, err := store.QueryByTimeRange(ctx, db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			SortAscending: false,
		})
		require.NoError(t, err, "QueryByTimeRange should succeed")
		assert.Len(t, result.Facts, 3, "Should return 3 facts in time range")
		// Facts should be in descending order by timestamp
		assert.Equal(t, "other-ns", result.Facts[0].Namespace, "First fact should be from other-ns")
		assert.Equal(t, "query-ns", result.Facts[1].Namespace, "Second fact should be from query-ns")
		assert.Equal(t, "field2", result.Facts[1].FieldName, "Second fact should be field2")
	})

	// Test QueryByNamespace
	t.Run("QueryByNamespace", func(t *testing.T) {
		startTime := baseTime.Add(-time.Minute)
		endTime := baseTime.Add(5 * time.Minute)
		result, err := store.QueryByNamespace(ctx, "query-ns", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			SortAscending: true,
		})
		require.NoError(t, err, "QueryByNamespace should succeed")
		assert.Len(t, result.Facts, 3, "Should return 3 facts for query-ns")
		assert.Equal(t, "field1", result.Facts[0].FieldName, "First fact should be field1")
		assert.Equal(t, "field1", result.Facts[1].FieldName, "Second fact should be field1")
		assert.Equal(t, "field2", result.Facts[2].FieldName, "Third fact should be field2")
	})
}

func testSnapshotOperations(t *testing.T, ctx context.Context, store db.Store) {
	// Create multiple facts with various timestamps for snapshot testing
	baseTime := time.Now().UTC()
	facts := []*db.Fact{
		{
			ID:        "snap-fact-1",
			Timestamp: baseTime,
			Namespace: "snap-ns",
			FieldName: "snap-field1",
			DataType:  db.DataTypeString,
			Value:     "initial-value",
			UserID:    "test-user",
		},
		{
			ID:        "snap-fact-1", // Same ID, updated version
			Timestamp: baseTime.Add(time.Minute),
			Namespace: "snap-ns",
			FieldName: "snap-field1",
			DataType:  db.DataTypeString,
			Value:     "updated-value",
			UserID:    "test-user",
		},
		{
			ID:        "snap-fact-2",
			Timestamp: baseTime.Add(2 * time.Minute),
			Namespace: "snap-ns",
			FieldName: "snap-field2",
			DataType:  db.DataTypeNumber,
			Value:     "100",
			UserID:    "test-user",
		},
		{
			ID:        "snap-fact-3",
			Timestamp: baseTime.Add(3 * time.Minute),
			Namespace: "other-snap-ns",
			FieldName: "snap-field3",
			DataType:  db.DataTypeBoolean,
			Value:     "true",
			UserID:    "test-user",
		},
		{
			ID:        "snap-fact-2", // Delete snap-fact-2
			Timestamp: baseTime.Add(4 * time.Minute),
			Namespace: "snap-ns",
			FieldName: "snap-field2",
			DataType:  db.DataTypeNumber,
			Value:     "100",
			UserID:    "test-user",
			IsDeleted: true,
		},
	}

	// Insert facts
	for _, fact := range facts {
		err := store.PutFact(ctx, fact)
		require.NoError(t, err, "PutFact should succeed")
	}

	// Test GetSnapshotAtTime
	t.Run("Snapshot at initial time", func(t *testing.T) {
		// Snapshot right after the first fact
		snapshotTime := baseTime.Add(30 * time.Second)
		snapshot, err := store.GetSnapshotAtTime(ctx, "snap-ns", snapshotTime)
		require.NoError(t, err, "GetSnapshotAtTime should succeed")
		assert.Len(t, snapshot, 1, "Snapshot should have 1 field")
		key := "snap-ns#snap-field1"
		fact, ok := snapshot[key]
		assert.True(t, ok, "snap-field1 should be in snapshot")
		assert.Equal(t, "initial-value", fact.Value, "Value should be initial-value")
	})

	t.Run("Snapshot after update", func(t *testing.T) {
		// Snapshot after field1 was updated
		snapshotTime := baseTime.Add(90 * time.Second)
		snapshot, err := store.GetSnapshotAtTime(ctx, "snap-ns", snapshotTime)
		require.NoError(t, err, "GetSnapshotAtTime should succeed")
		assert.Len(t, snapshot, 1, "Snapshot should have 1 field")
		key := "snap-ns#snap-field1"
		fact, ok := snapshot[key]
		assert.True(t, ok, "snap-field1 should be in snapshot")
		assert.Equal(t, "updated-value", fact.Value, "Value should be updated-value")
	})

	t.Run("Snapshot with multiple fields", func(t *testing.T) {
		// Snapshot after field2 was added
		snapshotTime := baseTime.Add(150 * time.Second)
		snapshot, err := store.GetSnapshotAtTime(ctx, "snap-ns", snapshotTime)
		require.NoError(t, err, "GetSnapshotAtTime should succeed")
		assert.Len(t, snapshot, 2, "Snapshot should have 2 fields")
		key1 := "snap-ns#snap-field1"
		fact1, ok := snapshot[key1]
		assert.True(t, ok, "snap-field1 should be in snapshot")
		assert.Equal(t, "updated-value", fact1.Value, "Value should be updated-value")
		
		key2 := "snap-ns#snap-field2"
		fact2, ok := snapshot[key2]
		assert.True(t, ok, "snap-field2 should be in snapshot")
		assert.Equal(t, "100", fact2.Value, "Value should be 100")
	})

	t.Run("Snapshot after deletion", func(t *testing.T) {
		// Snapshot after field2 was deleted
		snapshotTime := baseTime.Add(5 * time.Minute)
		snapshot, err := store.GetSnapshotAtTime(ctx, "snap-ns", snapshotTime)
		require.NoError(t, err, "GetSnapshotAtTime should succeed")
		assert.Len(t, snapshot, 1, "Snapshot should have 1 field")
		key1 := "snap-ns#snap-field1"
		fact1, ok := snapshot[key1]
		assert.True(t, ok, "snap-field1 should be in snapshot")
		
		key2 := "snap-ns#snap-field2"
		_, ok = snapshot[key2]
		assert.False(t, ok, "snap-field2 should not be in snapshot after deletion")
	})

	t.Run("Snapshot across all namespaces", func(t *testing.T) {
		// Snapshot across all namespaces
		snapshotTime := baseTime.Add(3 * time.Minute + 30 * time.Second)
		snapshot, err := store.GetSnapshotAtTime(ctx, "", snapshotTime)
		require.NoError(t, err, "GetSnapshotAtTime should succeed")
		assert.Len(t, snapshot, 3, "Snapshot should have 3 fields across all namespaces")
		
		// Check for field from other namespace
		key3 := "other-snap-ns#snap-field3"
		fact3, ok := snapshot[key3]
		assert.True(t, ok, "snap-field3 from other-snap-ns should be in snapshot")
		assert.Equal(t, "true", fact3.Value, "Value should be true")
	})
}

// TestMockStore verifies that the mock implementation satisfies the Store interface
func TestMockStore(t *testing.T) {
	store := db.NewMockStore()
	testStore(t, store)
}

// TestMockStoreExpectations tests the expectation functionality of MockStore
func TestMockStoreExpectations(t *testing.T) {
	store := db.NewMockStore()
	ctx := context.Background()
	
	// Set expectations
	store.ExpectCall("CreateTable", 1)
	store.ExpectCall("PutFact", 2)
	store.ExpectCall("GetFact", 1)
	
	// Meet expectations
	_ = store.CreateTable(ctx)
	_ = store.PutFact(ctx, &db.Fact{ID: "1", Namespace: "test", FieldName: "field"})
	_ = store.PutFact(ctx, &db.Fact{ID: "2", Namespace: "test", FieldName: "field"})
	_, _ = store.GetFact(ctx, "1")
	
	// Verify expectations
	err := store.VerifyExpectations()
	assert.NoError(t, err, "Expectations should be met")
}

// TestMockStoreFailureModes tests simulated failures in the mock store
func TestMockStoreFailureModes(t *testing.T) {
	store := db.NewMockStore()
	ctx := context.Background()
	
	// Create the table first
	err := store.CreateTable(ctx)
	require.NoError(t, err, "CreateTable should succeed")
	
	// Setup a simulated failure for PutFact
	expectedError := fmt.Errorf("simulated failure")
	store.SimulateFailure("PutFact", expectedError)
	
	// Attempt to put a fact, which should fail
	err = store.PutFact(ctx, &db.Fact{ID: "test", Namespace: "test", FieldName: "field"})
	require.Error(t, err, "PutFact should fail")
	
	// Check that it's our expected error
	assert.Contains(t, err.Error(), expectedError.Error(), "Error should contain our simulated failure")
}

// intcegrationTestEnabled returns true if DynamoDB integration tests should run
func integrationTestEnabled() bool {
	return os.Getenv("DYNAMODB_INTEGRATION_TEST") == "true"
}

// TestDynamoDBStore tests the DynamoDB implementation if integration tests are enabled
func TestDynamoDBStore(t *testing.T) {
	if !integrationTestEnabled() {
		t.Skip("Skipping DynamoDB integration test. Set DYNAMODB_INTEGRATION_TEST=true to run")
	}
	
	ctx := context.Background()
	
	// Set up DynamoDB client (use local or real AWS based on environment)
	opts := []func(*config.LoadOptions) error{}
	
	if ep := os.Getenv("DYNAMODB_ENDPOINT_URL"); ep != "" {
		resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{URL: ep, SigningRegion: os.Getenv("AWS_REGION")}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		opts = append(opts, config.WithEndpointResolver(resolver))
	}
	
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	require.NoError(t, err, "Loading AWS config should succeed")
	
	// Create a unique table name for this test run to avoid conflicts
	tableName := fmt.Sprintf("TestTable%d", time.Now().Unix())
	userID := "test-user"
	
	store := db.NewDynamoDBStore(&db.Config{
		TableName:    tableName,
		UserID:       userID,
		DynamoClient: dynamodb.NewFromConfig(cfg),
	})
	
	testStore(t, store)
}