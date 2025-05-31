package models

import (
	"time"

	"github.com/elibdev/notably/internal/models"
)

type CreateEntityRequest struct {
	Fields map[string]interface{} `json:"fields" binding:"required"`
}

type UpdateEntityRequest struct {
	Fields map[string]interface{} `json:"fields" binding:"required"`
}

type EntityResponse struct {
	EntityID  string                 `json:"entity_id"`
	TableID   string                 `json:"table_id"`
	Fields    map[string]interface{} `json:"fields"`
	IsDeleted bool                   `json:"is_deleted"`
	CreatedAt *string                `json:"created_at,omitempty"`
	DeletedAt *string                `json:"deleted_at,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// Conversion functions
func EntityFromInternal(entity *models.EntitySnapshot) EntityResponse {
	var createdAt, deletedAt *string

	if entity.CreatedAt != nil {
		created := entity.CreatedAt.Format(time.RFC3339)
		createdAt = &created
	}

	if entity.DeletedAt != nil {
		deleted := entity.DeletedAt.Format(time.RFC3339)
		deletedAt = &deleted
	}

	return EntityResponse{
		EntityID:  entity.EntityID,
		TableID:   entity.TableID,
		Fields:    entity.Fields,
		IsDeleted: entity.IsDeleted,
		CreatedAt: createdAt,
		DeletedAt: deletedAt,
		Timestamp: entity.Timestamp.Format(time.RFC3339),
	}
}

// Example usage and testing files
