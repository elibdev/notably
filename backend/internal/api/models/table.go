package models

import (
	"time"

	"github.com/elibdev/notably/internal/models"
)

type FieldRequest struct {
	Name     string `json:"name" binding:"required"`
	DataType string `json:"data_type" binding:"required,oneof=string int float bool date json reference"`
}

type CreateTableRequest struct {
	ID     string         `json:"id" binding:"required"`
	Fields []FieldRequest `json:"fields" binding:"required,min=1"`
}

type UpdateTableRequest struct {
	Fields []FieldRequest `json:"fields" binding:"required,min=1"`
}

type FieldResponse struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

type TableResponse struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Fields    []FieldResponse `json:"fields"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type TableHistoryResponse struct {
	TableID string         `json:"table_id"`
	Changes []models.Tuple `json:"changes"`
}

type FieldHistoryResponse struct {
	TableID   string                `json:"table_id"`
	FieldName string                `json:"field_name"`
	Changes   []FieldChangeResponse `json:"changes"`
}

type FieldChangeResponse struct {
	EntityID  string  `json:"entity_id"`
	Timestamp string  `json:"timestamp"`
	OldValue  *string `json:"old_value,omitempty"`
	NewValue  string  `json:"new_value"`
}

// Conversion functions
func TableFromInternal(table *models.TableSchema) TableResponse {
	fields := make([]FieldResponse, len(table.Fields))
	for i, field := range table.Fields {
		fields[i] = FieldResponse{
			Name:     field.Name,
			DataType: string(field.DataType),
		}
	}

	return TableResponse{
		ID:        table.ID,
		UserID:    table.UserID,
		Fields:    fields,
		CreatedAt: table.CreatedAt.Format(time.RFC3339),
		UpdatedAt: table.UpdatedAt.Format(time.RFC3339),
	}
}

func TableHistoryFromInternal(history *models.TableHistory) TableHistoryResponse {
	return TableHistoryResponse{
		TableID: history.TableID,
		Changes: history.Changes,
	}
}

func FieldHistoryFromInternal(history *models.FieldHistory) FieldHistoryResponse {
	changes := make([]FieldChangeResponse, len(history.Changes))
	for i, change := range history.Changes {
		changes[i] = FieldChangeResponse{
			EntityID:  change.EntityID,
			Timestamp: change.Timestamp.Format(time.RFC3339),
			OldValue:  change.OldValue,
			NewValue:  change.NewValue,
		}
	}

	return FieldHistoryResponse{
		TableID:   history.TableID,
		FieldName: history.FieldName,
		Changes:   changes,
	}
}
