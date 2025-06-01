package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/elibdev/notably/internal/models"
)

// DynamoUserManager implements UserManager interface using DynamoDB
type DynamoUserManager struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoUserManager creates a new DynamoDB-based UserManager
func NewDynamoUserManager(client *dynamodb.Client, tableName string) UserManager {
	return &DynamoUserManager{
		client:    client,
		tableName: tableName,
	}
}

// GetUserRepository returns a UserRepository for a specific user
func (m *DynamoUserManager) GetUserRepository(userID string) UserRepository {
	return &DynamoUserRepository{
		client:    m.client,
		tableName: m.tableName,
		userID:    userID,
	}
}

// ValidateUserAccess checks if a user has access to a table
func (m *DynamoUserManager) ValidateUserAccess(ctx context.Context, userID string, tableID string) error {
	repo := m.GetUserRepository(userID)
	_, err := repo.GetTable(ctx, tableID)
	if err != nil {
		if IsNotFound(err) {
			return NewRepositoryError(ErrorTypeUnauthorized, "user does not have access to table", err)
		}
		return err
	}
	return nil
}

// CreateUser creates a new user entry
func (m *DynamoUserManager) CreateUser(ctx context.Context, userID string) (*UserStats, error) {
	now := time.Now()
	item := map[string]types.AttributeValue{
		"PK":        &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
		"SK":        &types.AttributeValueMemberS{Value: "METADATA"},
		"UserID":    &types.AttributeValueMemberS{Value: userID},
		"CreatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"Type":      &types.AttributeValueMemberS{Value: "USER"},
	}

	_, err := m.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(m.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})

	if err != nil {
		// Check if this is a conditional check failure (user already exists)
		var conditionalCheckErr *types.ConditionalCheckFailedException
		if errors.As(err, &conditionalCheckErr) {
			return nil, NewRepositoryError(ErrorTypeAlreadyExists, "user already exists", err)
		}
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to create user", err)
	}

	// Return user stats
	stats := &UserStats{
		UserID:      userID,
		TableCount:  0,
		EntityCount: 0,
		LastActive:  now,
		CreatedAt:   now,
	}

	return stats, nil
}

// GetUser retrieves user information
func (m *DynamoUserManager) GetUser(ctx context.Context, userID string) (*UserStats, error) {
	return m.GetUserStats(ctx, userID)
}

// Health checks the health of the user manager
func (m *DynamoUserManager) Health(ctx context.Context) error {
	_, err := m.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(m.tableName),
	})

	if err != nil {
		// Check if table doesn't exist
		var resourceNotFound *types.ResourceNotFoundException
		if errors.As(err, &resourceNotFound) {
			// Try to create the table
			return m.createTableIfNotExists(ctx)
		}
		return err
	}

	return nil
}

// createTableIfNotExists creates the DynamoDB table with the required schema
func (m *DynamoUserManager) createTableIfNotExists(ctx context.Context) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(m.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("GSI1PK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("GSI1SK"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("PK"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("SK"),
				KeyType:       types.KeyTypeRange,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GSI1"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("GSI1PK"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("GSI1SK"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := m.client.CreateTable(ctx, input)
	if err != nil {
		var resourceInUse *types.ResourceInUseException
		if errors.As(err, &resourceInUse) {
			// Table already exists, which is fine
			return nil
		}
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Wait for table to be active
	return m.waitForTableActive(ctx)
}

// waitForTableActive waits for the DynamoDB table to be in ACTIVE state
func (m *DynamoUserManager) waitForTableActive(ctx context.Context) error {
	waiter := dynamodb.NewTableExistsWaiter(m.client)
	return waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(m.tableName),
	}, 2*time.Minute)
}

// DeleteUser removes a user and all their data
func (m *DynamoUserManager) DeleteUser(ctx context.Context, userID string) error {
	// This would require scanning all user data and deleting it
	// For now, just mark the user as deleted
	now := time.Now()

	_, err := m.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(m.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET DeletedAt = :deletedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":deletedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
	})

	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to delete user", err)
	}

	return nil
}

// GetUserStats retrieves user statistics
func (m *DynamoUserManager) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	// Query for user metadata
	result, err := m.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(m.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to get user stats", err)
	}

	if result.Item == nil {
		return nil, NewRepositoryError(ErrorTypeNotFound, "user not found", nil)
	}

	// Parse created_at
	createdAtStr := result.Item["CreatedAt"].(*types.AttributeValueMemberS).Value
	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)

	// TODO: Query for actual table and entity counts
	stats := &UserStats{
		UserID:      userID,
		TableCount:  0,
		EntityCount: 0,
		LastActive:  time.Now(),
		CreatedAt:   createdAt,
	}

	return stats, nil
}

