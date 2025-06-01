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

// CreateEntity godoc
// @Summary      Create a new entity
// @Description  Create a new entity in the specified table
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path      string                    true  "Table ID"
// @Param        request  body      models.CreateEntityRequest  true  "Entity creation details"
// @Success      201      {object}  models.EntityResponse     "Entity created successfully"
// @Failure      400      {object}  models.ErrorResponse      "Invalid request"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized"
// @Failure      500      {object}  models.ErrorResponse      "Internal server error"
// @Router       /tables/{tableId}/entities [post]
func (h *EntityHandler) CreateEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	var req models.CreateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	entity, err := userRepo.CreateEntity(c.Request.Context(), tableID, req.Fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, models.EntityFromInternal(entity))
}

// ListEntities godoc
// @Summary      List entities in a table
// @Description  Get a list of all entities in the specified table with optional time-based filtering
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path   string  true   "Table ID"
// @Param        asOf     query  string  false  "Point-in-time query (RFC3339 format)"
// @Param        limit    query  int     false  "Maximum number of entities to return"
// @Success      200      {object}  models.EntityListResponse  "List of entities"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      500      {object}  models.ErrorResponse       "Internal server error"
// @Router       /tables/{tableId}/entities [get]
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

// GetEntity godoc
// @Summary      Get entity by ID
// @Description  Retrieve a specific entity by its ID with optional point-in-time query
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path   string  true   "Table ID"
// @Param        entityId  path   string  true   "Entity ID"
// @Param        asOf      query  string  false  "Point-in-time query (RFC3339 format)"
// @Success      200       {object}  models.EntityResponse  "Entity information"
// @Failure      401       {object}  models.ErrorResponse   "Unauthorized"
// @Failure      404       {object}  models.ErrorResponse   "Entity not found"
// @Failure      500       {object}  models.ErrorResponse   "Internal server error"
// @Router       /tables/{tableId}/entities/{entityId} [get]
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
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Entity not found"})
		return
	}

	if entity == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
		return
	}

	c.JSON(http.StatusOK, models.EntityFromInternal(entity))
}

// UpdateEntity godoc
// @Summary      Update entity
// @Description  Update an existing entity's fields
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path      string                    true  "Table ID"
// @Param        entityId  path      string                    true  "Entity ID"
// @Param        request   body      models.UpdateEntityRequest  true  "Entity update details"
// @Success      200       {object}  models.EntityResponse     "Entity updated successfully"
// @Failure      400       {object}  models.ErrorResponse      "Invalid request"
// @Failure      401       {object}  models.ErrorResponse      "Unauthorized"
// @Failure      404       {object}  models.ErrorResponse      "Entity not found"
// @Failure      500       {object}  models.ErrorResponse      "Internal server error"
// @Router       /tables/{tableId}/entities/{entityId} [put]
func (h *EntityHandler) UpdateEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	var req models.UpdateEntityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	entity, err := userRepo.UpdateEntity(c.Request.Context(), tableID, entityID, req.Fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.EntityFromInternal(entity))
}

// DeleteEntity godoc
// @Summary      Delete entity
// @Description  Mark an entity as deleted (soft delete)
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path  string  true  "Table ID"
// @Param        entityId  path  string  true  "Entity ID"
// @Success      204       "Entity deleted successfully"
// @Failure      401       {object}  models.ErrorResponse  "Unauthorized"
// @Failure      404       {object}  models.ErrorResponse  "Entity not found"
// @Failure      500       {object}  models.ErrorResponse  "Internal server error"
// @Router       /tables/{tableId}/entities/{entityId} [delete]
func (h *EntityHandler) DeleteEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	err := userRepo.DeleteEntity(c.Request.Context(), tableID, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UndeleteEntity godoc
// @Summary      Undelete entity
// @Description  Restore a previously deleted entity
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path  string  true  "Table ID"
// @Param        entityId  path  string  true  "Entity ID"
// @Success      204       "Entity undeleted successfully"
// @Failure      401       {object}  models.ErrorResponse  "Unauthorized"
// @Failure      404       {object}  models.ErrorResponse  "Entity not found"
// @Failure      500       {object}  models.ErrorResponse  "Internal server error"
// @Router       /tables/{tableId}/entities/{entityId}/undelete [post]
func (h *EntityHandler) UndeleteEntity(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	entityID := c.Param("entityId")

	err := userRepo.UndeleteEntity(c.Request.Context(), tableID, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetEntityHistory godoc
// @Summary      Get entity history
// @Description  Retrieve the history of changes for a specific entity
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path   string  true   "Table ID"
// @Param        entityId  path   string  true   "Entity ID"
// @Param        limit     query  int     false  "Maximum number of history entries to return"
// @Param        since     query  string  false  "Start date for history (RFC3339 format)"
// @Success      200       {object}  models.EntityListResponse  "Entity history"
// @Failure      401       {object}  models.ErrorResponse       "Unauthorized"
// @Failure      404       {object}  models.ErrorResponse       "Entity not found"
// @Failure      500       {object}  models.ErrorResponse       "Internal server error"
// @Router       /tables/{tableId}/entities/{entityId}/history [get]
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
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	entities := make([]models.EntityResponse, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		entities = append(entities, models.EntityFromInternal(&entity))
	}

	c.JSON(http.StatusOK, models.EntityListResponse{Entities: entities})
}
