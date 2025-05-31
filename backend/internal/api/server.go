package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"

	"github.com/elibdev/notably/internal/api/handlers"
	"github.com/elibdev/notably/internal/api/middleware"
	"github.com/elibdev/notably/internal/config"
	"github.com/elibdev/notably/internal/repository"
)

type Server struct {
	config      *config.Config
	userManager repository.UserManager
	router      *gin.Engine
	httpServer  *http.Server
}

func NewServer(cfg *config.Config, userManager repository.UserManager) *Server {
	return &Server{
		config:      cfg,
		userManager: userManager,
	}
}

func (s *Server) Start() error {
	s.setupRouter()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) setupRouter() {
	if s.config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()

	// Middleware
	s.router.Use(gin.Recovery())
	s.router.Use(logger.SetLogger())

	// CORS
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Configure for production
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Custom middleware
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.ErrorHandler())
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", handlers.HealthCheck(s.userManager))
	s.router.GET("/ready", handlers.ReadinessCheck(s.userManager))

	// API v1
	v1 := s.router.Group("/api/v1")

	// Authentication
	authHandler := handlers.NewAuthHandler(s.config.Auth, s.userManager)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/register", authHandler.Register)

	// Protected routes
	protected := v1.Group("")
	if s.config.Auth.RequireAuth {
		protected.Use(middleware.JWTAuth(s.config.Auth.JWTSecret))
	}
	protected.Use(middleware.UserContext(s.userManager))

	// User management
	userHandler := handlers.NewUserHandler(s.userManager)
	protected.GET("/users/me", userHandler.GetCurrentUser)
	protected.PUT("/users/me", userHandler.UpdateCurrentUser)

	// Table management
	tableHandler := handlers.NewTableHandler()
	protected.POST("/tables", tableHandler.CreateTable)
	protected.GET("/tables", tableHandler.ListTables)
	protected.GET("/tables/:tableId", tableHandler.GetTable)
	protected.PUT("/tables/:tableId", tableHandler.UpdateTable)
	protected.DELETE("/tables/:tableId", tableHandler.DeleteTable)

	// Entity management
	entityHandler := handlers.NewEntityHandler()
	entities := protected.Group("/tables/:tableId/entities")
	entities.POST("", entityHandler.CreateEntity)
	entities.GET("", entityHandler.ListEntities)
	entities.GET("/:entityId", entityHandler.GetEntity)
	entities.PUT("/:entityId", entityHandler.UpdateEntity)
	entities.DELETE("/:entityId", entityHandler.DeleteEntity)
	entities.POST("/:entityId/undelete", entityHandler.UndeleteEntity)
	entities.DELETE("/:entityId/fields/:fieldName", entityHandler.DeleteField)

	// History and time travel
	history := entities.Group("/:entityId/history")
	history.GET("", entityHandler.GetEntityHistory)

	tableHistory := protected.Group("/tables/:tableId/history")
	tableHistory.GET("", tableHandler.GetTableHistory)
	tableHistory.GET("/fields/:fieldName", tableHandler.GetFieldHistory)

	// Admin routes (include deleted entities)
	admin := protected.Group("/admin/tables/:tableId")
	admin.GET("/entities", entityHandler.ListEntitiesIncludingDeleted)

	// OpenAPI documentation
	s.router.Static("/docs", "./docs/swagger")
	s.router.GET("/openapi.json", handlers.OpenAPISpec)
}