// DynamoUserRepository implements UserRepository interface using DynamoDB
type DynamoUserRepository struct {
	client    *dynamodb.Client
	tableName string
	userID    string
}

// Table operations

// CreateTable creates a new table schema
func (r *DynamoUserRepository) CreateTable(ctx context.Context, tableID string, fields []models.FieldDefinition) (*models.TableSchema, error) {
	now := time.Now()
	table := &models.TableSchema{
		ID:        tableID,
		UserID:    r.userID,
		Fields:    fields,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate fields
	if err := table.ValidateFields(); err != nil {
		return nil, NewRepositoryError(ErrorTypeInvalidInput, "invalid table fields", err)
	}

	// Serialize fields
	fieldsBytes, err := json.Marshal(fields)
	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to serialize fields", err)
	}

	item := map[string]types.AttributeValue{
		"PK":        &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", r.userID)},
		"SK":        &types.AttributeValueMemberS{Value: fmt.Sprintf("TABLE#%s", tableID)},
		"TableID":   &types.AttributeValueMemberS{Value: tableID},
		"UserID":    &types.AttributeValueMemberS{Value: r.userID},
		"Fields":    &types.AttributeValueMemberS{Value: string(fieldsBytes)},
		"CreatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"UpdatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		"Type":      &types.AttributeValueMemberS{Value: "TABLE"},
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeAlreadyExists, "table already exists", err)
	}

	return table, nil
}

// GetTable retrieves a table schema
func (r *DynamoUserRepository) GetTable(ctx context.Context, tableID string) (*models.TableSchema, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", r.userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TABLE#%s", tableID)},
		},
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to get table", err)
	}

	if result.Item == nil {
		return nil, NewRepositoryError(ErrorTypeNotFound, "table not found", nil)
	}

	return r.parseTableFromItem(result.Item)
}

// ListTables retrieves all tables for a user
func (r *DynamoUserRepository) ListTables(ctx context.Context) ([]models.TableSchema, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", r.userID)},
			":sk": &types.AttributeValueMemberS{Value: "TABLE#"},
		},
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to list tables", err)
	}

	tables := make([]models.TableSchema, 0, len(result.Items))
	for _, item := range result.Items {
		table, err := r.parseTableFromItem(item)
		if err != nil {
			continue // Skip invalid items
		}
		tables = append(tables, *table)
	}

	return tables, nil
}

// UpdateTable updates a table schema
func (r *DynamoUserRepository) UpdateTable(ctx context.Context, table models.TableSchema) error {
	if err := table.ValidateFields(); err != nil {
		return NewRepositoryError(ErrorTypeInvalidInput, "invalid table fields", err)
	}

	fieldsBytes, err := json.Marshal(table.Fields)
	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to serialize fields", err)
	}

	now := time.Now()
	_, err = r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", r.userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TABLE#%s", table.ID)},
		},
		UpdateExpression: aws.String("SET Fields = :fields, UpdatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":fields":    &types.AttributeValueMemberS{Value: string(fieldsBytes)},
			":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})

	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to update table", err)
	}

	return nil
}

// DeleteTable deletes a table schema
func (r *DynamoUserRepository) DeleteTable(ctx context.Context, tableID string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", r.userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("TABLE#%s", tableID)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})

	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to delete table", err)
	}

	return nil
}

// GetTableHistory retrieves table change history
func (r *DynamoUserRepository) GetTableHistory(ctx context.Context, tableID string, opts models.QueryOptions) (*models.TableHistory, error) {
	// Query for table-related tuples
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#TABLE#%s", r.userID, tableID)},
			":sk": &types.AttributeValueMemberS{Value: "TUPLE#"},
		},
		ScanIndexForward: aws.Bool(false), // Most recent first
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to get table history", err)
	}

	tuples := make([]models.Tuple, 0, len(result.Items))
	for _, item := range result.Items {
		tuple, err := r.parseTupleFromItem(item)
		if err != nil {
			continue
		}
		tuples = append(tuples, *tuple)
	}

	return &models.TableHistory{
		TableID: tableID,
		Changes: tuples,
	}, nil
}

// Entity operations

