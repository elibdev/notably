package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/elibdev/notably/internal/models"
)

func TestNewTableSchema(t *testing.T) {
	userID := "user_123"
	tableID := "contacts"
	fields := []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
		{Name: "email", DataType: models.DataTypeString},
	}

	schema := models.NewTableSchema(userID, tableID, fields)

	assert.Equal(t, tableID, schema.ID)
	assert.Equal(t, userID, schema.UserID)
	assert.Equal(t, fields, schema.Fields)
	assert.Equal(t, "USER#"+userID, schema.PK)
	assert.Equal(t, "TABLE#"+tableID, schema.SK)
	assert.WithinDuration(t, time.Now(), schema.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), schema.UpdatedAt, time.Second)
}

func TestTableSchema_GetField(t *testing.T) {
	schema := models.NewTableSchema("user_123", "contacts", []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
		{Name: "email", DataType: models.DataTypeString},
		{Name: "age", DataType: models.DataTypeInt},
	})

	tests := []struct {
		name      string
		fieldName string
		wantFound bool
		wantType  models.DataType
	}{
		{"existing string field", "name", true, models.DataTypeString},
		{"existing email field", "email", true, models.DataTypeString},
		{"existing int field", "age", true, models.DataTypeInt},
		{"non-existing field", "phone", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, found := schema.GetField(tt.fieldName)
			assert.Equal(t, tt.wantFound, found)
			if found {
				assert.Equal(t, tt.fieldName, field.Name)
				assert.Equal(t, tt.wantType, field.DataType)
			} else {
				assert.Nil(t, field)
			}
		})
	}
}

func TestTableSchema_AddField(t *testing.T) {
	schema := models.NewTableSchema("user_123", "contacts", []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
	})

	originalUpdateTime := schema.UpdatedAt
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	newField := models.FieldDefinition{Name: "phone", DataType: models.DataTypeString}
	schema.AddField(newField)

	assert.Len(t, schema.Fields, 2)
	field, found := schema.GetField("phone")
	assert.True(t, found)
	assert.Equal(t, newField, *field)
	assert.True(t, schema.UpdatedAt.After(originalUpdateTime))
}

func TestTableSchema_RemoveField(t *testing.T) {
	schema := models.NewTableSchema("user_123", "contacts", []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
		{Name: "email", DataType: models.DataTypeString},
		{Name: "phone", DataType: models.DataTypeString},
	})

	originalUpdateTime := schema.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	// Remove existing field
	removed := schema.RemoveField("email")
	assert.True(t, removed)
	assert.Len(t, schema.Fields, 2)
	_, found := schema.GetField("email")
	assert.False(t, found)
	assert.True(t, schema.UpdatedAt.After(originalUpdateTime))

	// Try to remove non-existing field
	removed = schema.RemoveField("nonexistent")
	assert.False(t, removed)
	assert.Len(t, schema.Fields, 2)
}

func TestDataTypes(t *testing.T) {
	tests := []struct {
		name     string
		dataType models.DataType
		expected string
	}{
		{"string type", models.DataTypeString, "string"},
		{"int type", models.DataTypeInt, "int"},
		{"float type", models.DataTypeFloat, "float"},
		{"bool type", models.DataTypeBool, "bool"},
		{"date type", models.DataTypeDate, "date"},
		{"json type", models.DataTypeJSON, "json"},
		{"reference type", models.DataTypeReference, "reference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.dataType))
		})
	}
}

func TestDataType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		dataType models.DataType
		expected bool
	}{
		{"valid string", models.DataTypeString, true},
		{"valid int", models.DataTypeInt, true},
		{"valid float", models.DataTypeFloat, true},
		{"valid bool", models.DataTypeBool, true},
		{"valid date", models.DataTypeDate, true},
		{"valid json", models.DataTypeJSON, true},
		{"valid reference", models.DataTypeReference, true},
		{"invalid type", models.DataType("invalid"), false},
		{"empty type", models.DataType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dataType.IsValid())
		})
	}
}

