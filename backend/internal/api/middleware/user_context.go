package middleware

import (
	"net/http"
	"strings"

	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func UserContext(userManager repository.UserManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userIDStr string
		
		// First, check if user_id is already set by JWT middleware
		userID, exists := c.Get("user_id")
		if exists {
			var ok bool
			userIDStr, ok = userID.(string)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
				c.Abort()
				return
			}
		} else {
			// If not set, try to extract from Authorization header
			// This handles the case where auth is disabled but we still need the user ID
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
				c.Abort()
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
				c.Abort()
				return
			}

			tokenString := parts[1]
			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				// We need to get the JWT secret somehow - for now use a default
				// This should be passed as a parameter in a real implementation
				return []byte("local-development-secret"), nil
			})

			if err != nil || !token.Valid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				c.Abort()
				return
			}

			claims, ok := token.Claims.(*Claims)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
				c.Abort()
				return
			}

			userIDStr = claims.UserID
			c.Set("user_id", userIDStr)
		}

		// Get user repository
		userRepo := userManager.GetUserRepository(userIDStr)

		c.Set("user_repo", userRepo)
		c.Next()
	}
}
