package models

import (
	"testing"
	"time"

	"github.com/elibdev/notably/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEntityID(t *testing.T) {
	// Generate multiple IDs to test uniqueness and format
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := models.NewEntityID()

		// Check ULID format (26 characters, base32)
		assert.Len(t, id, 26)
		assert.Regexp(t, "^[0-9A-HJKMNP-TV-Z]{26}$", id)

		// Check uniqueness
		assert.False(t, ids[id], "Duplicate ULID generated: %s", id)
		ids[id] = true
	}
}

func TestNewTuple(t *testing.T) {
	userID := "user_123"
	entityID := "01HZYX987654321ZYXWVUTSRQP"
	timestamp := time.Now()
	tableID := "contacts"
	fieldName := "name"
	value := "John Doe"

	tuple := models.NewTuple(userID, entityID, timestamp, tableID, fieldName, value)

	assert.Equal(t, entityID, tuple.EntityID)
	assert.Equal(t, timestamp, tuple.Timestamp)
	assert.Equal(t, userID, tuple.UserID)
	assert.Equal(t, tableID, tuple.TableID)
	assert.Equal(t, fieldName, tuple.FieldName)
	assert.Equal(t, value, tuple.Value)

	// Check DynamoDB keys
	expectedPK := "USER#" + userID
	expectedSK := "TUPLE#" + tableID + "#" + entityID + "#" + timestamp.Format(time.RFC3339) + "#" + fieldName
	assert.Equal(t, expectedPK, tuple.PK)
	assert.Equal(t, expectedSK, tuple.SK)

	// Check GSI keys
	expectedGSI1PK := "USER#" + userID + "#" + tableID + "#" + fieldName
	expectedGSI1SK := timestamp.Format(time.RFC3339) + "#" + entityID
	assert.Equal(t, expectedGSI1PK, tuple.GSI1PK)
	assert.Equal(t, expectedGSI1SK, tuple.GSI1SK)

	expectedGSI2PK := "USER#" + userID + "#" + tableID + "#" + entityID
	expectedGSI2SK := timestamp.Format(time.RFC3339) + "#" + fieldName
	assert.Equal(t, expectedGSI2PK, tuple.GSI2PK)
	assert.Equal(t, expectedGSI2SK, tuple.GSI2SK)

	expectedGSI3PK := "USER#" + userID + "#" + tableID
	expectedGSI3SK := entityID + "#" + timestamp.Format(time.RFC3339) + "#" + fieldName
	assert.Equal(t, expectedGSI3PK, tuple.GSI3PK)
	assert.Equal(t, expectedGSI3SK, tuple.GSI3SK)
}

func TestEncodeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{"nil value", nil, "", false},
		{"string value", "hello", "hello", false},
		{"int value", 42, "42", false},
		{"int64 value", int64(42), "42", false},
		{"float value", 3.14, "3.14", false},
		{"bool true", true, "true", false},
		{"bool false", false, "false", false},
		{"slice value", []string{"a", "b"}, `["a","b"]`, false},
		{"map value", map[string]string{"key": "value"}, `{"key":"value"}`, false},
		{"time value", time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), `"2023-01-01T12:00:00Z"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := models.EncodeValue(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDecodeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dataType models.DataType
		expected interface{}
		wantErr  bool
	}{
		{"empty string", "", models.DataTypeString, nil, false},
		{"string value", "hello", models.DataTypeString, "hello", false},
		{"int value", "42", models.DataTypeInt, int64(42), false},
		{"int from json", "42", models.DataTypeInt, int64(42), false},
		{"float value", "3.14", models.DataTypeFloat, 3.14, false},
		{"bool true", "true", models.DataTypeBool, true, false},
		{"bool false", "false", models.DataTypeBool, false, false},
		{"json array", `["a","b"]`, models.DataTypeJSON, []interface{}{"a", "b"}, false},
		{"json object", `{"key":"value"}`, models.DataTypeJSON, map[string]interface{}{"key": "value"}, false},
		{"date value", "2023-01-01T12:00:00Z", models.DataTypeDate, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), false},
		{"reference value", "entity_123", models.DataTypeReference, "entity_123", false},
		{"invalid int", "not_a_number", models.DataTypeInt, nil, true},
		{"invalid float", "not_a_float", models.DataTypeFloat, nil, true},
		{"invalid bool", "not_a_bool", models.DataTypeBool, nil, true},
		{"invalid json", `{"incomplete": json`, models.DataTypeJSON, nil, true},
		{"invalid date", "not_a_date", models.DataTypeDate, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := models.DecodeValue(tt.input, tt.dataType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSystemFields(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{"deleted field", models.SystemFieldDeleted, true},
		{"created field", models.SystemFieldCreated, true},
		{"regular field", "name", false},
		{"user field starting with underscore", "_myfield", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.IsSystemField(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTupleKeyGeneration(t *testing.T) {
	userID := "user_123"
	entityID := "01HZYX987654321ZYXWVUTSRQP"
	timestamp := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
	tableID := "contacts"
	fieldName := "email"
	value := "test@example.com"

	tuple := models.NewTuple(userID, entityID, timestamp, tableID, fieldName, value)

	// Test specific key formats
	assert.Equal(t, "USER#user_123", tuple.PK)
	assert.Equal(t, "TUPLE#contacts#01HZYX987654321ZYXWVUTSRQP#2023-12-25T10:30:00Z#email", tuple.SK)
	assert.Equal(t, "USER#user_123#contacts#email", tuple.GSI1PK)
	assert.Equal(t, "2023-12-25T10:30:00Z#01HZYX987654321ZYXWVUTSRQP", tuple.GSI1SK)
	assert.Equal(t, "USER#user_123#contacts#01HZYX987654321ZYXWVUTSRQP", tuple.GSI2PK)
	assert.Equal(t, "2023-12-25T10:30:00Z#email", tuple.GSI2SK)
	assert.Equal(t, "USER#user_123#contacts", tuple.GSI3PK)
	assert.Equal(t, "01HZYX987654321ZYXWVUTSRQP#2023-12-25T10:30:00Z#email", tuple.GSI3SK)
}

func TestGetUserIDFromPK(t *testing.T) {
	tests := []struct {
		name     string
		pk       string
		expected string
	}{
		{"valid user PK", "USER#user_123", "user_123"},
		{"email user PK", "USER#user@example.com", "user@example.com"},
		{"uuid user PK", "USER#550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
		{"invalid PK format", "INVALID#user_123", ""},
		{"empty PK", "", ""},
		{"wrong prefix", "TABLE#something", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.GetUserIDFromPK(tt.pk)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTableIDFromSK(t *testing.T) {
	tests := []struct {
		name     string
		sk       string
		expected string
	}{
		{"valid table SK", "TABLE#contacts", "contacts"},
		{"table with underscores", "TABLE#contact_notes", "contact_notes"},
		{"table with dashes", "TABLE#contact-info", "contact-info"},
		{"invalid SK format", "INVALID#contacts", ""},
		{"empty SK", "", ""},
		{"wrong prefix", "USER#something", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.GetTableIDFromSK(tt.sk)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTupleSK(t *testing.T) {
	tests := []struct {
		name          string
		sk            string
		wantTableID   string
		wantEntityID  string
		wantTimestamp string
		wantFieldName string
		wantErr       bool
	}{
		{
			name:          "valid tuple SK",
			sk:            "TUPLE#contacts#01HZYX987654321ZYXWVUTSRQP#2023-12-25T10:30:00Z#email",
			wantTableID:   "contacts",
			wantEntityID:  "01HZYX987654321ZYXWVUTSRQP",
			wantTimestamp: "2023-12-25T10:30:00Z",
			wantFieldName: "email",
			wantErr:       false,
		},
		{
			name:    "invalid prefix",
			sk:      "INVALID#contacts#entity#timestamp#field",
			wantErr: true,
		},
		{
			name:    "too few parts",
			sk:      "TUPLE#contacts#entity",
			wantErr: true,
		},
		{
			name:    "empty SK",
			sk:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableID, entityID, timestamp, fieldName, err := models.ParseTupleSK(tt.sk)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTableID, tableID)
				assert.Equal(t, tt.wantEntityID, entityID)
				assert.Equal(t, tt.wantTimestamp, timestamp)
				assert.Equal(t, tt.wantFieldName, fieldName)
			}
		})
	}
}

func TestIsValidEntityID(t *testing.T) {
	// Generate some valid ULIDs for testing
	validULIDs := make([]string, 5)
	for i := range validULIDs {
		validULIDs[i] = models.NewEntityID()
	}

	tests := []struct {
		name     string
		entityID string
		expected bool
	}{
		{"valid ULID 1", validULIDs[0], true},
		{"valid ULID 2", validULIDs[1], true},
		{"known valid ULID", "01HZYX987654321ZYXWVSTRSQP", true},
		{"too short", "01HZYX98765", false},
		{"too long", "01HZYX987654321ZYXWVUTSRQPXX", false},
		{"invalid characters", "01HZYX987654321ZYXWVUTSRQI", false}, // Contains 'I'
		{"invalid characters", "01HZYX987654321ZYXWVUTSRQL", false}, // Contains 'L'
		{"invalid characters", "01HZYX987654321ZYXWVUTSRQO", false}, // Contains 'O'
		{"invalid characters", "01HZYX987654321ZYXWVUTSRQU", false}, // Contains 'U'
		{"empty string", "", false},
		{"lowercase", "01hzyx987654321zyxwvutsrqp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.IsValidEntityID(tt.entityID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTuple_ValueEncoding(t *testing.T) {
	userID := "user_123"
	entityID := models.NewEntityID()
	timestamp := time.Now()
	tableID := "test_table"

	tests := []struct {
		name      string
		fieldName string
		value     interface{}
		expected  string
	}{
		{"string field", "name", "John Doe", "John Doe"},
		{"int field", "age", 30, "30"},
		{"bool field", "active", true, "true"},
		{"float field", "score", 95.5, "95.5"},
		{"nil field", "optional", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuple := models.NewTuple(userID, entityID, timestamp, tableID, tt.fieldName, tt.value)
			assert.Equal(t, tt.expected, tuple.Value)
		})
	}
}

func TestTuple_ComplexValues(t *testing.T) {
	userID := "user_123"
	entityID := models.NewEntityID()
	timestamp := time.Now()
	tableID := "test_table"

	// Test complex JSON values
	complexValue := map[string]interface{}{
		"nested": map[string]interface{}{
			"array": []int{1, 2, 3},
			"bool":  true,
		},
		"string": "test",
	}

	tuple := models.NewTuple(userID, entityID, timestamp, tableID, "complex_field", complexValue)

	// The value should be JSON encoded
	assert.Contains(t, tuple.Value, "nested")
	assert.Contains(t, tuple.Value, "array")
	assert.Contains(t, tuple.Value, "test")

	// Test that we can decode it back
	decoded, err := models.DecodeValue(tuple.Value, models.DataTypeJSON)
	require.NoError(t, err)

	decodedMap := decoded.(map[string]interface{})
	assert.Equal(t, "test", decodedMap["string"])

	nested := decodedMap["nested"].(map[string]interface{})
	assert.Equal(t, true, nested["bool"])
}

func TestTuple_TimestampConsistency(t *testing.T) {
	userID := "user_123"
	entityID := models.NewEntityID()
	timestamp := time.Date(2023, 12, 25, 10, 30, 15, 123456789, time.UTC)
	tableID := "test_table"
	fieldName := "test_field"

	tuple := models.NewTuple(userID, entityID, timestamp, tableID, fieldName, "test_value")

	// The timestamp should be preserved exactly
	assert.Equal(t, timestamp, tuple.Timestamp)

	// The timestamp in the sort key should be RFC3339 formatted
	expectedTimestampStr := timestamp.Format(time.RFC3339)
	assert.Contains(t, tuple.SK, expectedTimestampStr)
	assert.Contains(t, tuple.GSI1SK, expectedTimestampStr)
	assert.Contains(t, tuple.GSI2SK, expectedTimestampStr)
	assert.Contains(t, tuple.GSI3SK, expectedTimestampStr)
}