// CreateEntity creates a new entity
func (r *DynamoUserRepository) CreateEntity(ctx context.Context, tableID string, fields map[string]interface{}) (*models.EntitySnapshot, error) {
	entityID := uuid.New().String()
	now := time.Now()

	entity := &models.EntitySnapshot{
		EntityID:  entityID,
		TableID:   tableID,
		UserID:    r.userID,
		Fields:    fields,
		IsDeleted: false,
		CreatedAt: &now,
		Timestamp: now,
	}

	// Store the entity snapshot
	if err := r.storeEntitySnapshot(ctx, entity); err != nil {
		return nil, err
	}

	// Store creation tuples for each field
	for fieldName, value := range fields {
		tuple := models.NewTuple(r.userID, entityID, now, tableID, fieldName, value)
		if err := r.storeTuple(ctx, &tuple); err != nil {
			return nil, err
		}
	}

	// Store system creation marker
	creationTuple := models.NewTuple(r.userID, entityID, now, tableID, models.SystemFieldCreated, "true")
	if err := r.storeTuple(ctx, &creationTuple); err != nil {
		return nil, err
	}

	return entity, nil
}

// GetEntity retrieves an entity at a specific point in time
func (r *DynamoUserRepository) GetEntity(ctx context.Context, tableID string, entityID string, asOf *time.Time) (*models.EntitySnapshot, error) {
	if asOf == nil {
		now := time.Now()
		asOf = &now
	}

	// Query for the latest snapshot before or at asOf time
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND SK <= :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#ENTITY#%s", r.userID, entityID)},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("SNAPSHOT#%d", asOf.Unix())},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(1),
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to get entity", err)
	}

	if len(result.Items) == 0 {
		return nil, NewRepositoryError(ErrorTypeNotFound, "entity not found", nil)
	}

	return r.parseEntityFromItem(result.Items[0])
}

// GetAllEntities retrieves all active entities for a table
func (r *DynamoUserRepository) GetAllEntities(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error) {
	if asOf == nil {
		now := time.Now()
		asOf = &now
	}

	// Query GSI1 for all entity snapshots in this table
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#TABLE#%s", r.userID, tableID)},
			":sk": &types.AttributeValueMemberS{Value: "ENTITY#"},
		},
		ScanIndexForward: aws.Bool(false), // Get most recent snapshots first
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to get all entities", err)
	}

	// Group all snapshots by entity ID first
	allSnapshots := make(map[string][]*models.EntitySnapshot)

	for _, item := range result.Items {
		entity, err := r.parseEntityFromItem(item)
		if err != nil {
			continue
		}

		allSnapshots[entity.EntityID] = append(allSnapshots[entity.EntityID], entity)
	}

	// For each entity, find the most recent snapshot before or at asOf time
	entityMap := make(map[string]*models.EntitySnapshot)

	for entityID, snapshots := range allSnapshots {
		var bestSnapshot *models.EntitySnapshot

		for _, snapshot := range snapshots {
			// Only consider snapshots at or before the asOf time
			// Use truncation to second precision to handle nanosecond differences
			snapshotTime := snapshot.Timestamp.Truncate(time.Second)
			asOfTime := asOf.Truncate(time.Second)

			if snapshotTime.Before(asOfTime) || snapshotTime.Equal(asOfTime) {
				// Keep the most recent valid snapshot
				if bestSnapshot == nil || snapshot.Timestamp.After(bestSnapshot.Timestamp) {
					bestSnapshot = snapshot
				}
			}
		}

		// Only include if we found a valid snapshot and it's not deleted
		if bestSnapshot != nil && !bestSnapshot.IsDeleted {
			entityMap[entityID] = bestSnapshot
		}
	}

	// Convert map to slice
	entities := make(map[string]models.EntitySnapshot)
	for _, entity := range entityMap {
		entities[entity.EntityID] = *entity
	}

	return &models.EntitiesSnapshot{
		TableID:   tableID,
		Entities:  entities,
		Timestamp: *asOf,
	}, nil
}

