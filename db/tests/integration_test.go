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

// TestIntegration_DynamoDBStore runs integration tests against a real or emulated DynamoDB
func TestIntegration_DynamoDBStore(t *testing.T) {
	// Skip unless explicitly enabled
	if os.Getenv("DYNAMODB_INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set DYNAMODB_INTEGRATION_TEST=true to run")
	}

	ctx := context.Background()
	store, tableName := setupIntegrationTest(t, ctx)
	defer cleanupIntegrationTest(t, ctx, store, tableName)

	t.Run("BasicCRUD", func(t *testing.T) {
		testIntegrationBasicCRUD(t, ctx, store)
	})

	t.Run("Querying", func(t *testing.T) {
		testIntegrationQuerying(t, ctx, store)
	})

	t.Run("HistoricalData", func(t *testing.T) {
		testIntegrationHistoricalData(t, ctx, store)
	})

	t.Run("Pagination", func(t *testing.T) {
		testIntegrationPagination(t, ctx, store)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testIntegrationErrorHandling(t, ctx, store)
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		testIntegrationConcurrentAccess(t, ctx, store)
	})
}

// setupIntegrationTest initializes resources for integration testing
func setupIntegrationTest(t *testing.T, ctx context.Context) (db.Store, string) {
	// Configure AWS client
	opts := []func(*config.LoadOptions) error{}

	// Use local DynamoDB if endpoint is specified
	if ep := os.Getenv("DYNAMODB_ENDPOINT_URL"); ep != "" {
		t.Logf("Using local DynamoDB endpoint: %s", ep)
		resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{URL: ep, SigningRegion: "us-west-2"}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		opts = append(opts, config.WithEndpointResolver(resolver))

		// Set dummy credentials for local testing
		os.Setenv("AWS_ACCESS_KEY_ID", "test")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
		os.Setenv("AWS_REGION", "us-west-2")
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	require.NoError(t, err, "Should load AWS config")

	// Create a unique table name for this test run
	timestamp := time.Now().UnixNano()
	tableName := fmt.Sprintf("IntegrationTest_%d", timestamp)
	userID := "test-user"

	store := db.NewDynamoDBStore(&db.Config{
		TableName:    tableName,
		UserID:       userID,
		DynamoClient: dynamodb.NewFromConfig(cfg),
	})

	// Create table
	err = store.CreateTable(ctx)
	require.NoError(t, err, "Should create table")

	return store, tableName
}

// cleanupIntegrationTest removes resources created during testing
func cleanupIntegrationTest(t *testing.T, ctx context.Context, store db.Store, tableName string) {
	// Delete the test table
	err := store.DeleteTable(ctx)
	if err != nil {
		t.Logf("Warning: Failed to delete test table %s: %v", tableName, err)
	}
}

// testIntegrationBasicCRUD tests basic CRUD operations
func testIntegrationBasicCRUD(t *testing.T, ctx context.Context, store db.Store) {
	// Create a test fact
	now := time.Now().UTC()
	factID := fmt.Sprintf("test-fact-%d", now.UnixNano())

	fact := &db.Fact{
		ID:        factID,
		Timestamp: now,
		Namespace: "test-ns",
		FieldName: "test-field",
		DataType:  db.DataTypeString,
		Value:     "initial-value",
		UserID:    "test-user",
	}

	// Create
	err := store.PutFact(ctx, fact)
	require.NoError(t, err, "Should put fact")

	// Read
	retrieved, err := store.GetFact(ctx, factID)
	require.NoError(t, err, "Should get fact")
	assert.Equal(t, fact.ID, retrieved.ID, "IDs should match")
	assert.Equal(t, fact.Value, retrieved.Value, "Values should match")

	// Update
	updatedFact := &db.Fact{
		ID:        factID,
		Timestamp: now.Add(time.Second),
		Namespace: "test-ns",
		FieldName: "test-field",
		DataType:  db.DataTypeString,
		Value:     "updated-value",
		UserID:    "test-user",
	}
	err = store.PutFact(ctx, updatedFact)
	require.NoError(t, err, "Should update fact")

	// Verify update
	retrievedUpdated, err := store.GetFact(ctx, factID)
	require.NoError(t, err, "Should get updated fact")
	assert.Equal(t, "updated-value", retrievedUpdated.Value, "Value should be updated")

	// Delete
	err = store.DeleteFact(ctx, factID)
	require.NoError(t, err, "Should delete fact")

	// Verify deletion - should have a tombstone marker
	tombstone, err := store.GetFact(ctx, factID)
	require.NoError(t, err, "Should get tombstone")
	assert.True(t, tombstone.IsDeleted, "Should be marked as deleted")
}

// testIntegrationQuerying tests query operations
func testIntegrationQuerying(t *testing.T, ctx context.Context, store db.Store) {
	// Create multiple facts to query
	now := time.Now().UTC()
	baseTime := now

	// Add facts at different times
	for i := 0; i < 5; i++ {
		fact := &db.Fact{
			ID:        fmt.Sprintf("query-fact-%d", i),
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Namespace: "query-ns",
			FieldName: fmt.Sprintf("field-%d", i%2), // alternate between field-0 and field-1
			DataType:  db.DataTypeString,
			Value:     fmt.Sprintf("value-%d", i),
			UserID:    "test-user",
		}
		err := store.PutFact(ctx, fact)
		require.NoError(t, err, "Should add fact for querying")
	}

	// Test QueryByField
	t.Run("QueryByField", func(t *testing.T) {
		startTime := baseTime.Add(-time.Minute)
		endTime := baseTime.Add(5 * time.Minute)

		result, err := store.QueryByField(ctx, "query-ns", "field-0", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			SortAscending: true,
		})

		require.NoError(t, err, "Should query by field")
		assert.Len(t, result.Facts, 3, "Should return 3 facts for field-0")

		// Verify sorting
		for i := 1; i < len(result.Facts); i++ {
			assert.True(t,
				result.Facts[i-1].Timestamp.Before(result.Facts[i].Timestamp) ||
					result.Facts[i-1].Timestamp.Equal(result.Facts[i].Timestamp),
				"Facts should be sorted by timestamp ascending")
		}
	})

	// Test QueryByNamespace
	t.Run("QueryByNamespace", func(t *testing.T) {
		startTime := baseTime.Add(-time.Minute)
		endTime := baseTime.Add(5 * time.Minute)

		result, err := store.QueryByNamespace(ctx, "query-ns", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			SortAscending: false, // descending order
		})

		require.NoError(t, err, "Should query by namespace")
		assert.Len(t, result.Facts, 5, "Should return all 5 facts in the namespace")

		// Verify sorting
		for i := 1; i < len(result.Facts); i++ {
			assert.True(t,
				result.Facts[i-1].Timestamp.After(result.Facts[i].Timestamp) ||
					result.Facts[i-1].Timestamp.Equal(result.Facts[i].Timestamp),
				"Facts should be sorted by timestamp descending")
		}
	})

	// Test QueryByTimeRange
	t.Run("QueryByTimeRange", func(t *testing.T) {
		// Query only facts in the middle of our time range
		startTime := baseTime.Add(1 * time.Minute)
		endTime := baseTime.Add(3 * time.Minute)

		result, err := store.QueryByTimeRange(ctx, db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			SortAscending: true,
		})

		require.NoError(t, err, "Should query by time range")
		assert.Len(t, result.Facts, 3, "Should return 3 facts in the time range")
	})
}

