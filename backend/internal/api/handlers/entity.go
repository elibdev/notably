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
// @Summary      Create a new entity with dynamic fields
// @Description  ## Create Entity
// @Description
// @Description  Creates a new entity in the specified table with flexible field values.
// @Description  The entity will automatically get a unique ID and timestamp. All changes
// @Description  are tracked in the history system for audit trails and time travel queries.
// @Description
// @Description  ### Field Value Types
// @Description  Entity fields can contain any JSON-compatible values:
// @Description  - **Strings**: Text, emails, descriptions
// @Description  - **Numbers**: Integers and floats
// @Description  - **Booleans**: true/false values
// @Description  - **Objects**: Nested JSON structures
// @Description  - **Arrays**: Lists of values
// @Description  - **Null**: Empty/undefined values
// @Description
// @Description  ### Example Request
// @Description  ```json
// @Description  {
// @Description    "fields": {
// @Description      "name": "John Doe",
// @Description      "email": "john.doe@company.com",
// @Description      "age": 30,
// @Description      "active": true,
// @Description      "metadata": {
// @Description        "department": "engineering",
// @Description        "level": "senior",
// @Description        "skills": ["Go", "React", "PostgreSQL"]
// @Description      },
// @Description      "hire_date": "2024-01-15T09:00:00Z"
// @Description    }
// @Description  }
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "entity_id": "01HWQR3K2N8X9YZ123ABC456EF",
// @Description    "table_id": "employees",
// @Description    "fields": {
// @Description      "name": "John Doe",
// @Description      "email": "john.doe@company.com",
// @Description      "age": 30,
// @Description      "active": true,
// @Description      "metadata": {
// @Description        "department": "engineering",
// @Description        "level": "senior"
// @Description      }
// @Description    },
// @Description    "is_deleted": false,
// @Description    "created_at": "2024-01-01T12:00:00Z",
// @Description    "timestamp": "2024-01-01T12:00:00Z"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Next Steps
// @Description  - Update entity: `PUT /tables/{tableId}/entities/{entityId}`
// @Description  - Query entity: `GET /tables/{tableId}/entities/{entityId}`
// @Description  - View history: `GET /tables/{tableId}/entities/{entityId}/history`
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path      string                    true  "Unique table identifier"
// @Param        request  body      models.CreateEntityRequest  true  "Entity data with flexible field values"
// @Success      201      {object}  models.EntityResponse     "Entity created successfully with generated ID"
// @Failure      400      {object}  models.ErrorResponse      "Invalid request data or field validation failed"
// @Failure      401      {object}  models.ErrorResponse      "Authentication required - missing or invalid Bearer token"
// @Failure      500      {object}  models.ErrorResponse      "Internal server error during entity creation"
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
// @Summary      Get all entities in a table with time travel support
// @Description  ## List Table Entities
// @Description
// @Description  Retrieves all active (non-deleted) entities from the specified table.
// @Description  Supports powerful time travel queries to see entities as they existed
// @Description  at any point in the past using the `asOf` parameter.
// @Description
// @Description  ### Time Travel Queries
// @Description  Use the `asOf` parameter to query historical data:
// @Description  - `asOf=2024-01-01T12:00:00Z` - See entities as they were at noon on Jan 1st
// @Description  - Without `asOf` - Get current state (most recent data)
// @Description
// @Description  ### Example Request (Current State)
// @Description  ```
// @Description  GET /tables/employees/entities?limit=10
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Example Request (Historical State)
// @Description  ```
// @Description  GET /tables/employees/entities?asOf=2024-01-01T12:00:00Z&limit=50
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "entities": [
// @Description      {
// @Description        "entity_id": "01HWQR3K2N8X9YZ123ABC456EF",
// @Description        "table_id": "employees",
// @Description        "fields": {
// @Description          "name": "John Doe",
// @Description          "email": "john.doe@company.com",
// @Description          "active": true
// @Description        },
// @Description        "is_deleted": false,
// @Description        "created_at": "2024-01-01T12:00:00Z",
// @Description        "timestamp": "2024-01-01T14:30:00Z"
// @Description      }
// @Description    ]
// @Description  }
// @Description  ```
// @Description
// @Description  ### Use Cases
// @Description  - Dashboard data display
// @Description  - Data export functionality
// @Description  - Historical analysis and reporting
// @Description  - Compliance audits with time travel
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path   string  true   "Unique table identifier"
// @Param        asOf     query  string  false  "Point-in-time query timestamp (RFC3339 format, e.g., 2024-01-01T12:00:00Z)"
// @Param        limit    query  int     false  "Maximum number of entities to return (default: 100, max: 1000)"
// @Success      200      {object}  models.EntityListResponse  "List of active entities, optionally filtered by time"
// @Failure      401      {object}  models.ErrorResponse       "Authentication required - missing or invalid Bearer token"
// @Failure      500      {object}  models.ErrorResponse       "Internal server error retrieving entities"
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
// @Summary      Get a specific entity with time travel support
// @Description  ## Get Entity Details
// @Description
// @Description  Retrieves a specific entity by its unique ID. Supports time travel
// @Description  queries to see the entity as it existed at any point in its history
// @Description  using the `asOf` parameter. Perfect for detailed views, editing forms,
// @Description  and historical analysis.
// @Description
// @Description  ### Time Travel Capability
// @Description  - **Current state**: No `asOf` parameter (most recent version)
// @Description  - **Historical state**: Add `asOf=2024-01-01T12:00:00Z` to see past version
// @Description  - **Deleted entities**: Can be retrieved with historical queries
// @Description
// @Description  ### Example Request (Current State)
// @Description  ```
// @Description  GET /tables/employees/entities/01HWQR3K2N8X9YZ123ABC456EF
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Example Request (Historical State)
// @Description  ```
// @Description  GET /tables/employees/entities/01HWQR3K2N8X9YZ123ABC456EF?asOf=2024-01-01T12:00:00Z
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "entity_id": "01HWQR3K2N8X9YZ123ABC456EF",
// @Description    "table_id": "employees",
// @Description    "fields": {
// @Description      "name": "John Doe",
// @Description      "email": "john.doe@company.com",
// @Description      "age": 30,
// @Description      "active": true,
// @Description      "metadata": {
// @Description        "department": "engineering",
// @Description        "level": "senior"
// @Description      }
// @Description    },
// @Description    "is_deleted": false,
// @Description    "created_at": "2024-01-01T12:00:00Z",
// @Description    "timestamp": "2024-01-01T14:30:00Z"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Use Cases
// @Description  - Entity detail views and editing forms
// @Description  - Historical data analysis
// @Description  - Audit trail investigation
// @Description  - Data recovery and comparison
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path   string  true   "Unique table identifier"
// @Param        entityId  path   string  true   "Unique entity identifier (ULID format)"
// @Param        asOf      query  string  false  "Point-in-time query timestamp (RFC3339 format, e.g., 2024-01-01T12:00:00Z)"
// @Success      200       {object}  models.EntityResponse  "Complete entity information, current or historical"
// @Failure      401       {object}  models.ErrorResponse   "Authentication required - missing or invalid Bearer token"
// @Failure      404       {object}  models.ErrorResponse   "Entity not found or access denied"
// @Failure      500       {object}  models.ErrorResponse   "Internal server error retrieving entity"
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
// @Summary      Update entity fields with full history tracking
// @Description  ## Update Entity
// @Description
// @Description  Updates an existing entity's fields while preserving complete history.
// @Description  All changes are automatically tracked for audit trails and time travel
// @Description  queries. You can update any combination of fields in a single request.
// @Description
// @Description  ### Update Behavior
// @Description  - **Partial updates**: Only specified fields are modified
// @Description  - **Field addition**: New fields are automatically added
// @Description  - **Field removal**: Set field to `null` to remove it
// @Description  - **History tracking**: All changes are logged with timestamps
// @Description  - **Atomic operation**: All field updates succeed or fail together
// @Description
// @Description  ### Example Request (Partial Update)
// @Description  ```json
// @Description  {
// @Description    "fields": {
// @Description      "email": "john.smith@newcompany.com",
// @Description      "active": false,
// @Description      "termination_date": "2024-02-01T17:00:00Z",
// @Description      "salary": null,
// @Description      "metadata": {
// @Description        "department": "operations",
// @Description        "exit_reason": "voluntary"
// @Description      }
// @Description    }
// @Description  }
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "entity_id": "01HWQR3K2N8X9YZ123ABC456EF",
// @Description    "table_id": "employees",
// @Description    "fields": {
// @Description      "name": "John Doe",
// @Description      "email": "john.smith@newcompany.com",
// @Description      "age": 30,
// @Description      "active": false,
// @Description      "termination_date": "2024-02-01T17:00:00Z",
// @Description      "metadata": {
// @Description        "department": "operations",
// @Description        "exit_reason": "voluntary"
// @Description      }
// @Description    },
// @Description    "is_deleted": false,
// @Description    "created_at": "2024-01-01T12:00:00Z",
// @Description    "timestamp": "2024-02-01T10:30:00Z"
// @Description  }
// @Description  ```
// @Description
// @Description  ### History Tracking
// @Description  Each field change creates a history entry viewable at:
// @Description  `GET /tables/{tableId}/entities/{entityId}/history`
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path      string                    true  "Unique table identifier"
// @Param        entityId  path      string                    true  "Unique entity identifier (ULID format)"
// @Param        request   body      models.UpdateEntityRequest  true  "Field updates (partial or complete)"
// @Success      200       {object}  models.EntityResponse     "Entity updated successfully with new timestamp"
// @Failure      400       {object}  models.ErrorResponse      "Invalid request data or field validation failed"
// @Failure      401       {object}  models.ErrorResponse      "Authentication required - missing or invalid Bearer token"
// @Failure      404       {object}  models.ErrorResponse      "Entity not found or access denied"
// @Failure      500       {object}  models.ErrorResponse      "Internal server error during update"
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
// @Summary      Get complete audit trail for entity changes
// @Description  ## Entity Change History
// @Description
// @Description  Retrieves the complete audit trail for a specific entity, showing
// @Description  every field change made over time. Essential for compliance, debugging,
// @Description  and understanding data evolution. Changes are returned in reverse
// @Description  chronological order (most recent first).
// @Description
// @Description  ### History Details
// @Description  Each history entry shows:
// @Description  - **Field name**: Which field was changed
// @Description  - **New value**: What the field was changed to
// @Description  - **Timestamp**: Exact time of the change
// @Description  - **Entity state**: Complete context of the change
// @Description
// @Description  ### Query Parameters
// @Description  - **limit**: Control number of results (default: 100, max: 1000)
// @Description  - **since**: Only show changes after this timestamp
// @Description
// @Description  ### Example Request
// @Description  ```
// @Description  GET /tables/employees/entities/01HWQR3K2N8X9YZ123ABC456EF/history?limit=20&since=2024-01-01T00:00:00Z
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "entities": [
// @Description      {
// @Description        "entity_id": "01HWQR3K2N8X9YZ123ABC456EF",
// @Description        "table_id": "employees",
// @Description        "field_name": "email",
// @Description        "value": "john.smith@newcompany.com",
// @Description        "timestamp": "2024-02-01T10:30:00Z",
// @Description        "user_id": "admin_user"
// @Description      },
// @Description      {
// @Description        "entity_id": "01HWQR3K2N8X9YZ123ABC456EF",
// @Description        "table_id": "employees",
// @Description        "field_name": "active",
// @Description        "value": "false",
// @Description        "timestamp": "2024-02-01T10:30:00Z",
// @Description        "user_id": "admin_user"
// @Description      }
// @Description    ]
// @Description  }
// @Description  ```
// @Description
// @Description  ### Use Cases
// @Description  - Compliance and audit requirements
// @Description  - Data change investigation
// @Description  - Recovery and rollback operations
// @Description  - User activity tracking
// @Tags         entities
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId   path   string  true   "Unique table identifier"
// @Param        entityId  path   string  true   "Unique entity identifier (ULID format)"
// @Param        limit     query  int     false  "Maximum number of history entries to return (default: 100, max: 1000)"
// @Param        since     query  string  false  "Start date for history (RFC3339 format, e.g., 2024-01-01T00:00:00Z)"
// @Success      200       {object}  models.EntityListResponse  "Complete entity change history"
// @Failure      401       {object}  models.ErrorResponse       "Authentication required - missing or invalid Bearer token"
// @Failure      404       {object}  models.ErrorResponse       "Entity not found or access denied"
// @Failure      500       {object}  models.ErrorResponse       "Internal server error retrieving history"
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
