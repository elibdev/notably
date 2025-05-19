package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	defaultGSIName = "FieldIndex"
	pkName         = "UserID"
	skName         = "SK"
	fieldKeyName   = "FieldKey"
	isDeletedName  = "IsDeleted"
)

// DynamoDBStore implements the Store interface for AWS DynamoDB
type DynamoDBStore struct {
	db        *dynamodb.Client
	tableName string
	userID    string
}

// NewDynamoDBStore creates a new store using the provided DynamoDB client
func NewDynamoDBStore(cfg *Config) *DynamoDBStore {
	return &DynamoDBStore{
		db:        cfg.DynamoClient,
		tableName: cfg.TableName,
		userID:    cfg.UserID,
	}
}

// NewDynamoDBStoreFromEnv creates a new store with AWS config from environment
func NewDynamoDBStoreFromEnv(ctx context.Context, tableName, userID string) (*DynamoDBStore, error) {
	opts := []func(*config.LoadOptions) error{}
	if ep := getEndpointFromEnv(); ep != "" {
		resolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{URL: ep, SigningRegion: region}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})
		opts = append(opts, config.WithEndpointResolver(resolver))
	}
	
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, &StoreError{
			Operation: "NewDynamoDBStoreFromEnv",
			Err:       fmt.Errorf("loading AWS config: %w", err),
		}
	}
	
	return &DynamoDBStore{
		db:        dynamodb.NewFromConfig(cfg),
		tableName: tableName,
		userID:    userID,
	}, nil
}

// getEndpointFromEnv returns the DynamoDB endpoint from environment
func getEndpointFromEnv() string {
	return strings.TrimSpace(getEnv("DYNAMODB_ENDPOINT_URL", ""))
}

// getEnv gets env variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := getEnvFn(key); exists {
		return value
	}
	return fallback
}

// getEnvFn is a variable for easier testing
var getEnvFn = func(key string) (string, bool) {
	value, exists := strings.CutPrefix(key, "=")
	if exists {
		return value, true
	}
	return "", false
}

// CreateTable implements Store.CreateTable
func (s *DynamoDBStore) CreateTable(ctx context.Context) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(s.tableName),
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
	
	_, err := s.db.CreateTable(ctx, input)
	if err != nil {
		var existsErr *types.ResourceInUseException
		if !errors.As(err, &existsErr) {
			return &StoreError{
				Operation: "CreateTable",
				Err:       fmt.Errorf("create table failed: %w", err),
			}
		}
	}
	
	waiter := dynamodb.NewTableExistsWaiter(s.db)
	return waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(s.tableName)}, 5*time.Minute)
}

// DeleteTable implements Store.DeleteTable
func (s *DynamoDBStore) DeleteTable(ctx context.Context) error {
	_, err := s.db.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(s.tableName),
	})
	
	if err != nil {
		return &StoreError{
			Operation: "DeleteTable",
			Err:       fmt.Errorf("delete table failed: %w", err),
		}
	}
	
	waiter := dynamodb.NewTableNotExistsWaiter(s.db)
	return waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(s.tableName)}, 5*time.Minute)
}

// PutFact implements Store.PutFact
func (s *DynamoDBStore) PutFact(ctx context.Context, fact *Fact) error {
	if fact == nil {
		return &StoreError{
			Operation: "PutFact",
			Err:       errors.New("fact cannot be nil"),
		}
	}

	if fact.ID == "" {
		return &StoreError{
			Operation: "PutFact",
			Err:       errors.New("fact ID cannot be empty"),
		}
	}
	
	if fact.UserID == "" {
		fact.UserID = s.userID
	}
	
	// Create sort key in format timestamp#id
	sk := fmt.Sprintf("%s#%s", fact.Timestamp.Format(time.RFC3339Nano), fact.ID)
	// Create field key in format userId#namespace#fieldName
	fk := fmt.Sprintf("%s#%s#%s", s.userID, fact.Namespace, fact.FieldName)
	
	// Prepare item for DynamoDB
	item := map[string]types.AttributeValue{
		pkName:        &types.AttributeValueMemberS{Value: s.userID},
		skName:        &types.AttributeValueMemberS{Value: sk},
		"ID":          &types.AttributeValueMemberS{Value: fact.ID},
		"Namespace":   &types.AttributeValueMemberS{Value: fact.Namespace},
		"FieldName":   &types.AttributeValueMemberS{Value: fact.FieldName},
		"DataType":    &types.AttributeValueMemberS{Value: string(fact.DataType)},
		"Value":       &types.AttributeValueMemberS{Value: fact.Value},
		fieldKeyName:  &types.AttributeValueMemberS{Value: fk},
	}
	
	if fact.IsDeleted {
		item[isDeletedName] = &types.AttributeValueMemberBOOL{Value: true}
	}
	
	_, err := s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	
	if err != nil {
		return &StoreError{
			Operation: "PutFact",
			Err:       fmt.Errorf("put fact failed: %w", err),
		}
	}
	
	return nil
}

