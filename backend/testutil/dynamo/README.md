# DynamoDB Testing Utilities

This package provides utilities for testing code that interacts with DynamoDB, focusing on using a local DynamoDB emulator instead of mocks.

## Overview

The `testutil/dynamo` package offers:

- `EmulatorClient`: A wrapper for working with a local DynamoDB emulator
- Helper functions for setting up and tearing down test databases
- Utilities for checking if the emulator is running

## Prerequisites

1. A running DynamoDB local emulator (default: http://localhost:8000)

### Setting up DynamoDB Local

The easiest way to run a local DynamoDB instance is using Docker:

```sh
docker run -p 8000:8000 amazon/dynamodb-local
```

## Usage

### Basic Usage with Helper Function

The simplest way to use the emulator is with the `WithDynamoTest` helper:

```go
func TestMyDynamoFunction(t *testing.T) {
    // This handles setup/teardown automatically
    dynamo.WithDynamoTest(t, "test-user", func(ec *dynamo.EmulatorClient) {
        // Use ec.Client for DynamoDB operations
        // ec.TableName contains the unique test table name
        // ec.UserID contains the user ID

        // Test your code here...
        err := ec.Client.PutFact(ctx, myFact)
        assert.NoError(t, err)
    })
}
```

### Manual Setup

For more control over the lifecycle:

```go
func TestAdvancedScenario(t *testing.T) {
    // Skip if emulator isn't running
    dynamo.SkipIfEmulatorNotRunning(t, nil)
    
    // Create custom config if needed
    config := dynamo.NewEmulatorConfig()
    config.TableNamePrefix = "mytest-"
    
    // Create the client
    ec, err := dynamo.NewEmulatorClient(t, "user123", config)
    if err != nil {
        t.Fatalf("Failed to create emulator client: %v", err)
    }
    
    // Clean up when done
    defer ec.CleanUp()
    
    // Run your tests...
}
```

### Creating Additional Clients

You can create additional clients for the same table but different users:

```go
// Create client for another user but same table
userClient := ec.CreateClientForUser("another-user")
```

## Backward Compatibility

For backward compatibility with existing code, you can use:

```go
// Old style setup, but uses the emulator underneath
setup, err := dynamo.NewTestSetup(t)
if err != nil {
    t.Fatalf("Failed to create test setup: %v", err)
}
```

## Configuration

The emulator can be configured using `EmulatorConfig`:

```go
config := &dynamo.EmulatorConfig{
    Endpoint:       "http://localhost:8000", // Your emulator endpoint
    Region:         "us-west-2",             // AWS region for the client
    TableNamePrefix: "myprefix-",            // Prefix for test tables
}
```

## Skipping Tests

To skip tests when the emulator isn't running:

```go
func TestRequiringEmulator(t *testing.T) {
    dynamo.SkipIfEmulatorNotRunning(t, nil)
    // Test code...
}
```

## Design Philosophy

This package favors using a real DynamoDB emulator over mocks because:

1. Tests run against the actual DynamoDB API
2. No need to mock complex DynamoDB behavior
3. Better test coverage of actual interactions
4. Closer to production behavior

The emulator approach strikes a balance between unit testing (fast, isolated) and integration testing (realistic, comprehensive).