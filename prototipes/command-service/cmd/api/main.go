package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"command-service/internal/auth"
	"command-service/internal/config"
	"command-service/internal/handlers"
	"command-service/pkg/logger"
	"command-service/pkg/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "command-service/docs" // Import docs for Swagger
)

// @title           Command Service API
// @version         1.0
// @description     API de escritura para el sistema de inventario basado en arquitectura CQRS + EDA
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  carlosand_01@hotmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @schemes   http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token. Example: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	appLogger := logger.New(cfg.Environment)
	defer appLogger.Sync()

	appLogger.Info("🚀 Starting Command Service",
		zap.String("environment", cfg.Environment),
		zap.String("port", cfg.Port),
	)

	appLogger.Info("🔐 JWT Configuration",
		zap.Int("secret_length", len(cfg.JWTSecret)),
		zap.String("note", "Token expiration: 10 minutes"),
	)

	appLogger.Info("📡 Kafka Configuration",
		zap.Strings("brokers", cfg.KafkaBrokers),
		zap.String("topic_items", cfg.KafkaTopicItems),
		zap.String("topic_stock", cfg.KafkaTopicStock),
		zap.String("client_id", cfg.KafkaClientID),
		zap.String("acks", cfg.KafkaAcks),
		zap.Int("retries", cfg.KafkaRetries),
	)

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.New()
	
	// CORS middleware (must be first to handle preflight requests)
	router.Use(middleware.CORSMiddleware())
	
	router.Use(middleware.RecoveryHandler(appLogger))
	router.Use(logger.GinMiddleware(appLogger))
	
	// Request ID middleware (must be early in the chain)
	router.Use(middleware.RequestIDMiddleware(appLogger))
	
	// Initialize request ID store for idempotency
	appLogger.Info("🔧 Initializing request ID store for idempotency...")
	requestIDStore := middleware.NewInMemoryRequestIDStore()
	appLogger.Info("✅ Request ID store initialized successfully")
	
	// Idempotency middleware (for write operations)
	router.Use(middleware.IdempotencyMiddleware(requestIDStore, appLogger, 5*time.Minute))
	
	// Error handler middleware
	router.Use(middleware.ErrorHandler(appLogger))
	
	// Store response middleware (for idempotency)
	router.Use(middleware.StoreResponseMiddleware(requestIDStore, appLogger, 5*time.Minute))

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Initialize JWT manager
	appLogger.Info("🔧 Initializing JWT manager...")
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, appLogger)
	appLogger.Info("✅ JWT manager initialized successfully")

	// Initialize auth handler
	appLogger.Info("🔧 Initializing auth handler...")
	authHandler := auth.NewAuthHandler(jwtManager, appLogger)
	appLogger.Info("✅ Auth handler initialized successfully")

	// Initialize handlers
	appLogger.Info("🔧 Initializing handlers...")
	inventoryHandler := handlers.NewInventoryHandler(appLogger, cfg)
	appLogger.Info("✅ Handlers initialized successfully")

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint (public)
		v1.GET("/health", healthCheck)

		// Auth endpoints (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
		}

		// Protected endpoints (require JWT authentication)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(jwtManager, appLogger))
		{
			inventory := protected.Group("/inventory")
			{
				inventory.POST("/items", inventoryHandler.CreateItem)
				inventory.PUT("/items/:id", inventoryHandler.UpdateItem)
				inventory.DELETE("/items/:id", inventoryHandler.DeleteItem)
				inventory.POST("/items/:id/adjust", inventoryHandler.AdjustStock)
				inventory.POST("/items/:id/reserve", inventoryHandler.ReserveStock)
				inventory.POST("/items/:id/release", inventoryHandler.ReleaseStock)
			}
		}
	}

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		appLogger.Info("Starting command service",
			zap.String("port", cfg.Port),
			zap.String("environment", cfg.Environment),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	appLogger.Info("Server exited")
}

// healthCheck godoc
// @Summary      Health check endpoint
// @Description  Verifica el estado del servicio. Retorna el estado del servicio y su nombre.
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string  "Servicio operativo"
// @Router       /health [get]
// @Example      Valid response
//
//	{
//	  "status": "ok",
//	  "service": "command-service"
//	}
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "command-service",
	})
}