// testIntegrationHistoricalData tests time-based operations
func testIntegrationHistoricalData(t *testing.T, ctx context.Context, store db.Store) {
	// Create a series of updates to a single field
	now := time.Now().UTC()
	factID := fmt.Sprintf("history-fact-%d", now.UnixNano())

	// Initial state
	initialFact := &db.Fact{
		ID:        factID,
		Timestamp: now,
		Namespace: "history-ns",
		FieldName: "versioned-field",
		DataType:  db.DataTypeString,
		Value:     "version-1",
		UserID:    "test-user",
	}
	err := store.PutFact(ctx, initialFact)
	require.NoError(t, err, "Should add initial fact")

	// First update - 1 minute later
	update1 := &db.Fact{
		ID:        factID,
		Timestamp: now.Add(1 * time.Minute),
		Namespace: "history-ns",
		FieldName: "versioned-field",
		DataType:  db.DataTypeString,
		Value:     "version-2",
		UserID:    "test-user",
	}
	err = store.PutFact(ctx, update1)
	require.NoError(t, err, "Should add first update")

	// Second update - 2 minutes later
	update2 := &db.Fact{
		ID:        factID,
		Timestamp: now.Add(2 * time.Minute),
		Namespace: "history-ns",
		FieldName: "versioned-field",
		DataType:  db.DataTypeString,
		Value:     "version-3",
		UserID:    "test-user",
	}
	err = store.PutFact(ctx, update2)
	require.NoError(t, err, "Should add second update")

	// Test GetSnapshotAtTime at different points in time
	t.Run("Snapshot at start", func(t *testing.T) {
		snapshot, err := store.GetSnapshotAtTime(ctx, "history-ns", now.Add(30*time.Second))
		require.NoError(t, err, "Should get snapshot")

		fact, ok := snapshot["history-ns#versioned-field"]
		assert.True(t, ok, "Field should be in snapshot")
		assert.Equal(t, "version-1", fact.Value, "Should have initial version")
	})

	t.Run("Snapshot after first update", func(t *testing.T) {
		snapshot, err := store.GetSnapshotAtTime(ctx, "history-ns", now.Add(90*time.Second))
		require.NoError(t, err, "Should get snapshot")

		fact, ok := snapshot["history-ns#versioned-field"]
		assert.True(t, ok, "Field should be in snapshot")
		assert.Equal(t, "version-2", fact.Value, "Should have first update")
	})

	t.Run("Snapshot at latest state", func(t *testing.T) {
		snapshot, err := store.GetSnapshotAtTime(ctx, "history-ns", now.Add(3*time.Minute))
		require.NoError(t, err, "Should get snapshot")

		fact, ok := snapshot["history-ns#versioned-field"]
		assert.True(t, ok, "Field should be in snapshot")
		assert.Equal(t, "version-3", fact.Value, "Should have latest version")
	})

	// Test historical query
	t.Run("Historical query", func(t *testing.T) {
		startTime := now.Add(-time.Minute)
		endTime := now.Add(3 * time.Minute)

		result, err := store.QueryByField(ctx, "history-ns", "versioned-field", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			SortAscending: true,
		})

		require.NoError(t, err, "Should query versions")
		assert.Len(t, result.Facts, 3, "Should return all 3 versions")
		assert.Equal(t, "version-1", result.Facts[0].Value, "First should be version-1")
		assert.Equal(t, "version-2", result.Facts[1].Value, "Second should be version-2")
		assert.Equal(t, "version-3", result.Facts[2].Value, "Third should be version-3")
	})
}

