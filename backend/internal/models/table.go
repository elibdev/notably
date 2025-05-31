package models

import (
	"fmt"
	"time"
)

// DataType represents the supported field data types
type DataType string

const (
	DataTypeString    DataType = "string"
	DataTypeInt       DataType = "int"
	DataTypeFloat     DataType = "float"
	DataTypeBool      DataType = "bool"
	DataTypeDate      DataType = "date"
	DataTypeJSON      DataType = "json"
	DataTypeReference DataType = "reference"
)

// FieldDefinition represents a field schema within a table
type FieldDefinition struct {
	Name     string   `json:"name"`
	DataType DataType `json:"data_type"`
}

// TableSchema represents a table's structure and metadata
type TableSchema struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Fields    []FieldDefinition `json:"fields"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// TableHistory represents historical changes to a table
type TableHistory struct {
	TableID string  `json:"table_id"`
	Changes []Tuple `json:"changes"`
}

// Tuple represents a single change event in the time-series data
type Tuple struct {
	EntityID  string                 `json:"entity_id"`
	TableID   string                 `json:"table_id"`
	Operation string                 `json:"operation"` // CREATE, UPDATE, DELETE, UNDELETE
	Fields    map[string]interface{} `json:"fields,omitempty"`
	FieldName string                 `json:"field_name,omitempty"` // For field-specific operations
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
}

// ValidateDataType checks if a data type is supported
func (dt DataType) IsValid() bool {
	switch dt {
	case DataTypeString, DataTypeInt, DataTypeFloat, DataTypeBool, 
		 DataTypeDate, DataTypeJSON, DataTypeReference:
		return true
	default:
		return false
	}
}

// ValidateFields checks if all field definitions are valid
func (ts *TableSchema) ValidateFields() error {
	if len(ts.Fields) == 0 {
		return fmt.Errorf("table must have at least one field")
	}

	fieldNames := make(map[string]bool)
	for _, field := range ts.Fields {
		if field.Name == "" {
			return fmt.Errorf("field name cannot be empty")
		}
		if fieldNames[field.Name] {
			return fmt.Errorf("duplicate field name: %s", field.Name)
		}
		if !field.DataType.IsValid() {
			return fmt.Errorf("invalid data type: %s", field.DataType)
		}
		fieldNames[field.Name] = true
	}

	return nil
}