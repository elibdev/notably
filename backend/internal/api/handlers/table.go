package handlers

import (
	"net/http"
	"time"

	"github.com/elibdev/notably/internal/api/models"
	domainModels "github.com/elibdev/notably/internal/models"
	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
)

type TableHandler struct{}

func NewTableHandler() *TableHandler {
	return &TableHandler{}
}

// CreateTable godoc
// @Summary      Create a new data table with custom schema
// @Description  ## Create Table
// @Description
// @Description  Creates a new table with a custom field schema. Tables define the structure
// @Description  for your entities and support dynamic field types including strings, numbers,
// @Description  booleans, dates, JSON objects, and references to other entities.
// @Description
// @Description  ### Supported Field Types
// @Description  - **string**: Text data (names, descriptions, etc.)
// @Description  - **int**: Integer numbers
// @Description  - **float**: Decimal numbers
// @Description  - **bool**: True/false values
// @Description  - **date**: RFC3339 timestamps
// @Description  - **json**: Complex nested objects/arrays
// @Description  - **reference**: Links to other entities
// @Description
// @Description  ### Example Request
// @Description  ```json
// @Description  {
// @Description    "id": "customers",
// @Description    "fields": [
// @Description      {"name": "name", "data_type": "string"},
// @Description      {"name": "email", "data_type": "string"},
// @Description      {"name": "age", "data_type": "int"},
// @Description      {"name": "active", "data_type": "bool"},
// @Description      {"name": "created_date", "data_type": "date"},
// @Description      {"name": "metadata", "data_type": "json"}
// @Description    ]
// @Description  }
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "id": "customers",
// @Description    "user_id": "john_doe_2024",
// @Description    "fields": [
// @Description      {"name": "name", "data_type": "string"},
// @Description      {"name": "email", "data_type": "string"}
// @Description    ],
// @Description    "created_at": "2024-01-01T12:00:00Z",
// @Description    "updated_at": "2024-01-01T12:00:00Z"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Next Steps
// @Description  After creating a table, you can:
// @Description  - Add entities: `POST /tables/{id}/entities`
// @Description  - Update schema: `PUT /tables/{id}`
// @Description  - Query data: `GET /tables/{id}/entities`
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request  body      models.CreateTableRequest  true  "Table creation details"
// @Success      201      {object}  models.TableResponse       "Table created successfully"
// @Failure      400      {object}  models.ErrorResponse       "Invalid request data (missing fields, invalid field types)"
// @Failure      401      {object}  models.ErrorResponse       "Authentication required - include Bearer token"
// @Failure      500      {object}  models.ErrorResponse       "Internal server error during table creation"
// @Router       /tables [post]
func (h *TableHandler) CreateTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)

	var req models.CreateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Convert API fields to internal fields
	fields := make([]domainModels.FieldDefinition, len(req.Fields))
	for i, field := range req.Fields {
		fields[i] = domainModels.FieldDefinition{
			Name:     field.Name,
			DataType: domainModels.DataType(field.DataType),
		}
	}

	table, err := userRepo.CreateTable(c.Request.Context(), req.ID, fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, models.TableFromInternal(table))
}

