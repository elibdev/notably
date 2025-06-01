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
	EntityID  string                 `json:"entity_id" example:"entity_123"`
	TableID   string                 `json:"table_id" example:"users_table"`
	Fields    map[string]interface{} `json:"fields"`
	IsDeleted bool                   `json:"is_deleted" example:"false"`
	CreatedAt *string                `json:"created_at,omitempty" example:"2023-12-31T23:59:59Z"`
	DeletedAt *string                `json:"deleted_at,omitempty" example:"2023-12-31T23:59:59Z"`
	Timestamp string                 `json:"timestamp" example:"2023-12-31T23:59:59Z"`
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
