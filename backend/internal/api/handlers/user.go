package handlers

import (
	"net/http"
	"time"

	"github.com/elibdev/notably/internal/api/models"
	"github.com/elibdev/notably/internal/repository"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userManager repository.UserManager
}

func NewUserHandler(userManager repository.UserManager) *UserHandler {
	return &UserHandler{userManager: userManager}
}

// GetCurrentUser godoc
// @Summary      Get current user information
// @Description  Retrieve the current authenticated user's profile information
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.UserInfoResponse  "User information"
// @Failure      401  {object}  models.ErrorResponse     "Unauthorized"
// @Failure      404  {object}  models.ErrorResponse     "User not found"
// @Router       /users/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := h.userManager.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}

	c.JSON(http.StatusOK, models.UserInfoResponse{
		UserID:    user.UserID,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.LastActive.Format(time.RFC3339),
	})
}

// UpdateCurrentUser godoc
// @Summary      Update current user profile
// @Description  Update the current authenticated user's profile information
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.SuccessResponse  "User updated successfully"
// @Failure      400  {object}  models.ErrorResponse    "Invalid request"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized"
// @Failure      501  {object}  models.ErrorResponse    "Not implemented"
// @Router       /users/me [put]
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	// In a real implementation, you'd update user profile information
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{Error: "Not implemented"})
}