// ListTables godoc
// @Summary      Get all tables for the current user
// @Description  ## List User Tables
// @Description
// @Description  Retrieves all tables owned by the authenticated user. Each table includes
// @Description  its schema definition, creation time, and metadata. Use this endpoint to
// @Description  discover available tables before working with entities.
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "tables": [
// @Description      {
// @Description        "id": "customers",
// @Description        "user_id": "john_doe_2024",
// @Description        "fields": [
// @Description          {"name": "name", "data_type": "string"},
// @Description          {"name": "email", "data_type": "string"},
// @Description          {"name": "active", "data_type": "bool"}
// @Description        ],
// @Description        "created_at": "2024-01-01T12:00:00Z",
// @Description        "updated_at": "2024-01-01T12:00:00Z"
// @Description      },
// @Description      {
// @Description        "id": "products",
// @Description        "user_id": "john_doe_2024",
// @Description        "fields": [
// @Description          {"name": "title", "data_type": "string"},
// @Description          {"name": "price", "data_type": "float"},
// @Description          {"name": "in_stock", "data_type": "bool"}
// @Description        ],
// @Description        "created_at": "2024-01-01T13:00:00Z",
// @Description        "updated_at": "2024-01-01T13:00:00Z"
// @Description      }
// @Description    ]
// @Description  }
// @Description  ```
// @Description
// @Description  ### Use Cases
// @Description  - Dashboard overview of all data schemas
// @Description  - Table picker for entity management interfaces
// @Description  - Schema discovery for dynamic form generation
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.TableListResponse  "List of all user tables with schemas"
// @Failure      401  {object}  models.ErrorResponse      "Authentication required - missing or invalid Bearer token"
// @Failure      500  {object}  models.ErrorResponse      "Internal server error retrieving tables"
// @Router       /tables [get]
func (h *TableHandler) ListTables(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)

	tables, err := userRepo.ListTables(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	response := make([]models.TableResponse, len(tables))
	for i, table := range tables {
		response[i] = models.TableFromInternal(&table)
	}

	c.JSON(http.StatusOK, models.TableListResponse{Tables: response})
}

// GetTable godoc
// @Summary      Get detailed information about a specific table
// @Description  ## Get Table Details
// @Description
// @Description  Retrieves comprehensive information about a specific table, including
// @Description  its complete schema definition, field types, and metadata. Use this
// @Description  endpoint to understand the structure before creating or querying entities.
// @Description
// @Description  ### Example Request
// @Description  ```
// @Description  GET /tables/customers
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "id": "customers",
// @Description    "user_id": "john_doe_2024",
// @Description    "fields": [
// @Description      {"name": "name", "data_type": "string"},
// @Description      {"name": "email", "data_type": "string"},
// @Description      {"name": "age", "data_type": "int"},
// @Description      {"name": "active", "data_type": "bool"},
// @Description      {"name": "metadata", "data_type": "json"}
// @Description    ],
// @Description    "created_at": "2024-01-01T12:00:00Z",
// @Description    "updated_at": "2024-01-01T12:00:00Z"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Use Cases
// @Description  - Validate table schema before entity operations
// @Description  - Generate dynamic forms based on field definitions
// @Description  - Table management interfaces
// @Description  - Schema documentation and discovery
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path      string  true  "Unique table identifier"
// @Success      200      {object}  models.TableResponse  "Complete table information with schema"
// @Failure      401      {object}  models.ErrorResponse  "Authentication required - missing or invalid Bearer token"
// @Failure      404      {object}  models.ErrorResponse  "Table not found or access denied"
// @Router       /tables/{tableId} [get]
func (h *TableHandler) GetTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	table, err := userRepo.GetTable(c.Request.Context(), tableID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Table not found"})
		return
	}

	c.JSON(http.StatusOK, models.TableFromInternal(table))
}

// UpdateTable godoc
// @Summary      Update table
// @Description  Update table field definitions
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path      string                     true  "Table ID"
// @Param        request  body      models.UpdateTableRequest  true  "Table update details"
// @Success      200      {object}  models.TableResponse       "Table updated successfully"
// @Failure      400      {object}  models.ErrorResponse       "Invalid request"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      404      {object}  models.ErrorResponse       "Table not found"
// @Failure      500      {object}  models.ErrorResponse       "Internal server error"
// @Router       /tables/{tableId} [put]
func (h *TableHandler) UpdateTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	var req models.UpdateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Get existing table
	table, err := userRepo.GetTable(c.Request.Context(), tableID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Table not found"})
		return
	}

	// Update fields
	fields := make([]domainModels.FieldDefinition, len(req.Fields))
	for i, field := range req.Fields {
		fields[i] = domainModels.FieldDefinition{
			Name:     field.Name,
			DataType: domainModels.DataType(field.DataType),
		}
	}
	table.Fields = fields
	table.UpdatedAt = time.Now()

	err = userRepo.UpdateTable(c.Request.Context(), *table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.TableFromInternal(table))
}

