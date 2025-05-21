package dynamotest_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elibdev/notably/testutil/dynamotest"
	"github.com/stretchr/testify/assert"
)

// mockClientFactory is a test helper that creates DynamoClient instances for tests
func mockClientFactory(cfg aws.Config, tableName, userID string) dynamotest.DynamoClient {
	return &mockDynamoClient{
		tableName: tableName,
		userID:    userID,
		facts:     make(map[string][]dynamotest.Fact),
	}
}

// mockDynamoClient implements the DynamoClient interface for testing
type mockDynamoClient struct {
	tableName string
	userID    string
	facts     map[string][]dynamotest.Fact // namespace#fieldName -> []Fact
}

func (m *mockDynamoClient) CreateTable(ctx context.Context) error {
	return nil
}

func (m *mockDynamoClient) PutFact(ctx context.Context, fact dynamotest.Fact) error {
	key := fact.Namespace + "#" + fact.FieldName
	m.facts[key] = append(m.facts[key], fact)
	return nil
}

func (m *mockDynamoClient) QueryByField(ctx context.Context, namespace, fieldName string, start, end time.Time) ([]dynamotest.Fact, error) {
	key := namespace + "#" + fieldName
	return m.facts[key], nil
}

func (m *mockDynamoClient) QueryByTimeRange(ctx context.Context, start, end time.Time) ([]dynamotest.Fact, error) {
	var results []dynamotest.Fact
	for _, facts := range m.facts {
		results = append(results, facts...)
	}
	return results, nil
}

// Test_Example_basicUsage demonstrates the simplest way to use the DynamoDB emulator
// with the WithDynamoTest helper function.
func Test_Example_basicUsage(t *testing.T) {
	// Skip this example in short mode or CI environments
	if testing.Short() {
		t.Skip("Skipping example in short mode")
	}

	// WithDynamoTest handles all setup and cleanup automatically
	dynamotest.WithDynamoTest(t, "example-user", mockClientFactory, func(ec *dynamotest.EmulatorClient) {
		ctx := context.Background()

		// Create a sample fact to store
		fact := dynamotest.Fact{
			ID:        "example-id",
			Timestamp: time.Now().UTC(),
			Namespace: ec.UserID,
			FieldName: "ExampleField",
			DataType:  "string",
			Value:     "Hello, DynamoDB!",
		}

		// Store the fact using the provided client
		err := ec.Client.PutFact(ctx, fact)
		assert.NoError(t, err, "Storing fact should succeed")

		// Query the fact back
		facts, err := ec.Client.QueryByField(
			ctx,
			ec.UserID,
			"ExampleField",
			time.Time{},
			time.Now().UTC().Add(time.Minute),
		)
		assert.NoError(t, err, "Querying fact should succeed")
		assert.NotEmpty(t, facts, "Should return the stored fact")

		if len(facts) > 0 {
			assert.Equal(t, fact.ID, facts[0].ID, "IDs should match")
			assert.Equal(t, fact.Value, facts[0].Value, "Values should match")
		}
	})
}

// Test_Example_manualSetup demonstrates how to manually create and manage the emulator client
// for more complex test scenarios.
func Test_Example_manualSetup(t *testing.T) {
	// Skip if emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	// Create custom configuration if needed
	config := dynamotest.NewEmulatorConfig()
	config.TableNamePrefix = "example-"

	// Create the emulator client
	ec, err := dynamotest.NewEmulatorClient(t, "manual-user", config, mockClientFactory)
	if err != nil {
		t.Fatalf("Failed to create emulator client: %v", err)
	}

	// Make sure to clean up when done
	defer ec.CleanUp()

	// Use the client for testing...
	ctx := context.Background()

	// Create a fact
	fact := dynamotest.Fact{
		ID:        "manual-example",
		Timestamp: time.Now().UTC(),
		Namespace: ec.UserID,
		FieldName: "ManualField",
		DataType:  "string",
		Value:     "Manual setup example",
	}

	err = ec.Client.PutFact(ctx, fact)
	assert.NoError(t, err, "Storing fact should succeed")
}

// Test_Example_multiUser demonstrates testing with multiple users accessing the same table
func Test_Example_multiUser(t *testing.T) {
	// Skip if emulator is not running
	dynamotest.SkipIfEmulatorNotRunning(t, nil)

	dynamotest.WithDynamoTest(t, "user1", mockClientFactory, func(ec *dynamotest.EmulatorClient) {
		// Create a client for a second user but same table
		user2Client := ec.CreateClientForUser("user2")

		ctx := context.Background()

		// Store facts for both users
		user1Fact := dynamotest.Fact{
			ID:        "fact1",
			Timestamp: time.Now().UTC(),
			Namespace: "user1",
			FieldName: "SharedField",
			DataType:  "string",
			Value:     "User 1's value",
		}

		user2Fact := dynamotest.Fact{
			ID:        "fact2",
			Timestamp: time.Now().UTC(),
			Namespace: "user2",
			FieldName: "SharedField",
			DataType:  "string",
			Value:     "User 2's value",
		}

		// Store both facts
		err := ec.Client.PutFact(ctx, user1Fact)
		assert.NoError(t, err, "Storing user1 fact should succeed")

		err = user2Client.PutFact(ctx, user2Fact)
		assert.NoError(t, err, "Storing user2 fact should succeed")

		// User 1 should only see their own data
		facts, err := ec.Client.QueryByField(ctx, "user1", "SharedField", time.Time{}, time.Now().UTC().Add(time.Minute))
		assert.NoError(t, err)
		assert.Len(t, facts, 1, "User1 should see only their own fact")

		// User 2 should only see their own data
		facts, err = user2Client.QueryByField(ctx, "user2", "SharedField", time.Time{}, time.Now().UTC().Add(time.Minute))
		assert.NoError(t, err)
		assert.Len(t, facts, 1, "User2 should see only their own fact")
	})
}

// Test_Example_oldStyleSetup demonstrates using the old-style TestSetup for backward compatibility
func Test_Example_oldStyleSetup(t *testing.T) {
	// Create a TestSetup (which now uses the emulator underneath)
	setup, err := dynamotest.NewTestSetup(t, mockClientFactory)
	if err != nil {
		t.Fatalf("Failed to create test setup: %v", err)
	}

	// Use cleanup helper
	defer dynamotest.CleanupTest(t, setup)

	// Use the client from the setup
	ctx := context.Background()
	fact := dynamotest.Fact{
		ID:        "backward-compat",
		Timestamp: time.Now().UTC(),
		Namespace: setup.UserID,
		FieldName: "LegacyField",
		DataType:  "string",
		Value:     "Using legacy setup",
	}

	err = setup.Client.PutFact(ctx, fact)
	assert.NoError(t, err, "Storing fact should succeed")
}
