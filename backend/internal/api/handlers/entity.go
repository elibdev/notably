package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/elibdev/notably/internal/api/models"
	domainModels "github.com/elibdev/notably/internal/models"
	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
)

type EntityHandler struct{}

func NewEntityHandler() *EntityHandler {
	return &EntityHandler{}
}

func (h *EntityHandler) CreateEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	var req models.CreateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entity, err := userRepo.CreateEntity(c.Request.Context(), tableID, req.Fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, models.EntityFromInternal(entity))
}

func (h *EntityHandler) ListEntities(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	// Parse query parameters
	var asOf *time.Time
	if asOfStr := c.Query("asOf"); asOfStr != "" {
		if t, err := time.Parse(time.RFC3339, asOfStr); err == nil {
			asOf = &t
		}
	}

	snapshot, err := userRepo.GetAllEntities(c.Request.Context(), tableID, asOf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entities := make([]models.EntityResponse, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		entities = append(entities, models.EntityFromInternal(&entity))
	}

	c.JSON(http.StatusOK, gin.H{
		"entities":  entities,
		"table_id":  snapshot.TableID,
		"timestamp": snapshot.Timestamp.Format(time.RFC3339),
	})
}

func (h *EntityHandler) GetEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	// Parse query parameters
	var asOf *time.Time
	if asOfStr := c.Query("asOf"); asOfStr != "" {
		if t, err := time.Parse(time.RFC3339, asOfStr); err == nil {
			asOf = &t
		}
	}

	entity, err := userRepo.GetEntity(c.Request.Context(), tableID, entityID, asOf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if entity == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	c.JSON(http.StatusOK, models.EntityFromInternal(entity))
}

func (h *EntityHandler) UpdateEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	var req models.UpdateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entity, err := userRepo.UpdateEntity(c.Request.Context(), tableID, entityID, req.Fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.EntityFromInternal(entity))
}

func (h *EntityHandler) DeleteEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	err := userRepo.DeleteEntity(c.Request.Context(), tableID, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *EntityHandler) UndeleteEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	err := userRepo.UndeleteEntity(c.Request.Context(), tableID, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *EntityHandler) DeleteField(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")
	fieldName := c.Param("fieldName")

	err := userRepo.DeleteField(c.Request.Context(), tableID, entityID, fieldName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *EntityHandler) GetEntityHistory(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	// Parse query parameters
	opts := domainModels.QueryOptions{}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	result, err := userRepo.GetEntityHistory(c.Request.Context(), tableID, entityID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tuples":   result.Tuples,
		"has_more": result.HasMore,
	})
}

func (h *EntityHandler) ListEntitiesIncludingDeleted(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	snapshot, err := userRepo.GetAllEntitiesIncludingDeleted(c.Request.Context(), tableID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entities := make([]models.EntityResponse, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		entities = append(entities, models.EntityFromInternal(&entity))
	}

	c.JSON(http.StatusOK, gin.H{
		"entities":  entities,
		"table_id":  snapshot.TableID,
		"timestamp": snapshot.Timestamp.Format(time.RFC3339),
	})
}
