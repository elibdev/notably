package handlers

import (
	"net/http"
	"time"

	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userManager repository.UserManager
}

func NewUserHandler(userManager repository.UserManager) *UserHandler {
	return &UserHandler{userManager: userManager}
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := h.userManager.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    user.UserID,
		"created_at": user.CreatedAt.Format(time.RFC3339),
		"updated_at": user.LastActive.Format(time.RFC3339),
	})
}

func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	// In a real implementation, you'd update user profile information
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}