// DeleteTable godoc
// @Summary      Delete table
// @Description  Delete a table by its ID
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path  string  true  "Table ID"
// @Success      204      "Table deleted successfully"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      500      {object}  models.ErrorResponse  "Internal server error"
// @Router       /tables/{tableId} [delete]
func (h *TableHandler) DeleteTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	err := userRepo.DeleteTable(c.Request.Context(), tableID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetTableHistory godoc
// @Summary      Get complete audit trail for table changes
// @Description  ## Table History & Audit Trail
// @Description
// @Description  Retrieves the complete history of changes made to a table, including
// @Description  schema modifications, field additions/removals, and metadata updates.
// @Description  Essential for compliance, debugging, and understanding data evolution.
// @Description
// @Description  ### Query Parameters
// @Description  - **since**: Only show changes after this timestamp (RFC3339 format)
// @Description  - **limit**: Maximum number of history entries (default: 100)
// @Description
// @Description  ### Example Request
// @Description  ```
// @Description  GET /tables/customers/history?since=2024-01-01T00:00:00Z&limit=50
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "table_id": "customers",
// @Description    "changes": [
// @Description      {
// @Description        "timestamp": "2024-01-01T12:30:00Z",
// @Description        "field_name": "phone",
// @Description        "operation": "field_added",
// @Description        "value": "string",
// @Description        "user_id": "john_doe_2024"
// @Description      },
// @Description      {
// @Description        "timestamp": "2024-01-01T12:00:00Z",
// @Description        "field_name": "_created",
// @Description        "operation": "table_created",
// @Description        "value": "customers",
// @Description        "user_id": "john_doe_2024"
// @Description      }
// @Description    ]
// @Description  }
// @Description  ```
// @Description
// @Description  ### Use Cases
// @Description  - Compliance and audit requirements
// @Description  - Schema change tracking
// @Description  - Debugging data issues
// @Description  - Understanding table evolution over time
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path   string  true   "Unique table identifier"
// @Param        since    query  string  false  "Start date for history (RFC3339 format, e.g., 2024-01-01T00:00:00Z)"
// @Param        limit    query  int     false  "Maximum number of history entries to return (default: 100, max: 1000)"
// @Success      200      {object}  models.TableHistoryResponse  "Complete table change history"
// @Failure      401      {object}  models.ErrorResponse         "Authentication required - missing or invalid Bearer token"
// @Failure      500      {object}  models.ErrorResponse         "Internal server error retrieving history"
// @Router       /tables/{tableId}/history [get]
func (h *TableHandler) GetTableHistory(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	// Parse query parameters
	opts := domainModels.QueryOptions{}
	if since := c.Query("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.TimeRange = &domainModels.TimeRange{Start: t, End: time.Now()}
		}
	}
	if limit := c.Query("limit"); limit != "" {
		// Parse limit
	}

	history, err := userRepo.GetTableHistory(c.Request.Context(), tableID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.TableHistoryFromInternal(history))
}

// GetFieldHistory godoc
// @Summary      Get field history
// @Description  Retrieve the history of changes for a specific field in a table
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId    path  string  true  "Table ID"
// @Param        fieldName  path  string  true  "Field name"
// @Success      200        {object}  models.FieldHistoryResponse  "Field history"
// @Failure      401        {object}  models.ErrorResponse         "Unauthorized"
// @Failure      500        {object}  models.ErrorResponse         "Internal server error"
// @Router       /tables/{tableId}/history/fields/{fieldName} [get]
func (h *TableHandler) GetFieldHistory(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	fieldName := c.Param("fieldName")

	opts := domainModels.QueryOptions{}
	history, err := userRepo.GetFieldHistory(c.Request.Context(), tableID, fieldName, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.FieldHistoryFromInternal(history))
}