// GetFact implements Store.GetFact
func (s *DynamoDBStore) GetFact(ctx context.Context, id string) (*Fact, error) {
	// Query for the latest version of this fact ID
	result, err := s.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String(fmt.Sprintf("%s = :uid", pkName)),
		FilterExpression:       aws.String("ID = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: s.userID},
			":id":  &types.AttributeValueMemberS{Value: id},
		},
		ScanIndexForward: aws.Bool(false), // descending order by sort key
		Limit:            aws.Int32(1),    // just get the latest
	})
	
	if err != nil {
		return nil, &StoreError{
			Operation: "GetFact",
			Err:       fmt.Errorf("get fact failed: %w", err),
		}
	}
	
	if result.Count == 0 {
		return nil, &StoreError{
			Operation: "GetFact",
			Err:       &types.ResourceNotFoundException{Message: aws.String("fact not found")},
		}
	}
	
	facts, err := unmarshalFactItems(result.Items)
	if err != nil {
		return nil, &StoreError{
			Operation: "GetFact",
			Err:       fmt.Errorf("unmarshal failed: %w", err),
		}
	}
	
	return &facts[0], nil
}

// DeleteFact implements Store.DeleteFact
func (s *DynamoDBStore) DeleteFact(ctx context.Context, id string) error {
	// First get the latest version of the fact
	fact, err := s.GetFact(ctx, id)
	if err != nil {
		return &StoreError{
			Operation: "DeleteFact",
			Err:       fmt.Errorf("get fact failed: %w", err),
		}
	}
	
	// Set deletion marker
	fact.IsDeleted = true
	fact.Timestamp = time.Now()
	
	// Put the deletion marker
	return s.PutFact(ctx, fact)
}

// QueryByField implements Store.QueryByField
func (s *DynamoDBStore) QueryByField(ctx context.Context, namespace, fieldName string, opts QueryOptions) (*QueryResult, error) {
	// Create field key
	fk := fmt.Sprintf("%s#%s#%s", s.userID, namespace, fieldName)
	
	// Build query params
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String(defaultGSIName),
		KeyConditionExpression: aws.String(fmt.Sprintf("%s = :fk", fieldKeyName)),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":fk": &types.AttributeValueMemberS{Value: fk},
		},
		ScanIndexForward: aws.Bool(opts.SortAscending),
	}
	
	// Apply time range if provided
	if opts.StartTime != nil && opts.EndTime != nil {
		skStart := fmt.Sprintf("%s#", opts.StartTime.Format(time.RFC3339Nano))
		skEnd := fmt.Sprintf("%s#", opts.EndTime.Format(time.RFC3339Nano))
		
		queryInput.KeyConditionExpression = aws.String(
			fmt.Sprintf("%s = :fk AND %s BETWEEN :start AND :end", fieldKeyName, skName),
		)
		
		queryInput.ExpressionAttributeValues[":start"] = &types.AttributeValueMemberS{Value: skStart}
		queryInput.ExpressionAttributeValues[":end"] = &types.AttributeValueMemberS{Value: skEnd}
	}
	
	// Apply limit if provided
	if opts.Limit != nil {
		queryInput.Limit = opts.Limit
	}
	
	// Apply pagination token if provided
	if opts.NextToken != nil {
		var exclusiveStartKey map[string]types.AttributeValue
		if err := json.Unmarshal([]byte(*opts.NextToken), &exclusiveStartKey); err != nil {
			return nil, &StoreError{
				Operation: "QueryByField",
				Err:       fmt.Errorf("invalid next token: %w", err),
			}
		}
		queryInput.ExclusiveStartKey = exclusiveStartKey
	}
	
	// Execute query
	result, err := s.db.Query(ctx, queryInput)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByField",
			Err:       fmt.Errorf("query failed: %w", err),
		}
	}
	
	// Process results
	facts, err := unmarshalFactItems(result.Items)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByField",
			Err:       fmt.Errorf("unmarshal failed: %w", err),
		}
	}
	
	// Create pagination token if there's more data
	var nextToken *string
	if result.LastEvaluatedKey != nil {
		tokenBytes, err := json.Marshal(result.LastEvaluatedKey)
		if err != nil {
			return nil, &StoreError{
				Operation: "QueryByField",
				Err:       fmt.Errorf("marshal next token failed: %w", err),
			}
		}
		token := string(tokenBytes)
		nextToken = &token
	}
	
	return &QueryResult{
		Facts:     facts,
		NextToken: nextToken,
	}, nil
}