// GetAllEntitiesIncludingDeleted retrieves all entities including deleted ones
func (r *DynamoUserRepository) GetAllEntitiesIncludingDeleted(ctx context.Context, tableID string, asOf *time.Time) (*models.EntitiesSnapshot, error) {
	if asOf == nil {
		now := time.Now()
		asOf = &now
	}

	// Query GSI1 for all entity snapshots in this table
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#TABLE#%s", r.userID, tableID)},
			":sk": &types.AttributeValueMemberS{Value: "ENTITY#"},
		},
		ScanIndexForward: aws.Bool(false), // Get most recent snapshots first
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to get all entities including deleted", err)
	}

	// Group all snapshots by entity ID first
	allSnapshots := make(map[string][]*models.EntitySnapshot)

	for _, item := range result.Items {
		entity, err := r.parseEntityFromItem(item)
		if err != nil {
			continue
		}

		allSnapshots[entity.EntityID] = append(allSnapshots[entity.EntityID], entity)
	}

	// For each entity, find the most recent snapshot before or at asOf time
	entityMap := make(map[string]*models.EntitySnapshot)

	for entityID, snapshots := range allSnapshots {
		var bestSnapshot *models.EntitySnapshot

		for _, snapshot := range snapshots {
			// Only consider snapshots at or before the asOf time
			// Use truncation to second precision to handle nanosecond differences
			snapshotTime := snapshot.Timestamp.Truncate(time.Second)
			asOfTime := asOf.Truncate(time.Second)

			if snapshotTime.Before(asOfTime) || snapshotTime.Equal(asOfTime) {
				// Keep the most recent valid snapshot
				if bestSnapshot == nil || snapshot.Timestamp.After(bestSnapshot.Timestamp) {
					bestSnapshot = snapshot
				}
			}
		}

		// Include both active and deleted entities in this version
		if bestSnapshot != nil {
			entityMap[entityID] = bestSnapshot
		}
	}

	// Convert map to slice
	entities := make(map[string]models.EntitySnapshot)
	for _, entity := range entityMap {
		entities[entity.EntityID] = *entity
	}

	return &models.EntitiesSnapshot{
		TableID:   tableID,
		Entities:  entities,
		Timestamp: *asOf,
	}, nil
}

// UpdateEntity updates an entity
func (r *DynamoUserRepository) UpdateEntity(ctx context.Context, tableID string, entityID string, fields map[string]interface{}) (*models.EntitySnapshot, error) {
	// Get current entity
	current, err := r.GetEntity(ctx, tableID, entityID, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Create updated entity
	updated := current.Clone()
	for k, v := range fields {
		updated.SetFieldValue(k, v)
	}
	updated.Timestamp = now

	// Store the updated snapshot
	if err := r.storeEntitySnapshot(ctx, updated); err != nil {
		return nil, err
	}

	// Store update tuples for each field
	for fieldName, value := range fields {
		tuple := models.NewTuple(r.userID, entityID, now, tableID, fieldName, value)
		if err := r.storeTuple(ctx, &tuple); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// DeleteEntity soft deletes an entity
func (r *DynamoUserRepository) DeleteEntity(ctx context.Context, tableID string, entityID string) error {
	current, err := r.GetEntity(ctx, tableID, entityID, nil)
	if err != nil {
		return err
	}

	now := time.Now()
	current.IsDeleted = true
	current.DeletedAt = &now
	current.Timestamp = now

	if err := r.storeEntitySnapshot(ctx, current); err != nil {
		return err
	}

	deletionTuple := models.NewTuple(r.userID, entityID, now, tableID, models.SystemFieldDeleted, "true")
	return r.storeTuple(ctx, &deletionTuple)
}

// UndeleteEntity restores a deleted entity
func (r *DynamoUserRepository) UndeleteEntity(ctx context.Context, tableID string, entityID string) error {
	current, err := r.GetEntity(ctx, tableID, entityID, nil)
	if err != nil {
		return err
	}

	now := time.Now()
	current.IsDeleted = false
	current.DeletedAt = nil
	current.Timestamp = now

	if err := r.storeEntitySnapshot(ctx, current); err != nil {
		return err
	}

	undeletionTuple := models.NewTuple(r.userID, entityID, now, tableID, models.SystemFieldDeleted, "false")
	return r.storeTuple(ctx, &undeletionTuple)
}

// DeleteField removes a field from an entity
func (r *DynamoUserRepository) DeleteField(ctx context.Context, tableID string, entityID string, fieldName string) error {
	current, err := r.GetEntity(ctx, tableID, entityID, nil)
	if err != nil {
		return err
	}

	now := time.Now()
	current.DeleteField(fieldName)
	current.Timestamp = now

	if err := r.storeEntitySnapshot(ctx, current); err != nil {
		return err
	}

	// Create a deletion tuple for the field (empty value indicates deletion)
	deletionTuple := models.NewTuple(r.userID, entityID, now, tableID, fieldName, "")
	return r.storeTuple(ctx, &deletionTuple)
}

// GetFieldHistory retrieves history for a specific field
func (r *DynamoUserRepository) GetFieldHistory(ctx context.Context, tableID string, fieldName string, opts models.QueryOptions) (*models.FieldHistory, error) {
	// This would require a GSI on field names
	// For now, return empty history
	return &models.FieldHistory{
		TableID:   tableID,
		FieldName: fieldName,
		Changes:   []models.FieldChange{},
	}, nil
}

// GetEntityHistory retrieves the complete history of an entity
func (r *DynamoUserRepository) GetEntityHistory(ctx context.Context, tableID string, entityID string, opts models.QueryOptions) (*models.QueryResult, error) {
	result, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		FilterExpression:       aws.String("EntityID = :entityID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":       &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#TABLE#%s", r.userID, tableID)},
			":sk":       &types.AttributeValueMemberS{Value: "TUPLE#"},
			":entityID": &types.AttributeValueMemberS{Value: entityID},
		},
		ScanIndexForward: aws.Bool(false),
	})

	if err != nil {
		return nil, NewRepositoryError(ErrorTypeInternal, "failed to get entity history", err)
	}

	tuples := make([]models.Tuple, 0, len(result.Items))
	for _, item := range result.Items {
		tuple, err := r.parseTupleFromItem(item)
		if err != nil {
			continue
		}
		tuples = append(tuples, *tuple)
	}

	return &models.QueryResult{
		Tuples:  tuples,
		HasMore: false,
	}, nil
}

// HealthCheck verifies the repository is accessible
func (r *DynamoUserRepository) HealthCheck(ctx context.Context) error {
	_, err := r.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	})
	return err
}

