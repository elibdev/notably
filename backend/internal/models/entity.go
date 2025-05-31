package models

import (
	"time"
)

// EntitySnapshot represents a point-in-time view of an entity
type EntitySnapshot struct {
	EntityID  string                 `json:"entity_id"`
	TableID   string                 `json:"table_id"`
	UserID    string                 `json:"user_id"`
	Fields    map[string]interface{} `json:"fields"`
	IsDeleted bool                   `json:"is_deleted"`
	CreatedAt *time.Time             `json:"created_at,omitempty"`
	DeletedAt *time.Time             `json:"deleted_at,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// EntitiesSnapshot represents a collection of entities at a specific point in time
type EntitiesSnapshot struct {
	TableID   string           `json:"table_id"`
	Entities  []EntitySnapshot `json:"entities"`
	Timestamp time.Time        `json:"timestamp"`
}

// EntityHistory represents the complete history of changes to an entity
type EntityHistory struct {
	EntityID string  `json:"entity_id"`
	TableID  string  `json:"table_id"`
	Tuples   []Tuple `json:"tuples"`
}

// FieldHistory represents changes to a specific field across entities
type FieldHistory struct {
	TableID   string        `json:"table_id"`
	FieldName string        `json:"field_name"`
	Changes   []FieldChange `json:"changes"`
}

// FieldChange represents a single field modification
type FieldChange struct {
	EntityID  string     `json:"entity_id"`
	Timestamp time.Time  `json:"timestamp"`
	OldValue  *string    `json:"old_value,omitempty"`
	NewValue  string     `json:"new_value"`
	Operation string     `json:"operation"` // SET, DELETE
}

// QueryOptions provides filtering and pagination options for queries
type QueryOptions struct {
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
	TimeRange *TimeRange `json:"time_range,omitempty"`
	AsOf      *time.Time `json:"as_of,omitempty"`
}

// TimeRange specifies a time window for filtering
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// QueryResult provides paginated results with metadata
type QueryResult struct {
	Tuples  []Tuple `json:"tuples"`
	HasMore bool    `json:"has_more"`
	Total   *int    `json:"total,omitempty"`
}

// EntityQueryResult provides paginated entity results
type EntityQueryResult struct {
	Entities []EntitySnapshot `json:"entities"`
	HasMore  bool             `json:"has_more"`
	Total    *int             `json:"total,omitempty"`
}

// IsActive returns true if the entity is not deleted
func (e *EntitySnapshot) IsActive() bool {
	return !e.IsDeleted
}

// GetFieldValue safely retrieves a field value
func (e *EntitySnapshot) GetFieldValue(fieldName string) (interface{}, bool) {
	value, exists := e.Fields[fieldName]
	return value, exists
}

// SetFieldValue sets a field value
func (e *EntitySnapshot) SetFieldValue(fieldName string, value interface{}) {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[fieldName] = value
}

// DeleteField removes a field from the entity
func (e *EntitySnapshot) DeleteField(fieldName string) {
	if e.Fields != nil {
		delete(e.Fields, fieldName)
	}
}

// Clone creates a deep copy of the entity snapshot
func (e *EntitySnapshot) Clone() *EntitySnapshot {
	clone := &EntitySnapshot{
		EntityID:  e.EntityID,
		TableID:   e.TableID,
		UserID:    e.UserID,
		IsDeleted: e.IsDeleted,
		Timestamp: e.Timestamp,
		Fields:    make(map[string]interface{}),
	}

	if e.CreatedAt != nil {
		createdAt := *e.CreatedAt
		clone.CreatedAt = &createdAt
	}

	if e.DeletedAt != nil {
		deletedAt := *e.DeletedAt
		clone.DeletedAt = &deletedAt
	}

	// Deep copy fields
	for k, v := range e.Fields {
		clone.Fields[k] = v
	}

	return clone
}

// WithinTimeRange checks if the entity's timestamp falls within the specified range
func (e *EntitySnapshot) WithinTimeRange(tr *TimeRange) bool {
	if tr == nil {
		return true
	}
	return e.Timestamp.After(tr.Start) && e.Timestamp.Before(tr.End)
}