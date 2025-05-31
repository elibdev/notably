package middleware

import (
	"net/http"

	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
)

func UserContext(userManager repository.UserManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
			c.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
			c.Abort()
			return
		}

		// Get user repository
		userRepo := userManager.GetUserRepository(userIDStr)

		c.Set("user_repo", userRepo)
		c.Next()
	}
}