// testIntegrationPagination tests pagination functionality
func testIntegrationPagination(t *testing.T, ctx context.Context, store db.Store) {
	// Insert many facts to test pagination
	now := time.Now().UTC()

	// Create 25 facts
	for i := 0; i < 25; i++ {
		fact := &db.Fact{
			ID:        fmt.Sprintf("page-fact-%d", i),
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Namespace: "pagination-ns",
			FieldName: "paginated-field",
			DataType:  db.DataTypeNumber,
			Value:     fmt.Sprintf("%d", i),
			UserID:    "test-user",
		}
		err := store.PutFact(ctx, fact)
		require.NoError(t, err, "Should add fact for pagination")
	}

	// Test pagination with small page size
	t.Run("Paginated query", func(t *testing.T) {
		startTime := now.Add(-time.Minute)
		endTime := now.Add(5 * time.Minute)
		limit := int32(10)

		// First page
		result1, err := store.QueryByField(ctx, "pagination-ns", "paginated-field", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			Limit:         &limit,
			SortAscending: true,
		})

		require.NoError(t, err, "Should query first page")
		assert.Len(t, result1.Facts, 10, "First page should have 10 facts")
		assert.NotNil(t, result1.NextToken, "Should have pagination token")

		// Second page
		result2, err := store.QueryByField(ctx, "pagination-ns", "paginated-field", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			Limit:         &limit,
			NextToken:     result1.NextToken,
			SortAscending: true,
		})

		require.NoError(t, err, "Should query second page")
		assert.Len(t, result2.Facts, 10, "Second page should have 10 facts")
		assert.NotNil(t, result2.NextToken, "Should have pagination token")

		// Third page (should have 5 remaining facts)
		result3, err := store.QueryByField(ctx, "pagination-ns", "paginated-field", db.QueryOptions{
			StartTime:     &startTime,
			EndTime:       &endTime,
			Limit:         &limit,
			NextToken:     result2.NextToken,
			SortAscending: true,
		})

		require.NoError(t, err, "Should query third page")
		assert.Len(t, result3.Facts, 5, "Third page should have 5 facts")
		assert.Nil(t, result3.NextToken, "Should not have another pagination token")

		// Verify we got all 25 facts with no duplicates
		allValues := make(map[string]bool)
		for _, fact := range result1.Facts {
			allValues[fact.Value] = true
		}
		for _, fact := range result2.Facts {
			allValues[fact.Value] = true
		}
		for _, fact := range result3.Facts {
			allValues[fact.Value] = true
		}

		assert.Len(t, allValues, 25, "Should have 25 unique values")
	})
}