func TestTableSchema_ValidateFields(t *testing.T) {
	tests := []struct {
		name      string
		fields    []models.FieldDefinition
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid fields",
			fields: []models.FieldDefinition{
				{Name: "name", DataType: models.DataTypeString},
				{Name: "age", DataType: models.DataTypeInt},
			},
			wantError: false,
		},
		{
			name:      "empty fields",
			fields:    []models.FieldDefinition{},
			wantError: true,
			errorMsg:  "table must have at least one field",
		},
		{
			name: "empty field name",
			fields: []models.FieldDefinition{
				{Name: "", DataType: models.DataTypeString},
			},
			wantError: true,
			errorMsg:  "field name cannot be empty",
		},
		{
			name: "duplicate field name",
			fields: []models.FieldDefinition{
				{Name: "name", DataType: models.DataTypeString},
				{Name: "name", DataType: models.DataTypeInt},
			},
			wantError: true,
			errorMsg:  "duplicate field name: name",
		},
		{
			name: "invalid data type",
			fields: []models.FieldDefinition{
				{Name: "test", DataType: models.DataType("invalid")},
			},
			wantError: true,
			errorMsg:  "invalid data type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := models.NewTableSchema("user_123", "test_table", tt.fields)
			err := schema.ValidateFields()

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTableSchema_KeyGeneration(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		tableID   string
		expectedPK string
		expectedSK string
	}{
		{"basic IDs", "user_123", "contacts", "USER#user_123", "TABLE#contacts"},
		{"email user", "user@example.com", "notes", "USER#user@example.com", "TABLE#notes"},
		{"complex IDs", "user-with-dashes", "table_with_underscores", "USER#user-with-dashes", "TABLE#table_with_underscores"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := models.NewTableSchema(tt.userID, tt.tableID, []models.FieldDefinition{
				{Name: "test", DataType: models.DataTypeString},
			})

			assert.Equal(t, tt.expectedPK, schema.PK)
			assert.Equal(t, tt.expectedSK, schema.SK)
		})
	}
}

func TestTableSchema_FieldManipulation(t *testing.T) {
	schema := models.NewTableSchema("user_123", "test_table", []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
		{Name: "email", DataType: models.DataTypeString},
	})

	// Test initial state
	assert.Len(t, schema.Fields, 2)

	// Test adding multiple fields
	schema.AddField(models.FieldDefinition{Name: "age", DataType: models.DataTypeInt})
	schema.AddField(models.FieldDefinition{Name: "active", DataType: models.DataTypeBool})
	assert.Len(t, schema.Fields, 4)

	// Test that all fields are present
	fields := []string{"name", "email", "age", "active"}
	for _, fieldName := range fields {
		_, found := schema.GetField(fieldName)
		assert.True(t, found, "Field %s should exist", fieldName)
	}

	// Test removing fields
	removed := schema.RemoveField("email")
	assert.True(t, removed)
	assert.Len(t, schema.Fields, 3)

	// Verify email is gone but others remain
	_, found := schema.GetField("email")
	assert.False(t, found)

	for _, fieldName := range []string{"name", "age", "active"} {
		_, found := schema.GetField(fieldName)
		assert.True(t, found, "Field %s should still exist", fieldName)
	}
}

func TestTableSchema_TimestampBehavior(t *testing.T) {
	schema := models.NewTableSchema("user_123", "test_table", []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
	})

	// Initially, CreatedAt and UpdatedAt should be the same
	assert.Equal(t, schema.CreatedAt, schema.UpdatedAt)

	originalCreatedAt := schema.CreatedAt
	originalUpdatedAt := schema.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	// Adding a field should update UpdatedAt but not CreatedAt
	schema.AddField(models.FieldDefinition{Name: "email", DataType: models.DataTypeString})

	assert.Equal(t, originalCreatedAt, schema.CreatedAt) // CreatedAt shouldn't change
	assert.True(t, schema.UpdatedAt.After(originalUpdatedAt)) // UpdatedAt should be later

	updatedAfterAdd := schema.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	// Removing a field should also update UpdatedAt
	schema.RemoveField("email")

	assert.Equal(t, originalCreatedAt, schema.CreatedAt) // CreatedAt still shouldn't change
	assert.True(t, schema.UpdatedAt.After(updatedAfterAdd)) // UpdatedAt should be even later
}