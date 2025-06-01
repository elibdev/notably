package helpers

import (
	"context"
	"testing"
	"time"

	"github.com/elibdev/notably/internal/models"
	"github.com/elibdev/notably/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertEntityEqual compares two EntitySnapshots for equality
func AssertEntityEqual(t *testing.T, expected, actual *models.EntitySnapshot) {
	require.NotNil(t, actual)
	assert.Equal(t, expected.EntityID, actual.EntityID)
	assert.Equal(t, expected.TableID, actual.TableID)
	assert.Equal(t, expected.IsDeleted, actual.IsDeleted)

	// Compare fields
	assert.Len(t, actual.Fields, len(expected.Fields))
	for fieldName, expectedValue := range expected.Fields {
		actualValue, exists := actual.Fields[fieldName]
		assert.True(t, exists, "Field %s should exist", fieldName)
		assert.Equal(t, expectedValue, actualValue, "Field %s value mismatch", fieldName)
	}
}

// AssertTableSnapshotEqual compares two TableSnapshots for equality
func AssertTableSnapshotEqual(t *testing.T, expected, actual *models.TableSnapshot) {
	require.NotNil(t, actual)
	assert.Equal(t, expected.TableID, actual.TableID)
	assert.Len(t, actual.Entities, len(expected.Entities))

	for entityID, expectedEntity := range expected.Entities {
		actualEntity, exists := actual.Entities[entityID]
		assert.True(t, exists, "Entity %s should exist", entityID)
		AssertEntityEqual(t, &expectedEntity, &actualEntity)
	}
}

// SetupTestRepository sets up a test repository with sample data
func SetupTestRepository(t *testing.T, userManager repository.UserManager, userID string) repository.UserRepository {
	ctx := context.Background()

	// Create user if not exists
	_, err := userManager.CreateUser(ctx, userID)
	if err != nil {
		// User might already exist, try to get repository
		t.Logf("User creation failed (might already exist): %v", err)
	}

	repo := userManager.GetUserRepository(userID)

	return repo
}

// CreateTestTables creates standard test tables in the repository
func CreateTestTables(t *testing.T, repo repository.UserRepository) (contacts, notes, projects *models.TableSchema) {
	ctx := context.Background()

	var err error
	contacts, err = repo.CreateTable(ctx, "contacts", ContactsFields)
	require.NoError(t, err)

	notes, err = repo.CreateTable(ctx, "notes", NotesFields)
	require.NoError(t, err)

	projects, err = repo.CreateTable(ctx, "projects", ProjectsFields)
	require.NoError(t, err)

	return contacts, notes, projects
}

// PopulateTestData adds sample data to the repository
func PopulateTestData(t *testing.T, repo repository.UserRepository) {
	userID := getUserIDFromRepo(repo)

	// Create tables first
	_, _, _ = CreateTestTables(t, repo)

	// Add all test tuples - need to implement PutBatch method
	tuples := CreateCompleteTestDataset(userID)

	// For now, put tuples individually since PutBatch might not be implemented yet
	for _, tuple := range tuples {
		// This would need to be implemented in the repository
		t.Logf("Would put tuple: %+v", tuple)
	}
}

// AssertEntityExists checks that an entity exists and optionally validates specific fields
func AssertEntityExists(t *testing.T, repo repository.UserRepository, tableID, entityID string, expectedFields map[string]interface{}) {
	ctx := context.Background()

	entity, err := repo.GetEntity(ctx, tableID, entityID, nil)
	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.False(t, entity.IsDeleted)

	if expectedFields != nil {
		for fieldName, expectedValue := range expectedFields {
			actualValue, exists := entity.Fields[fieldName]
			assert.True(t, exists, "Field %s should exist", fieldName)
			assert.Equal(t, expectedValue, actualValue, "Field %s value mismatch", fieldName)
		}
	}
}

// AssertEntityDeleted checks that an entity is marked as deleted
func AssertEntityDeleted(t *testing.T, repo repository.UserRepository, tableID, entityID string) {
	ctx := context.Background()

	entity, err := repo.GetEntity(ctx, tableID, entityID, nil)
	// Should return nil for deleted entities in normal queries
	assert.Nil(t, entity)

	// But should exist in "including deleted" queries
	entities, err := repo.GetAllEntitiesIncludingDeleted(ctx, tableID, nil)
	require.NoError(t, err)

	deletedEntity, exists := entities.Entities[entityID]
	require.True(t, exists, "Entity should exist in deleted entities query")
	assert.True(t, deletedEntity.IsDeleted)
}

// AssertEntityDoesNotExist checks that an entity doesn't exist at all
func AssertEntityDoesNotExist(t *testing.T, repo repository.UserRepository, tableID, entityID string) {
	ctx := context.Background()

	entity, err := repo.GetEntity(ctx, tableID, entityID, nil)
	assert.NoError(t, err)
	assert.Nil(t, entity)
}

// WaitForConsistency adds a small delay for eventual consistency in tests
func WaitForConsistency() {
	time.Sleep(100 * time.Millisecond)
}

// CreateTimeSeriesData creates data across multiple time points for time travel testing
func CreateTimeSeriesData(userID, entityID, tableID string, fieldName string, values []string, baseTime time.Time, interval time.Duration) []models.Tuple {
	var tuples []models.Tuple

	for i, value := range values {
		timestamp := baseTime.Add(time.Duration(i) * interval)
		tuple := models.NewTuple(userID, entityID, timestamp, tableID, fieldName, value)
		tuples = append(tuples, tuple)
	}

	return tuples
}

// AssertTimeTravel validates that querying at a specific time returns expected data
func AssertTimeTravel(t *testing.T, repo repository.UserRepository, tableID, entityID string, asOfTime time.Time, expectedFields map[string]interface{}) {
	ctx := context.Background()

	entity, err := repo.GetEntity(ctx, tableID, entityID, &asOfTime)
	require.NoError(t, err)

	if expectedFields == nil {
		assert.Nil(t, entity, "Entity should not exist at time %v", asOfTime)
		return
	}

	require.NotNil(t, entity, "Entity should exist at time %v", asOfTime)

	for fieldName, expectedValue := range expectedFields {
		actualValue, exists := entity.Fields[fieldName]
		assert.True(t, exists, "Field %s should exist at time %v", fieldName, asOfTime)
		assert.Equal(t, expectedValue, actualValue, "Field %s value mismatch at time %v", fieldName, asOfTime)
	}
}

// AssertEntitiesSnapshot compares an EntitiesSnapshot for expected content
func AssertEntitiesSnapshot(t *testing.T, snapshot *models.EntitiesSnapshot, expectedEntityIDs []string) {
	require.NotNil(t, snapshot)
	assert.Len(t, snapshot.Entities, len(expectedEntityIDs))

	for _, entityID := range expectedEntityIDs {
		_, exists := snapshot.Entities[entityID]
		assert.True(t, exists, "Entity %s should exist in snapshot", entityID)
	}
}

// CreateEntityWithFields creates an EntitySnapshot with the given fields
func CreateEntityWithFields(entityID, tableID string, fields map[string]interface{}, isDeleted bool) models.EntitySnapshot {
	now := time.Now()
	entity := models.EntitySnapshot{
		EntityID:  entityID,
		TableID:   tableID,
		Fields:    make(map[string]interface{}),
		IsDeleted: isDeleted,
		Timestamp: now,
	}

	if !isDeleted {
		createdAt := now
		entity.CreatedAt = &createdAt
	}

	if isDeleted {
		deletedAt := now
		entity.DeletedAt = &deletedAt
	}

	for k, v := range fields {
		entity.Fields[k] = v
	}

	return entity
}

// CreateMockTableSnapshot creates a mock TableSnapshot for testing
func CreateMockTableSnapshot(tableID string, entities map[string]models.EntitySnapshot) *models.TableSnapshot {
	return &models.TableSnapshot{
		TableID:   tableID,
		Entities:  entities,
		Timestamp: time.Now(),
	}
}

// AssertFieldValue checks that an entity has a specific field value
func AssertFieldValue(t *testing.T, entity *models.EntitySnapshot, fieldName string, expectedValue interface{}) {
	require.NotNil(t, entity)
	actualValue, exists := entity.Fields[fieldName]
	assert.True(t, exists, "Field %s should exist", fieldName)
	assert.Equal(t, expectedValue, actualValue, "Field %s value mismatch", fieldName)
}

// AssertFieldDoesNotExist checks that an entity does not have a specific field
func AssertFieldDoesNotExist(t *testing.T, entity *models.EntitySnapshot, fieldName string) {
	require.NotNil(t, entity)
	_, exists := entity.Fields[fieldName]
	assert.False(t, exists, "Field %s should not exist", fieldName)
}

// CreateTestEntitySnapshot creates a test entity snapshot with basic fields
func CreateTestEntitySnapshot(entityID, tableID string) models.EntitySnapshot {
	now := time.Now()
	return models.EntitySnapshot{
		EntityID:  entityID,
		TableID:   tableID,
		Fields:    make(map[string]interface{}),
		IsDeleted: false,
		CreatedAt: &now,
		Timestamp: now,
	}
}

// getUserIDFromRepo extracts user ID from a repository (helper function)
func getUserIDFromRepo(repo repository.UserRepository) string {
	// This would need to be implemented based on the actual repository interface
	// For now, return a test user ID
	return "test_user"
}

// AssertTimestampWithinRange checks that a timestamp is within an expected range
func AssertTimestampWithinRange(t *testing.T, actual time.Time, expected time.Time, tolerance time.Duration) {
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff <= tolerance,
		"Timestamp %v should be within %v of %v (actual diff: %v)",
		actual, tolerance, expected, diff)
}

// CreateBatchTestEntities creates multiple test entities for batch operations
func CreateBatchTestEntities(userID, tableID string, count int) []models.EntitySnapshot {
	entities := make([]models.EntitySnapshot, count)
	for i := 0; i < count; i++ {
		entityID := models.NewEntityID()
		entities[i] = CreateTestEntitySnapshot(entityID, tableID)
		entities[i].Fields["index"] = i
		entities[i].Fields["name"] = "Entity " + entityID[:8]
	}
	return entities
}
