package dynamo

import (
	"context"
	"testing"
	"time"

	"github.com/elibdev/notably/testutil/dynamotest"
	"github.com/stretchr/testify/assert"
)

func TestColumnsStorage(t *testing.T) {
	// Skip if emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// Use the WithDynamoTest helper for clean setup/teardown
	dynamotest.WithDynamoTest(t, "test-user", ClientFactory, func(ec *dynamotest.EmulatorClient) {
		ctx := context.Background()

		// Test column definitions
		columns := []ColumnDefinition{
			{
				Name:     "name",
				DataType: "string",
			},
			{
				Name:     "age",
				DataType: "number",
			},
			{
				Name:     "active",
				DataType: "boolean",
			},
		}

		// Create a fact with columns
		tableFact := dynamotest.Fact{
			ID:        "test-table",
			Timestamp: time.Now().UTC(),
			Namespace: ec.UserID,
			FieldName: "TestTable",
			DataType:  "table",
			Value:     "",
			Columns:   make([]dynamotest.ColumnDefinition, len(columns)),
		}

		// Convert columns to dynamotest.ColumnDefinition
		for i, col := range columns {
			tableFact.Columns[i] = dynamotest.ColumnDefinition{
				Name:     col.Name,
				DataType: col.DataType,
			}
		}

		// Use the client from the emulator setup
		err := ec.Client.PutFact(ctx, tableFact)
		assert.NoError(t, err, "Should store fact with columns")

		// Query the fact back
		facts, err := ec.Client.QueryByField(ctx, ec.UserID, "TestTable", time.Time{}, time.Now().UTC().Add(time.Minute))
		assert.NoError(t, err, "Should query fact without error")
		assert.NotEmpty(t, facts, "Should return facts")

		// Verify columns are preserved
		if len(facts) > 0 {
			fact := facts[0]
			assert.Equal(t, tableFact.ID, fact.ID, "IDs should match")
			assert.Equal(t, tableFact.Namespace, fact.Namespace, "Namespaces should match")
			assert.Equal(t, tableFact.FieldName, fact.FieldName, "Field names should match")
			assert.Equal(t, tableFact.DataType, fact.DataType, "Data types should match")

			// Main test: verify column definitions are preserved
			assert.Equal(t, len(tableFact.Columns), len(fact.Columns), "Column count should match")
			for i, col := range fact.Columns {
				assert.Equal(t, tableFact.Columns[i].Name, col.Name, "Column name should match")
				assert.Equal(t, tableFact.Columns[i].DataType, col.DataType, "Column data type should match")
			}
		}
	})
}

// columnsForTest returns a set of column definitions for testing
func columnsForTest() []ColumnDefinition {
	return []ColumnDefinition{
		{Name: "title", DataType: "string"},
		{Name: "completed", DataType: "boolean"},
		{Name: "priority", DataType: "number"},
	}
}

func TestColumnsStorageWithRealEmulator(t *testing.T) {
	// Skip if emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// Create test setup using CreateTestClient
	client, _ := dynamotest.CreateTestClient(t, "test-user-2", ClientFactory)

	// Create a test context
	ctx := context.Background()

	// Test column definitions
	columns := columnsForTest()

	// Create a fact with columns directly using the client
	tableFact := dynamotest.Fact{
		ID:        "test-table-real",
		Timestamp: time.Now().UTC(),
		Namespace: "test-user-2",
		FieldName: "TestTable2",
		DataType:  "table",
		Value:     "test value",
		Columns:   make([]dynamotest.ColumnDefinition, len(columns)),
	}

	// Convert columns to dynamotest.ColumnDefinition
	for i, col := range columns {
		tableFact.Columns[i] = dynamotest.ColumnDefinition{
			Name:     col.Name,
			DataType: col.DataType,
		}
	}

	// Store the fact
	err := client.PutFact(ctx, tableFact)
	assert.NoError(t, err, "Should store fact with columns")

	// Query the fact back
	facts, err := client.QueryByField(ctx, "test-user-2", "TestTable2", time.Time{}, time.Now().UTC().Add(time.Minute))
	assert.NoError(t, err, "Should query fact without error")
	assert.NotEmpty(t, facts, "Should return facts")

	// Verify the fact and its columns were stored correctly
	if len(facts) > 0 {
		fact := facts[0]
		assert.Equal(t, tableFact.ID, fact.ID, "IDs should match")
		assert.Equal(t, tableFact.FieldName, fact.FieldName, "Field names should match")
		assert.Equal(t, len(tableFact.Columns), len(fact.Columns), "Column count should match")
	}
}
