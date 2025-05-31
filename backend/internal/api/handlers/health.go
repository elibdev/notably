package handlers

import (
	"net/http"

	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
)

func HealthCheck(userManager repository.UserManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "timedb",
			"version": "1.0.0",
		})
	}
}

func ReadinessCheck(userManager repository.UserManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check database connectivity
		if err := userManager.Health(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": "timedb",
		})
	}
}
