package dynamotest

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// TestSetup contains all the configuration needed for testing
// This struct is maintained for backward compatibility
type TestSetup struct {
	Config    aws.Config
	TableName string
	UserID    string
	Client    DynamoClient
	t         *testing.T
}

// NewTestSetup creates a new test setup with a local DynamoDB emulator configuration
func NewTestSetup(t *testing.T, clientFactory NewClientFunc) (*TestSetup, error) {
	// Use the emulator client to set up the test
	emulatorClient, err := NewEmulatorClient(t, "test-user", nil, clientFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to create emulator client: %w", err)
	}

	// Return the TestSetup for backward compatibility
	return &TestSetup{
		Config:    emulatorClient.Config,
		TableName: emulatorClient.TableName,
		UserID:    emulatorClient.UserID,
		Client:    emulatorClient.Client,
		t:         t,
	}, nil
}

// CreateTestClient creates a new DynamoDB client for testing
// It uses the emulator and creates a unique table name for isolation
func CreateTestClient(t *testing.T, userID string, clientFactory NewClientFunc) (DynamoClient, string) {
	// Skip in short test mode
	if testing.Short() {
		t.Skip("Skipping DynamoDB emulator test in short mode")
	}

	// Create and return a client using the emulator
	emulatorClient, err := NewEmulatorClient(t, userID, nil, clientFactory)
	if err != nil {
		t.Fatalf("Failed to create emulator client: %v", err)
	}

	return emulatorClient.Client, emulatorClient.TableName
}

// CleanupTest is a helper to use in defer statements to clean up test resources
func CleanupTest(t *testing.T, setup *TestSetup) {
	t.Logf("Test cleanup: table %s would be deleted in production", setup.TableName)
	// In a real implementation, would delete the table here
}
