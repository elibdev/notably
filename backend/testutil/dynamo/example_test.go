package dynamo

import (
	"context"
	"testing"
	"time"

	"github.com/elibdev/notably/dynamo"
	"github.com/stretchr/testify/assert"
)

// Example_basicUsage demonstrates the simplest way to use the DynamoDB emulator
// with the WithDynamoTest helper function.
func Example_basicUsage(t *testing.T) {
	// Skip this example in short mode or CI environments
	if testing.Short() {
		t.Skip("Skipping example in short mode")
	}

	// WithDynamoTest handles all setup and cleanup automatically
	WithDynamoTest(t, "example-user", func(ec *EmulatorClient) {
		ctx := context.Background()

		// Create a sample fact to store
		fact := dynamo.Fact{
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

// Example_manualSetup demonstrates how to manually create and manage the emulator client
// for more complex test scenarios.
func Example_manualSetup(t *testing.T) {
	// Skip if emulator is not running
	SkipIfEmulatorNotRunning(t, nil)

	// Create custom configuration if needed
	config := NewEmulatorConfig()
	config.TableNamePrefix = "example-"

	// Create the emulator client
	ec, err := NewEmulatorClient(t, "manual-user", config)
	if err != nil {
		t.Fatalf("Failed to create emulator client: %v", err)
	}

	// Make sure to clean up when done
	defer ec.CleanUp()

	// Use the client for testing...
	ctx := context.Background()

	// Create a fact
	fact := dynamo.Fact{
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

// Example_multiUser demonstrates testing with multiple users accessing the same table
func Example_multiUser(t *testing.T) {
	// Skip if emulator is not running
	SkipIfEmulatorNotRunning(t, nil)

	WithDynamoTest(t, "user1", func(ec *EmulatorClient) {
		// Create a client for a second user but same table
		user2Client := ec.CreateClientForUser("user2")

		ctx := context.Background()

		// Store facts for both users
		user1Fact := dynamo.Fact{
			ID:        "fact1",
			Timestamp: time.Now().UTC(),
			Namespace: "user1",
			FieldName: "SharedField",
			DataType:  "string",
			Value:     "User 1's value",
		}

		user2Fact := dynamo.Fact{
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

// Example_oldStyleSetup demonstrates using the old-style TestSetup for backward compatibility
func Example_oldStyleSetup(t *testing.T) {
	// Create a TestSetup (which now uses the emulator underneath)
	setup, err := NewTestSetup(t)
	if err != nil {
		t.Fatalf("Failed to create test setup: %v", err)
	}

	// Use cleanup helper
	defer CleanupTest(t, setup)

	// Use the client from the setup
	ctx := context.Background()
	fact := dynamo.Fact{
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