// QueryByTimeRange implements Store.QueryByTimeRange
func (s *DynamoDBStore) QueryByTimeRange(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	// Build query params
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String(fmt.Sprintf("%s = :uid", pkName)),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: s.userID},
		},
		ScanIndexForward: aws.Bool(opts.SortAscending),
	}
	
	// Apply time range if provided
	if opts.StartTime != nil && opts.EndTime != nil {
		skStart := fmt.Sprintf("%s#", opts.StartTime.Format(time.RFC3339Nano))
		skEnd := fmt.Sprintf("%s#", opts.EndTime.Format(time.RFC3339Nano))
		
		queryInput.KeyConditionExpression = aws.String(
			fmt.Sprintf("%s = :uid AND %s BETWEEN :start AND :end", pkName, skName),
		)
		
		queryInput.ExpressionAttributeValues[":start"] = &types.AttributeValueMemberS{Value: skStart}
		queryInput.ExpressionAttributeValues[":end"] = &types.AttributeValueMemberS{Value: skEnd}
	}
	
	// Apply limit if provided
	if opts.Limit != nil {
		queryInput.Limit = opts.Limit
	}
	
	// Apply pagination token if provided
	if opts.NextToken != nil {
		var exclusiveStartKey map[string]types.AttributeValue
		if err := json.Unmarshal([]byte(*opts.NextToken), &exclusiveStartKey); err != nil {
			return nil, &StoreError{
				Operation: "QueryByTimeRange",
				Err:       fmt.Errorf("invalid next token: %w", err),
			}
		}
		queryInput.ExclusiveStartKey = exclusiveStartKey
	}
	
	// Execute query
	result, err := s.db.Query(ctx, queryInput)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByTimeRange",
			Err:       fmt.Errorf("query failed: %w", err),
		}
	}
	
	// Process results
	facts, err := unmarshalFactItems(result.Items)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByTimeRange",
			Err:       fmt.Errorf("unmarshal failed: %w", err),
		}
	}
	
	// Create pagination token if there's more data
	var nextToken *string
	if result.LastEvaluatedKey != nil {
		tokenBytes, err := json.Marshal(result.LastEvaluatedKey)
		if err != nil {
			return nil, &StoreError{
				Operation: "QueryByTimeRange",
				Err:       fmt.Errorf("marshal next token failed: %w", err),
			}
		}
		token := string(tokenBytes)
		nextToken = &token
	}
	
	return &QueryResult{
		Facts:     facts,
		NextToken: nextToken,
	}, nil
}

// QueryByNamespace implements Store.QueryByNamespace
func (s *DynamoDBStore) QueryByNamespace(ctx context.Context, namespace string, opts QueryOptions) (*QueryResult, error) {
	// Build query params for full scan with filter
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String(fmt.Sprintf("%s = :uid", pkName)),
		FilterExpression:       aws.String("Namespace = :ns"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: s.userID},
			":ns":  &types.AttributeValueMemberS{Value: namespace},
		},
		ScanIndexForward: aws.Bool(opts.SortAscending),
	}
	
	// Apply time range if provided
	if opts.StartTime != nil && opts.EndTime != nil {
		skStart := fmt.Sprintf("%s#", opts.StartTime.Format(time.RFC3339Nano))
		skEnd := fmt.Sprintf("%s#", opts.EndTime.Format(time.RFC3339Nano))
		
		queryInput.KeyConditionExpression = aws.String(
			fmt.Sprintf("%s = :uid AND %s BETWEEN :start AND :end", pkName, skName),
		)
		
		queryInput.ExpressionAttributeValues[":start"] = &types.AttributeValueMemberS{Value: skStart}
		queryInput.ExpressionAttributeValues[":end"] = &types.AttributeValueMemberS{Value: skEnd}
	}
	
	// Apply limit if provided
	if opts.Limit != nil {
		queryInput.Limit = opts.Limit
	}
	
	// Apply pagination token if provided
	if opts.NextToken != nil {
		var exclusiveStartKey map[string]types.AttributeValue
		if err := json.Unmarshal([]byte(*opts.NextToken), &exclusiveStartKey); err != nil {
			return nil, &StoreError{
				Operation: "QueryByNamespace",
				Err:       fmt.Errorf("invalid next token: %w", err),
			}
		}
		queryInput.ExclusiveStartKey = exclusiveStartKey
	}
	
	// Execute query
	result, err := s.db.Query(ctx, queryInput)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByNamespace",
			Err:       fmt.Errorf("query failed: %w", err),
		}
	}
	
	// Process results
	facts, err := unmarshalFactItems(result.Items)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByNamespace",
			Err:       fmt.Errorf("unmarshal failed: %w", err),
		}
	}
	
	// Create pagination token if there's more data
	var nextToken *string
	if result.LastEvaluatedKey != nil {
		tokenBytes, err := json.Marshal(result.LastEvaluatedKey)
		if err != nil {
			return nil, &StoreError{
				Operation: "QueryByNamespace",
				Err:       fmt.Errorf("marshal next token failed: %w", err),
			}
		}
		token := string(tokenBytes)
		nextToken = &token
	}
	
	return &QueryResult{
		Facts:     facts,
		NextToken: nextToken,
	}, nil
}

