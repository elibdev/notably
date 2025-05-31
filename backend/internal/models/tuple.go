package models

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// System field constants
const (
	SystemFieldDeleted = "_deleted"
	SystemFieldCreated = "_created"
)

// Tuple represents a single data point in the time-versioned database
type Tuple struct {
	EntityID  string    `json:"entity_id"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
	TableID   string    `json:"table_id"`
	FieldName string    `json:"field_name"`
	Value     string    `json:"value"`

	// DynamoDB keys
	PK string `json:"pk"` // Partition key: USER#{userID}
	SK string `json:"sk"` // Sort key: TUPLE#{tableID}#{entityID}#{timestamp}#{fieldName}

	// GSI keys for different query patterns
	GSI1PK string `json:"gsi1_pk"` // USER#{userID}#{tableID}#{fieldName}
	GSI1SK string `json:"gsi1_sk"` // {timestamp}#{entityID}
	GSI2PK string `json:"gsi2_pk"` // USER#{userID}#{tableID}#{entityID}
	GSI2SK string `json:"gsi2_sk"` // {timestamp}#{fieldName}
	GSI3PK string `json:"gsi3_pk"` // USER#{userID}#{tableID}
	GSI3SK string `json:"gsi3_sk"` // {entityID}#{timestamp}#{fieldName}
}

// NewEntityID generates a new ULID for entity identification
func NewEntityID() string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// NewTuple creates a new tuple with proper DynamoDB keys
func NewTuple(userID, entityID string, timestamp time.Time, tableID, fieldName string, value interface{}) Tuple {
	encodedValue, _ := EncodeValue(value)
	timestampStr := timestamp.Format(time.RFC3339)

	return Tuple{
		EntityID:  entityID,
		Timestamp: timestamp,
		UserID:    userID,
		TableID:   tableID,
		FieldName: fieldName,
		Value:     encodedValue,

		// Main table keys
		PK: "USER#" + userID,
		SK: "TUPLE#" + tableID + "#" + entityID + "#" + timestampStr + "#" + fieldName,

		// GSI1: Query by field across time
		GSI1PK: "USER#" + userID + "#" + tableID + "#" + fieldName,
		GSI1SK: timestampStr + "#" + entityID,

		// GSI2: Query entity history
		GSI2PK: "USER#" + userID + "#" + tableID + "#" + entityID,
		GSI2SK: timestampStr + "#" + fieldName,

		// GSI3: Query table data with entity grouping
		GSI3PK: "USER#" + userID + "#" + tableID,
		GSI3SK: entityID + "#" + timestampStr + "#" + fieldName,
	}
}

// EncodeValue converts an interface{} value to a string representation
func EncodeValue(value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case time.Time:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to encode time value: %w", err)
		}
		return string(jsonBytes), nil
	default:
		// For complex types (slices, maps, etc.), use JSON encoding
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("failed to encode value: %w", err)
		}
		return string(jsonBytes), nil
	}
}

// DecodeValue converts a string representation back to the appropriate type
func DecodeValue(encoded string, dataType DataType) (interface{}, error) {
	if encoded == "" {
		return nil, nil
	}

	switch dataType {
	case DataTypeString:
		return encoded, nil
	case DataTypeInt:
		// Try to parse as JSON number first (for backwards compatibility)
		var intVal int64
		if err := json.Unmarshal([]byte(encoded), &intVal); err == nil {
			return intVal, nil
		}
		// Fall back to string parsing
		return strconv.ParseInt(encoded, 10, 64)
	case DataTypeFloat:
		return strconv.ParseFloat(encoded, 64)
	case DataTypeBool:
		return strconv.ParseBool(encoded)
	case DataTypeDate:
		return time.Parse(time.RFC3339, encoded)
	case DataTypeJSON:
		var jsonVal interface{}
		err := json.Unmarshal([]byte(encoded), &jsonVal)
		return jsonVal, err
	case DataTypeReference:
		return encoded, nil
	default:
		return encoded, nil
	}
}

// IsSystemField checks if a field name is a system field
func IsSystemField(fieldName string) bool {
	return fieldName == SystemFieldDeleted || fieldName == SystemFieldCreated
}



// GetUserIDFromPK extracts user ID from a partition key
func GetUserIDFromPK(pk string) string {
	if strings.HasPrefix(pk, "USER#") {
		return pk[5:] // Remove "USER#" prefix
	}
	return ""
}

// GetTableIDFromSK extracts table ID from a sort key (for TABLE entries)
func GetTableIDFromSK(sk string) string {
	if strings.HasPrefix(sk, "TABLE#") {
		return sk[6:] // Remove "TABLE#" prefix
	}
	return ""
}

// ParseTupleSK parses a tuple sort key and extracts its components
func ParseTupleSK(sk string) (tableID, entityID, timestamp, fieldName string, err error) {
	if !strings.HasPrefix(sk, "TUPLE#") {
		return "", "", "", "", fmt.Errorf("invalid tuple sort key format")
	}

	// Remove "TUPLE#" prefix
	remainder := sk[6:]
	parts := strings.Split(remainder, "#")
	
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("invalid tuple sort key format: expected 4 parts")
	}

	return parts[0], parts[1], parts[2], parts[3], nil
}

// IsValidEntityID checks if an entity ID is a valid ULID
func IsValidEntityID(entityID string) bool {
	// ULID is 26 characters long and uses base32 encoding
	if len(entityID) != 26 {
		return false
	}
	
	// Check if all characters are valid base32 (excluding I, L, O, U)
	for _, char := range entityID {
		if !isValidULIDChar(char) {
			return false
		}
	}
	
	return true
}

// isValidULIDChar checks if a character is valid in a ULID
func isValidULIDChar(c rune) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'H') ||
		(c >= 'J' && c <= 'K') ||
		(c >= 'M' && c <= 'N') ||
		(c >= 'P' && c <= 'T') ||
		(c >= 'V' && c <= 'Z')
}