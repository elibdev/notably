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

func (h *TableHandler) CreateTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)

	var req models.CreateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, models.TableFromInternal(table))
}

func (h *TableHandler) ListTables(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)

	tables, err := userRepo.ListTables(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]models.TableResponse, len(tables))
	for i, table := range tables {
		response[i] = models.TableFromInternal(&table)
	}

	c.JSON(http.StatusOK, gin.H{"tables": response})
}

func (h *TableHandler) GetTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	table, err := userRepo.GetTable(c.Request.Context(), tableID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
		return
	}

	c.JSON(http.StatusOK, models.TableFromInternal(table))
}

func (h *TableHandler) UpdateTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	var req models.UpdateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing table
	table, err := userRepo.GetTable(c.Request.Context(), tableID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.TableFromInternal(table))
}

func (h *TableHandler) DeleteTable(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")

	err := userRepo.DeleteTable(c.Request.Context(), tableID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.TableHistoryFromInternal(history))
}

func (h *TableHandler) GetFieldHistory(c *gin.Context) {
	userRepo := c.MustGet("user_repo").(repository.UserRepository)
	tableID := c.Param("tableId")
	fieldName := c.Param("fieldName")

	opts := domainModels.QueryOptions{}
	history, err := userRepo.GetFieldHistory(c.Request.Context(), tableID, fieldName, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.FieldHistoryFromInternal(history))
}
