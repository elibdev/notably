package handlers

import (
	"fmt"
	"net/http"
	"regexp"
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

var userIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)

func NewAuthHandler(authConfig config.AuthConfig, userManager repository.UserManager) *AuthHandler {
	return &AuthHandler{
		authConfig:  authConfig,
		userManager: userManager,
	}
}

// validateUserID checks if the user ID meets format requirements
func validateUserID(userID string) error {
	if len(userID) < 3 {
		return fmt.Errorf("user ID must be at least 3 characters long")
	}
	if len(userID) > 50 {
		return fmt.Errorf("user ID must be no more than 50 characters long")
	}
	if !userIDPattern.MatchString(userID) {
		return fmt.Errorf("user ID can only contain alphanumeric characters and underscores")
	}
	return nil
}

// Register godoc
// @Summary      Register a new user account
// @Description  ## Register New User
// @Description
// @Description  Creates a new user account in the Notably system. Upon successful registration,
// @Description  returns a JWT token that can be used immediately for authentication.
// @Description
// @Description  ### Requirements
// @Description  - **user_id**: Unique identifier (3-50 characters, alphanumeric and underscores)
// @Description  - **password**: Minimum 8 characters
// @Description  - **email**: Valid email address format
// @Description
// @Description  ### Example Request
// @Description  ```json
// @Description  {
// @Description    "user_id": "john_doe_2024",
// @Description    "password": "SecurePass123!",
// @Description    "email": "john.doe@company.com"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
// @Description    "expires_at": "2024-01-01T12:00:00Z",
// @Description    "user_id": "john_doe_2024"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Usage After Registration
// @Description  Include the token in subsequent API calls:
// @Description  ```
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "User registration details"
// @Success      201      {object}  AuthResponse     "User created successfully with JWT token"
// @Failure      400      {object}  models.ErrorResponse  "Invalid request data (missing fields, invalid email, weak password)"
// @Failure      409      {object}  models.ErrorResponse  "User with this user_id already exists"
// @Failure      500      {object}  models.ErrorResponse  "Internal server error during user creation"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Validate user ID format
	if err := validateUserID(req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Create user
	user, err := h.userManager.CreateUser(c.Request.Context(), req.UserID, req.Email, req.Password)
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
		UserID:    user.UserID,
	})
}

// Login godoc
// @Summary      Authenticate user and get access token
// @Description  ## User Authentication
// @Description
// @Description  Authenticates an existing user and returns a JWT token for accessing protected endpoints.
// @Description  The token is valid for a limited time and must be included in the Authorization header
// @Description  for all subsequent API calls.
// @Description
// @Description  ### Example Request
// @Description  ```json
// @Description  {
// @Description    "user_id": "john_doe_2024",
// @Description    "password": "SecurePass123!"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Example Success Response
// @Description  ```json
// @Description  {
// @Description    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
// @Description    "expires_at": "2024-01-01T12:00:00Z",
// @Description    "user_id": "john_doe_2024"
// @Description  }
// @Description  ```
// @Description
// @Description  ### Using the Token
// @Description  Add the token to the Authorization header for protected endpoints:
// @Description  ```
// @Description  Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
// @Description  ```
// @Description
// @Description  ### Token Expiration
// @Description  Tokens have a limited lifetime. When a token expires, you'll receive a 401 Unauthorized
// @Description  response and need to login again to get a new token.
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "User login credentials"
// @Success      200      {object}  AuthResponse  "Login successful with JWT token"
// @Failure      400      {object}  models.ErrorResponse  "Invalid request format or missing fields"
// @Failure      401      {object}  models.ErrorResponse  "Invalid user_id or password"
// @Failure      500      {object}  models.ErrorResponse  "Internal server error during authentication"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Verify user exists and password is correct
	user, err := h.userManager.GetUser(c.Request.Context(), req.UserID)
	if err != nil {
		logrus.WithError(err).WithField("user_id", req.UserID).Debug("User not found during login")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid credentials"})
		return
	}

	// Verify password
	isValidPassword, err := h.userManager.VerifyPassword(c.Request.Context(), req.UserID, req.Password)
	if err != nil {
		logrus.WithError(err).WithField("user_id", req.UserID).Error("Failed to verify password")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Authentication failed"})
		return
	}

	if !isValidPassword {
		logrus.WithField("user_id", req.UserID).Debug("Invalid password provided during login")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid credentials"})
		return
	}

	// Generate token
	token, expiresAt, err := h.generateToken(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    user.UserID,
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