// Helper methods

func (r *DynamoUserRepository) parseTableFromItem(item map[string]types.AttributeValue) (*models.TableSchema, error) {
	var fields []models.FieldDefinition
	if fieldsStr, ok := item["Fields"].(*types.AttributeValueMemberS); ok {
		if err := json.Unmarshal([]byte(fieldsStr.Value), &fields); err != nil {
			return nil, err
		}
	}

	createdAt, _ := time.Parse(time.RFC3339, item["CreatedAt"].(*types.AttributeValueMemberS).Value)
	updatedAt, _ := time.Parse(time.RFC3339, item["UpdatedAt"].(*types.AttributeValueMemberS).Value)

	return &models.TableSchema{
		ID:        item["TableID"].(*types.AttributeValueMemberS).Value,
		UserID:    item["UserID"].(*types.AttributeValueMemberS).Value,
		Fields:    fields,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (r *DynamoUserRepository) parseEntityFromItem(item map[string]types.AttributeValue) (*models.EntitySnapshot, error) {
	var entity models.EntitySnapshot
	if err := attributevalue.UnmarshalMap(item, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *DynamoUserRepository) parseTupleFromItem(item map[string]types.AttributeValue) (*models.Tuple, error) {
	var tuple models.Tuple
	if err := attributevalue.UnmarshalMap(item, &tuple); err != nil {
		return nil, err
	}
	return &tuple, nil
}

func (r *DynamoUserRepository) storeEntitySnapshot(ctx context.Context, entity *models.EntitySnapshot) error {
	item, err := attributevalue.MarshalMap(entity)
	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to marshal entity", err)
	}

	// Add DynamoDB keys
	item["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#ENTITY#%s", r.userID, entity.EntityID)}
	item["SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("SNAPSHOT#%d", entity.Timestamp.Unix())}

	// Add GSI1 keys for table-based queries
	item["GSI1PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#TABLE#%s", r.userID, entity.TableID)}
	item["GSI1SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("ENTITY#%s#%d", entity.EntityID, entity.Timestamp.Unix())}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})

	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to store entity snapshot", err)
	}

	return nil
}

func (r *DynamoUserRepository) storeTuple(ctx context.Context, tuple *models.Tuple) error {
	item, err := attributevalue.MarshalMap(tuple)
	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to marshal tuple", err)
	}

	// Add DynamoDB keys
	item["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s#TABLE#%s", r.userID, tuple.TableID)}
	item["SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("TUPLE#%d#%s", tuple.Timestamp.Unix(), tuple.EntityID)}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})

	if err != nil {
		return NewRepositoryError(ErrorTypeInternal, "failed to store tuple", err)
	}

	return nil
}
