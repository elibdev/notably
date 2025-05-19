package dynamo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	defaultGSIName = "FieldIndex"
	pkName         = "UserID"
	skName         = "SK"
	fieldKeyName   = "FieldKey"
)

// Fact represents a single versioned value for a field.
type Fact struct {
	ID        string
	Timestamp time.Time
	Namespace string
	FieldName string
	DataType  string
	Value     interface{}
}

// Client wraps DynamoDB operations for facts storage.
type Client struct {
	db        *dynamodb.Client
	tableName string
	userID    string
}

// NewClient creates a new Client for the given AWS config, table name, and user ID.
func NewClient(cfg aws.Config, tableName, userID string) *Client {
	return &Client{
		db:        dynamodb.NewFromConfig(cfg),
		tableName: tableName,
		userID:    userID,
	}
}

// CreateTable creates the DynamoDB table and the FieldIndex GSI.
func (c *Client) CreateTable(ctx context.Context) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(c.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String(pkName), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(skName), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(fieldKeyName), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(pkName), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String(skName), KeyType: types.KeyTypeRange},
		},
		BillingMode: types.BillingModePayPerRequest,
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(defaultGSIName),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String(fieldKeyName), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String(skName), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
	}
	_, err := c.db.CreateTable(ctx, input)
	if err != nil {
		var existsErr *types.ResourceInUseException
		if !errors.As(err, &existsErr) {
			return fmt.Errorf("create table: %w", err)
		}
	}
	waiter := dynamodb.NewTableExistsWaiter(c.db)
	return waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(c.tableName)}, 5*time.Minute)
}

// PutFact writes a Fact to DynamoDB.
func (c *Client) PutFact(ctx context.Context, fact Fact) error {
	sk := fmt.Sprintf("%s#%s", fact.Timestamp.Format(time.RFC3339Nano), fact.ID)
	fk := fmt.Sprintf("%s#%s#%s", c.userID, fact.Namespace, fact.FieldName)
	item := map[string]types.AttributeValue{
		pkName:       &types.AttributeValueMemberS{Value: c.userID},
		skName:       &types.AttributeValueMemberS{Value: sk},
		"Namespace":  &types.AttributeValueMemberS{Value: fact.Namespace},
		"FieldName":  &types.AttributeValueMemberS{Value: fact.FieldName},
		"DataType":   &types.AttributeValueMemberS{Value: fact.DataType},
		fieldKeyName: &types.AttributeValueMemberS{Value: fk},
	}
	av, err := attributevalue.Marshal(fact.Value)
	if err != nil {
		return err
	}
	item["Value"] = av
	_, err = c.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.tableName),
		Item:      item,
	})
	return err
}

// QueryByField returns all facts in a namespace/fieldName for the user in the time range [start, end].
func (c *Client) QueryByField(ctx context.Context, namespace, fieldName string, start, end time.Time) ([]Fact, error) {
	// Ensure start and end times are valid
	if start.IsZero() {
		start = time.Unix(0, 0) // Use Unix epoch as default start
	}
	if end.IsZero() {
		end = time.Now().UTC() // Use current time as default end
	}

	// Avoid potential timestamp formatting issues
	if start.After(end) {
		return nil, fmt.Errorf("invalid time range: start time (%v) is after end time (%v)", start, end)
	}

	fk := fmt.Sprintf("%s#%s#%s", c.userID, namespace, fieldName)
	skStart := fmt.Sprintf("%s#", start.Format(time.RFC3339Nano))
	skEnd := fmt.Sprintf("%s#", end.Format(time.RFC3339Nano))

	// Build query with required key conditions
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(c.tableName),
		IndexName:              aws.String(defaultGSIName),
		KeyConditionExpression: aws.String(fmt.Sprintf("%s = :fk AND %s BETWEEN :start AND :end", fieldKeyName, skName)),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":fk":    &types.AttributeValueMemberS{Value: fk},
			":start": &types.AttributeValueMemberS{Value: skStart},
			":end":   &types.AttributeValueMemberS{Value: skEnd},
		},
	}

	// Execute the query
	out, err := c.db.Query(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("DynamoDB query failed for field %s.%s in time range [%v, %v]: %w",
			namespace, fieldName, start, end, err)
	}

	return unmarshalFacts(out.Items)
}

// QueryByTimeRange returns all facts for the user in the time range [start, end].
func (c *Client) QueryByTimeRange(ctx context.Context, start, end time.Time) ([]Fact, error) {
	// Ensure start and end times are valid
	if start.IsZero() {
		start = time.Unix(0, 0) // Use Unix epoch as default start
	}
	if end.IsZero() {
		end = time.Now().UTC() // Use current time as default end
	}

	// Avoid potential timestamp formatting issues
	if start.After(end) {
		return nil, fmt.Errorf("invalid time range: start time (%v) is after end time (%v)", start, end)
	}

	skStart := fmt.Sprintf("%s#", start.Format(time.RFC3339Nano))
	skEnd := fmt.Sprintf("%s#", end.Format(time.RFC3339Nano))

	// Build query with required key conditions
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(c.tableName),
		KeyConditionExpression: aws.String(fmt.Sprintf("%s = :uid AND %s BETWEEN :start AND :end", pkName, skName)),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid":   &types.AttributeValueMemberS{Value: c.userID},
			":start": &types.AttributeValueMemberS{Value: skStart},
			":end":   &types.AttributeValueMemberS{Value: skEnd},
		},
	}

	// Execute the query
	out, err := c.db.Query(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("DynamoDB query failed for user %s in time range [%v, %v]: %w",
			c.userID, start, end, err)
	}

	return unmarshalFacts(out.Items)
}

func unmarshalFacts(items []map[string]types.AttributeValue) ([]Fact, error) {
	facts := make([]Fact, 0, len(items))
	for _, item := range items {
		var raw struct {
			SK        string      `dynamodbav:"SK"`
			Namespace string      `dynamodbav:"Namespace"`
			FieldName string      `dynamodbav:"FieldName"`
			DataType  string      `dynamodbav:"DataType"`
			Value     interface{} `dynamodbav:"Value"`
		}
		if err := attributevalue.UnmarshalMap(item, &raw); err != nil {
			return nil, err
		}
		parts := strings.SplitN(raw.SK, "#", 2)
		ts, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return nil, err
		}
		var id string
		if len(parts) > 1 {
			id = parts[1]
		}
		facts = append(facts, Fact{
			ID:        id,
			Timestamp: ts,
			Namespace: raw.Namespace,
			FieldName: raw.FieldName,
			DataType:  raw.DataType,
			Value:     raw.Value,
		})
	}
	return facts, nil
}
