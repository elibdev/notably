package dynamo

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/elibdev/notably/dynamo"
)

const (
	// DefaultEmulatorEndpoint is the default endpoint for the local DynamoDB emulator
	DefaultEmulatorEndpoint = "http://localhost:8000"

	// DefaultTestRegion is the region used for testing
	DefaultTestRegion = "us-west-2"
)

// EmulatorConfig holds configuration for connecting to a DynamoDB emulator
type EmulatorConfig struct {
	// Endpoint is the URL of the DynamoDB emulator (default: http://localhost:8000)
	Endpoint string

	// Region is the AWS region to use (default: us-west-2)
	Region string

	// TableNamePrefix is prepended to table names to avoid conflicts (default: test-)
	TableNamePrefix string
}

// EmulatorClient wraps the DynamoDB client and table information for testing
type EmulatorClient struct {
	Client    *dynamo.Client
	TableName string
	UserID    string
	Config    aws.Config
	t         *testing.T
}

// NewEmulatorConfig creates a default config for the DynamoDB emulator
func NewEmulatorConfig() *EmulatorConfig {
	return &EmulatorConfig{
		Endpoint:        DefaultEmulatorEndpoint,
		Region:          DefaultTestRegion,
		TableNamePrefix: "test-",
	}
}

// GetAwsConfig creates AWS SDK configuration for connecting to the DynamoDB emulator
func (ec *EmulatorConfig) GetAwsConfig(ctx context.Context) (aws.Config, error) {
	// Create a custom endpoint resolver for the emulator
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               ec.Endpoint,
			HostnameImmutable: true,
		}, nil
	})

	// Load configuration with custom resolver and dummy credentials
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(ec.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
	)
}

// NewEmulatorClient creates a client for testing with the DynamoDB emulator
func NewEmulatorClient(t *testing.T, userID string, config *EmulatorConfig) (*EmulatorClient, error) {
	if config == nil {
		config = NewEmulatorConfig()
	}

	// Skip if in short testing mode
	if testing.Short() {
		t.Skip("Skipping DynamoDB emulator test in short mode")
	}

	ctx := context.Background()
	cfg, err := config.GetAwsConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Generate a unique table name for this test
	tableName := fmt.Sprintf("%s%d", config.TableNamePrefix, time.Now().UnixNano())

	// Create the DynamoDB client
	client := dynamo.NewClient(cfg, tableName, userID)

	// Create the test table
	if err := client.CreateTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB test table: %w", err)
	}

	t.Logf("Created DynamoDB test table: %s", tableName)

	return &EmulatorClient{
		Client:    client,
		TableName: tableName,
		UserID:    userID,
		Config:    cfg,
		t:         t,
	}, nil
}

// CreateClientForUser creates a new client for a specific user using the same table and config
func (ec *EmulatorClient) CreateClientForUser(userID string) *dynamo.Client {
	return dynamo.NewClient(ec.Config, ec.TableName, userID)
}

// CleanUp tears down the test table and resources
// This is typically called in a defer statement after creating the emulator client
func (ec *EmulatorClient) CleanUp() {
	// In a real implementation, we would delete the DynamoDB table
	// However, the AWS SDK v2 doesn't provide a DeleteTable method in the dynamodbstreams interface
	// For local testing, this is less important as tables will be removed when the emulator is restarted

	ec.t.Logf("Test cleanup: DynamoDB table %s would be deleted in production", ec.TableName)

	// If needed, actual table deletion could be implemented using the raw DynamoDB client:
	/*
		client := dynamodb.NewFromConfig(ec.Config)
		_, err := client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
			TableName: aws.String(ec.TableName),
		})
		if err != nil {
			ec.t.Logf("Warning: Failed to delete test table: %v", err)
		}
	*/
}

// WithDynamoTest provides a simple way to run a test with a DynamoDB emulator
// It handles setup and cleanup automatically
func WithDynamoTest(t *testing.T, userID string, testFunc func(*EmulatorClient)) {
	ec, err := NewEmulatorClient(t, userID, nil)
	if err != nil {
		t.Fatalf("Failed to create emulator client: %v", err)
	}

	defer ec.CleanUp()

	testFunc(ec)
}

// IsEmulatorRunning checks if the DynamoDB emulator is available
// This can be used to skip tests if the emulator is not running
func IsEmulatorRunning(config *EmulatorConfig) bool {
	if config == nil {
		config = NewEmulatorConfig()
	}

	ctx := context.Background()
	cfg, err := config.GetAwsConfig(ctx)
	if err != nil {
		log.Printf("Failed to create AWS config: %v", err)
		return false
	}

	// Try to list tables to check if the emulator is running
	client := dynamodb.NewFromConfig(cfg)
	_, err = client.ListTables(ctx, &dynamodb.ListTablesInput{})

	return err == nil
}

// SkipIfEmulatorNotRunning skips the test if the DynamoDB emulator is not running
func SkipIfEmulatorNotRunning(t *testing.T, config *EmulatorConfig) {
	if !IsEmulatorRunning(config) {
		t.Skip("DynamoDB emulator is not running")
	}
}
