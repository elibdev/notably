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
// @Summary      Create a new table
// @Description  Create a new table with field definitions
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        request  body      models.CreateTableRequest  true  "Table creation details"
// @Success      201      {object}  models.TableResponse       "Table created successfully"
// @Failure      400      {object}  models.ErrorResponse       "Invalid request"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      500      {object}  models.ErrorResponse       "Internal server error"
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
// @Summary      List all tables
// @Description  Get a list of all tables for the authenticated user
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.TableListResponse  "List of tables"
// @Failure      401  {object}  models.ErrorResponse      "Unauthorized"
// @Failure      500  {object}  models.ErrorResponse      "Internal server error"
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
// @Summary      Get table by ID
// @Description  Retrieve a specific table by its ID
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path      string  true  "Table ID"
// @Success      200      {object}  models.TableResponse  "Table information"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      404      {object}  models.ErrorResponse  "Table not found"
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
// @Summary      Get table history
// @Description  Retrieve the history of changes for a specific table
// @Tags         tables
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        tableId  path   string  true   "Table ID"
// @Param        since    query  string  false  "Start date for history (RFC3339 format)"
// @Param        limit    query  int     false  "Maximum number of history entries to return"
// @Success      200      {object}  models.TableHistoryResponse  "Table history"
// @Failure      401      {object}  models.ErrorResponse         "Unauthorized"
// @Failure      500      {object}  models.ErrorResponse         "Internal server error"
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
