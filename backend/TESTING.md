# Testing Guidelines for Notably Backend

This document explains best practices for testing the Notably backend, with special emphasis on DynamoDB testing.

## Overview

The Notably backend uses Go's standard testing package along with testify assertions. Tests are organized alongside the code they test, following Go conventions.

## DynamoDB Testing

### Emulator vs. Mocks

We've moved away from using mocks for DynamoDB testing in favor of using a local DynamoDB emulator. This provides several benefits:

1. **More realistic testing**: Tests interact with a real DynamoDB API implementation
2. **Reduced maintenance**: No need to keep mock implementations in sync with API changes
3. **Better coverage**: Catches issues with query formation, marshaling/unmarshaling, etc.
4. **Simpler tests**: Focus on testing your logic, not reimplementing DynamoDB behavior

### Setting Up The DynamoDB Emulator

The easiest way to run a local DynamoDB emulator is using Docker:

```sh
docker run -p 8000:8000 amazon/dynamodb-local
```

This starts a local DynamoDB instance on port 8000.

### Using the TestUtil Package

We've created comprehensive utilities in the `testutil/dynamotest` package to simplify testing with DynamoDB:

#### Option 1: Using WithDynamoTest Helper (Recommended)

```go
func TestMyFunction(t *testing.T) {
    dynamotest.WithDynamoTest(t, "test-user", dynamo.ClientFactory, func(ec *dynamotest.EmulatorClient) {
        // ec.Client is a fully configured dynamo.Client
        // ec.TableName contains the unique test table name
        // ec.UserID contains the user ID
        
        // Your test code here
    })
    // Table cleanup happens automatically
}
```

#### Option 2: Manual Setup

```go
func TestAdvancedScenario(t *testing.T) {
    dynamotest.SkipIfEmulatorNotRunning(t, nil)
    
    ec, err := dynamotest.NewEmulatorClient(t, "user123", nil, dynamo.ClientFactory)
    if err != nil {
        t.Fatalf("Failed to create emulator client: %v", err)
    }
    
    defer ec.CleanUp()
    
    // Test using ec.Client
}
```

#### Backward Compatibility

For existing tests using the old pattern:

```go
setup, err := dynamotest.NewTestSetup(t, dynamo.ClientFactory)
if err != nil {
    t.Fatalf("Failed to create test setup: %v", err)
}

// Use setup.Client as before
```

This still works but uses the emulator behind the scenes.

### Best Practices

1. **Skip if necessary**: Use `dynamotest.SkipIfEmulatorNotRunning(t, nil)` to skip tests when the emulator isn't running
2. **Use unique table names**: The utilities automatically create unique table names to avoid conflicts
3. **Clean up**: Always clean up resources using `defer ec.CleanUp()` or by using the `WithDynamoTest` helper
4. **Isolation**: Create new clients for different users with `ec.CreateClientForUser("another-user")`

### CI/CD Integration

For CI/CD pipelines, you have two options:

1. **Run with emulator**: Start the DynamoDB emulator in the CI environment
2. **Skip DynamoDB tests**: Run tests with `-short` flag to skip emulator tests

## General Testing Tips

1. **Keep tests fast**: Avoid unnecessary setup/teardown
2. **Test independence**: Each test should be self-contained
3. **Clear assertions**: Use descriptive assertion messages
4. **Clean up after tests**: Particularly important for integration tests

## Adapter for dynamo.Client

To solve circular dependency issues, we've created an adapter in `dynamo/test_adapter.go` that allows `dynamo.Client` to implement the `dynamotest.DynamoClient` interface:

```go
// Use the ClientFactory function in your tests
dynamotest.WithDynamoTest(t, "test-user", dynamo.ClientFactory, func(ec *dynamotest.EmulatorClient) {
    // Testing code here...
})
```

The adapter handles conversion between the package-specific `Fact` types, allowing seamless testing without circular imports.

## Further Reading

- See `testutil/dynamotest/README.md` for detailed documentation on the DynamoDB testing utilities
- Check `testutil/dynamotest/example_test.go` for more examples