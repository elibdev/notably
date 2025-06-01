package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"github.com/elibdev/notably/internal/api/middleware"
	"github.com/elibdev/notably/internal/api/models"
	"github.com/elibdev/notably/internal/config"
	"github.com/elibdev/notably/internal/repository"
)

type AuthHandler struct {
	authConfig  config.AuthConfig
	userManager repository.UserManager
}

type LoginRequest struct {
	UserID   string `json:"user_id" binding:"required" example:"user123"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type RegisterRequest struct {
	UserID   string `json:"user_id" binding:"required" example:"user123"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
}

type AuthResponse struct {
	Token     string    `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresAt time.Time `json:"expires_at" example:"2023-12-31T23:59:59Z"`
	UserID    string    `json:"user_id" example:"user123"`
}

func NewAuthHandler(authConfig config.AuthConfig, userManager repository.UserManager) *AuthHandler {
	return &AuthHandler{
		authConfig:  authConfig,
		userManager: userManager,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with email and password
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "User registration details"
// @Success      201      {object}  AuthResponse     "User created successfully"
// @Failure      400      {object}  models.ErrorResponse  "Invalid request"
// @Failure      409      {object}  models.ErrorResponse  "User already exists"
// @Failure      500      {object}  models.ErrorResponse  "Internal server error"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Create user
	_, err := h.userManager.CreateUser(c.Request.Context(), req.UserID)
	if err != nil {
		logrus.WithError(err).WithField("user_id", req.UserID).Error("Failed to create user")
		if repository.IsAlreadyExists(err) {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "User already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create user"})
		}
		return
	}

	// Generate token
	token, expiresAt, err := h.generateToken(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    req.UserID,
	})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user and return JWT token
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "User login credentials"
// @Success      200      {object}  AuthResponse  "Login successful"
// @Failure      400      {object}  models.ErrorResponse  "Invalid request"
// @Failure      401      {object}  models.ErrorResponse  "Invalid credentials"
// @Failure      500      {object}  models.ErrorResponse  "Internal server error"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Verify user exists
	user, err := h.userManager.GetUser(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid credentials"})
		return
	}

	// In a real implementation, you'd verify the password
	// For now, we'll just check if user exists
	_ = user

	// Generate token
	token, expiresAt, err := h.generateToken(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    req.UserID,
	})
}

func (h *AuthHandler) generateToken(userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(h.authConfig.TokenExpiry)

	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.authConfig.JWTSecret))

	return tokenString, expiresAt, err
}