// testIntegrationErrorHandling tests error cases
func testIntegrationErrorHandling(t *testing.T, ctx context.Context, store db.Store) {
	// Test not found error
	t.Run("Not found error", func(t *testing.T) {
		_, err := store.GetFact(ctx, "non-existent-fact-id")
		assert.Error(t, err, "Should return error for non-existent fact")
		assert.True(t, db.IsNotFound(err), "Error should be not found type")
	})

	// Test invalid fact
	t.Run("Invalid fact", func(t *testing.T) {
		invalidFact := &db.Fact{
			// Missing ID
			Timestamp: time.Now(),
			Namespace: "test-ns",
			FieldName: "test-field",
			Value:     "test-value",
		}

		err := store.PutFact(ctx, invalidFact)
		assert.Error(t, err, "Should reject fact without ID")
	})

	// Test nil fact
	t.Run("Nil fact", func(t *testing.T) {
		err := store.PutFact(ctx, nil)
		assert.Error(t, err, "Should reject nil fact")
	})
}

// testIntegrationConcurrentAccess tests concurrent operations
func testIntegrationConcurrentAccess(t *testing.T, ctx context.Context, store db.Store) {
	// Create a fact to update concurrently
	now := time.Now().UTC()
	factID := fmt.Sprintf("concurrent-fact-%d", now.UnixNano())

	baseFact := &db.Fact{
		ID:        factID,
		Timestamp: now,
		Namespace: "concurrent-ns",
		FieldName: "concurrent-field",
		DataType:  db.DataTypeNumber,
		Value:     "0",
		UserID:    "test-user",
	}

	err := store.PutFact(ctx, baseFact)
	require.NoError(t, err, "Should add initial fact")

	// Run concurrent updates
	const numGoroutines = 5
	const updatesPerGoroutine = 3

	done := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			baseTime := now.Add(time.Duration(goroutineID) * time.Millisecond)

			for i := 0; i < updatesPerGoroutine; i++ {
				updateFact := &db.Fact{
					ID:        factID,
					Timestamp: baseTime.Add(time.Duration(i) * time.Second),
					Namespace: "concurrent-ns",
					FieldName: "concurrent-field",
					DataType:  db.DataTypeNumber,
					Value:     fmt.Sprintf("g%d-u%d", goroutineID, i),
					UserID:    "test-user",
				}

				err := store.PutFact(ctx, updateFact)
				if err != nil {
					t.Logf("Concurrent update failed: %v", err)
				}
			}

			done <- true
		}(g)
	}

	// Wait for all goroutines to complete
	for g := 0; g < numGoroutines; g++ {
		<-done
	}

	// Query all versions of the fact
	startTime := now.Add(-time.Minute)
	endTime := now.Add(5 * time.Minute)

	result, err := store.QueryByField(ctx, "concurrent-ns", "concurrent-field", db.QueryOptions{
		StartTime:     &startTime,
		EndTime:       &endTime,
		SortAscending: true,
	})

	require.NoError(t, err, "Should query concurrent updates")

	// We should have the initial fact + all concurrent updates
	expectedCount := 1 + (numGoroutines * updatesPerGoroutine)
	assert.Len(t, result.Facts, expectedCount, "Should have expected number of versions")

	// Verify timestamps are in ascending order
	for i := 1; i < len(result.Facts); i++ {
		assert.True(t,
			result.Facts[i-1].Timestamp.Before(result.Facts[i].Timestamp) ||
				result.Facts[i-1].Timestamp.Equal(result.Facts[i].Timestamp),
			"Facts should be sorted by timestamp")
	}
}
