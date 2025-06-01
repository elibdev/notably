package handlers

import (
	"net/http"
	"time"

	"github.com/elibdev/notably/internal/api/models"
	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
// @Summary      Health check
// @Description  Check if the service is healthy
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.HealthResponse  "Service is healthy"
// @Router       /health [get]
func HealthCheck(userManager repository.UserManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, models.HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().Format(time.RFC3339),
			Version:   "1.0.0",
		})
	}
}

// ReadinessCheck godoc
// @Summary      Readiness check
// @Description  Check if the service is ready to accept requests
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.HealthResponse  "Service is ready"
// @Failure      503  {object}  models.ErrorResponse   "Service not ready"
// @Router       /ready [get]
func ReadinessCheck(userManager repository.UserManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check database connectivity
		if err := userManager.Health(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
				Error: "database connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, models.HealthResponse{
			Status:    "ready",
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}
}
