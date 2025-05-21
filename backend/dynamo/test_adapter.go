package dynamo

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/elibdev/notably/testutil/dynamotest"
)

// ClientAdapter adapts the dynamo.Client to implement the dynamotest.DynamoClient interface
type ClientAdapter struct {
	client *Client
}

// NewClientAdapter creates a new adapter that wraps a dynamo.Client
func NewClientAdapter(client *Client) *ClientAdapter {
	return &ClientAdapter{client: client}
}

// CreateTable implements dynamotest.DynamoClient
func (ca *ClientAdapter) CreateTable(ctx context.Context) error {
	return ca.client.CreateTable(ctx)
}

// PutFact implements dynamotest.DynamoClient
func (ca *ClientAdapter) PutFact(ctx context.Context, fact dynamotest.Fact) error {
	// Convert dynamotest.Fact to dynamo.Fact
	dynamoFact := Fact{
		ID:        fact.ID,
		Timestamp: fact.Timestamp,
		Namespace: fact.Namespace,
		FieldName: fact.FieldName,
		DataType:  fact.DataType,
		Value:     fact.Value,
	}

	// Convert columns if present
	if len(fact.Columns) > 0 {
		dynamoFact.Columns = make([]ColumnDefinition, len(fact.Columns))
		for i, col := range fact.Columns {
			dynamoFact.Columns[i] = ColumnDefinition{
				Name:     col.Name,
				DataType: col.DataType,
			}
		}
	}

	return ca.client.PutFact(ctx, dynamoFact)
}

// QueryByField implements dynamotest.DynamoClient
func (ca *ClientAdapter) QueryByField(ctx context.Context, namespace, fieldName string, start, end time.Time) ([]dynamotest.Fact, error) {
	facts, err := ca.client.QueryByField(ctx, namespace, fieldName, start, end)
	if err != nil {
		return nil, err
	}

	// Convert []dynamo.Fact to []dynamotest.Fact
	return convertFactsToTestFacts(facts), nil
}

// QueryByTimeRange implements dynamotest.DynamoClient
func (ca *ClientAdapter) QueryByTimeRange(ctx context.Context, start, end time.Time) ([]dynamotest.Fact, error) {
	facts, err := ca.client.QueryByTimeRange(ctx, start, end)
	if err != nil {
		return nil, err
	}

	// Convert []dynamo.Fact to []dynamotest.Fact
	return convertFactsToTestFacts(facts), nil
}

// Helper function to convert []dynamo.Fact to []dynamotest.Fact
func convertFactsToTestFacts(facts []Fact) []dynamotest.Fact {
	testFacts := make([]dynamotest.Fact, len(facts))
	for i, fact := range facts {
		testFact := dynamotest.Fact{
			ID:        fact.ID,
			Timestamp: fact.Timestamp,
			Namespace: fact.Namespace,
			FieldName: fact.FieldName,
			DataType:  fact.DataType,
			Value:     fact.Value,
		}

		// Convert columns if present
		if len(fact.Columns) > 0 {
			testFact.Columns = make([]dynamotest.ColumnDefinition, len(fact.Columns))
			for j, col := range fact.Columns {
				testFact.Columns[j] = dynamotest.ColumnDefinition{
					Name:     col.Name,
					DataType: col.DataType,
				}
			}
		}

		testFacts[i] = testFact
	}
	return testFacts
}

// ClientFactory creates a new dynamotest.DynamoClient
// This function can be passed to dynamotest functions
func ClientFactory(cfg aws.Config, tableName, userID string) dynamotest.DynamoClient {
	client := NewClient(cfg, tableName, userID)
	return NewClientAdapter(client)
}