// GetSnapshotAtTime implements Store.GetSnapshotAtTime
func (s *DynamoDBStore) GetSnapshotAtTime(ctx context.Context, namespace string, at time.Time) (map[string]Fact, error) {
	// Query all facts in the namespace up to the given time
	queryOpts := QueryOptions{
		StartTime:     &time.Time{}, // UNIX epoch
		EndTime:       &at,          // Up to the specified time
		SortAscending: false,        // Get newest first for each field
	}
	
	var result *QueryResult
	var err error
	
	if namespace == "" {
		// Query all facts if no namespace specified
		result, err = s.QueryByTimeRange(ctx, queryOpts)
	} else {
		// Query only the specified namespace
		result, err = s.QueryByNamespace(ctx, namespace, queryOpts)
	}
	
	if err != nil {
		return nil, &StoreError{
			Operation: "GetSnapshotAtTime",
			Err:       fmt.Errorf("query failed: %w", err),
		}
	}
	
	// Build snapshot map - most recent fact for each field
	snapshot := make(map[string]Fact)
	for _, fact := range result.Facts {
		// We identify fields by namespace#fieldName
		key := fmt.Sprintf("%s#%s", fact.Namespace, fact.FieldName)
		
		// Check if we already have this field in our snapshot
		existingFact, exists := snapshot[key]
		
		// If we don't have it yet, or this version is newer, use this one
		if !exists || fact.Timestamp.After(existingFact.Timestamp) {
			// Skip deleted items
			if !fact.IsDeleted {
				snapshot[key] = fact
			} else if exists {
				// Remove the fact if it was deleted and we had it in the snapshot
				delete(snapshot, key)
			}
		}
	}
	
	return snapshot, nil
}

// unmarshalFactItems converts DynamoDB items to Fact structs
func unmarshalFactItems(items []map[string]types.AttributeValue) ([]Fact, error) {
	facts := make([]Fact, 0, len(items))
	
	for _, item := range items {
		fact := Fact{}
		
		// Extract standard fields
		if v, ok := item["ID"]; ok {
			if sv, ok := v.(*types.AttributeValueMemberS); ok {
				fact.ID = sv.Value
			}
		}
		
		if v, ok := item["Namespace"]; ok {
			if sv, ok := v.(*types.AttributeValueMemberS); ok {
				fact.Namespace = sv.Value
			}
		}
		
		if v, ok := item["FieldName"]; ok {
			if sv, ok := v.(*types.AttributeValueMemberS); ok {
				fact.FieldName = sv.Value
			}
		}
		
		if v, ok := item["DataType"]; ok {
			if sv, ok := v.(*types.AttributeValueMemberS); ok {
				fact.DataType = DataType(sv.Value)
			}
		}
		
		if v, ok := item["Value"]; ok {
			if sv, ok := v.(*types.AttributeValueMemberS); ok {
				fact.Value = sv.Value
			}
		}
		
		if v, ok := item[pkName]; ok {
			if sv, ok := v.(*types.AttributeValueMemberS); ok {
				fact.UserID = sv.Value
			}
		}
		
		if v, ok := item[isDeletedName]; ok {
			if bv, ok := v.(*types.AttributeValueMemberBOOL); ok {
				fact.IsDeleted = bv.Value
			}
		}
		
		// Extract timestamp from SK
		if v, ok := item[skName]; ok {
			if sv, ok := v.(*types.AttributeValueMemberS); ok {
				parts := strings.SplitN(sv.Value, "#", 2)
				if len(parts) > 0 {
					ts, err := time.Parse(time.RFC3339Nano, parts[0])
					if err != nil {
						return nil, fmt.Errorf("parse timestamp failed: %w", err)
					}
					fact.Timestamp = ts
				}
			}
		}
		
		facts = append(facts, fact)
	}
	
	return facts, nil
}