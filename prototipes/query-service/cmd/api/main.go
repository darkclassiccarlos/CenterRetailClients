package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"query-service/internal/auth"
	"query-service/internal/cache"
	"query-service/internal/config"
	"query-service/internal/handlers"
	"query-service/internal/kafka"
	"query-service/pkg/logger"
	"query-service/pkg/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "query-service/docs" // Import docs for Swagger
)

// @title           Query Service API
// @version         1.0
// @description     API de lectura para el sistema de inventario basado en arquitectura CQRS + EDA. Optimizado para baja latencia y alta escalabilidad.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  carlosand_01@hotmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8081
// @BasePath  /api/v1

// @schemes   http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token. Example: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

// Request ID Header
// @description All endpoints support X-Request-ID header for request tracking and correlation. If not provided, a new UUID will be generated and returned in the response header.
func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	appLogger := logger.New(cfg.Environment)
	defer appLogger.Sync()

	appLogger.Info("🚀 Starting Query Service",
		zap.String("environment", cfg.Environment),
		zap.String("port", cfg.Port),
	)

	appLogger.Info("💾 SQLite Configuration",
		zap.String("path", cfg.SQLitePath),
		zap.String("note", "Reading from same database as Listener Service"),
	)

	appLogger.Info("🔐 JWT Configuration",
		zap.Int("secret_length", len(cfg.JWTSecret)),
		zap.String("note", "Token expiration: 10 minutes"),
	)

	if cfg.UseCache {
		appLogger.Info("💾 Cache Configuration (Optional)",
			zap.String("redis_host", cfg.RedisHost),
			zap.String("redis_port", cfg.RedisPort),
			zap.Int("cache_ttl", cfg.CacheTTL),
			zap.Bool("enabled", cfg.UseCache),
		)
	} else {
		appLogger.Info("💾 Cache Configuration",
			zap.Bool("enabled", false),
			zap.String("note", "Cache is disabled (USE_CACHE=false)"),
		)
	}

	if cfg.UseKafka {
		appLogger.Info("📡 Kafka Configuration (Optional - for cache invalidation)",
			zap.Strings("brokers", cfg.KafkaBrokers),
			zap.String("topic_items", cfg.KafkaTopicItems),
			zap.String("topic_stock", cfg.KafkaTopicStock),
			zap.String("group_id", cfg.KafkaGroupID),
			zap.Bool("auto_commit", cfg.KafkaAutoCommit),
			zap.Bool("enabled", cfg.UseKafka),
		)
	} else {
		appLogger.Info("📡 Kafka Configuration",
			zap.Bool("enabled", false),
			zap.String("note", "Kafka is disabled (USE_KAFKA=false)"),
		)
	}

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

	// Initialize request ID store for idempotency (optional for read operations)
	appLogger.Info("🔧 Initializing request ID store for idempotency...")
	requestIDStore := middleware.NewInMemoryRequestIDStore()
	appLogger.Info("✅ Request ID store initialized successfully")

	// Idempotency middleware (mainly for consistency, query-service is mostly read-only)
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

	// Initialize cache (optional)
	var cacheClient cache.Cache
	if cfg.UseCache {
		appLogger.Info("🔧 Initializing cache (Redis)...")
		cacheClient = cache.NewCache(cfg, appLogger)
		appLogger.Info("✅ Cache initialized successfully")
	} else {
		appLogger.Info("⏭️  Skipping cache initialization (USE_CACHE=false)")
		cacheClient = nil
	}

	// Initialize handlers first (needed for Kafka consumer)
	appLogger.Info("🔧 Initializing handlers...")
	inventoryHandler, err := handlers.NewInventoryHandler(appLogger, cfg)
	if err != nil {
		appLogger.Fatal("Failed to initialize handlers", zap.Error(err))
	}
	appLogger.Info("✅ Handlers initialized successfully")

	// Initialize Kafka consumer for cache update/invalidation (optional)
	if cfg.UseKafka && cfg.UseCache {
		appLogger.Info("🔧 Initializing Kafka consumer for cache update/invalidation...")
		// Get repository from handler to pass to consumer
		repo := inventoryHandler.GetRepository()
		kafkaConsumer, err := kafka.NewConsumer(cfg, cacheClient, repo, appLogger)
		if err != nil {
			appLogger.Warn("Failed to initialize Kafka consumer, continuing without cache update/invalidation", zap.Error(err))
		} else {
			// Start Kafka consumer in background
			ctx, cancel := context.WithCancel(context.Background())
			defer func() {
				cancel()
				kafkaConsumer.Close()
			}()
			go func() {
				if err := kafkaConsumer.Start(ctx); err != nil {
					appLogger.Error("Kafka consumer error", zap.Error(err))
				}
			}()
			appLogger.Info("✅ Kafka consumer started for cache update/invalidation")
		}
	} else {
		if !cfg.UseKafka {
			appLogger.Info("⏭️  Skipping Kafka consumer (USE_KAFKA=false)")
		} else if !cfg.UseCache {
			appLogger.Info("⏭️  Skipping Kafka consumer (cache is disabled)")
		}
	}

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
				// Query endpoints
				inventory.GET("/items", inventoryHandler.ListItems)
				inventory.GET("/items/:id", inventoryHandler.GetItemByID)
				inventory.GET("/items/sku/:sku", inventoryHandler.GetItemBySKU)
				inventory.GET("/items/:id/stock", inventoryHandler.GetStockStatus)
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
		appLogger.Info("🌐 Starting HTTP server",
			zap.String("address", ":"+cfg.Port),
			zap.String("swagger_url", "http://localhost:"+cfg.Port+"/swagger/index.html"),
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
//	  "service": "query-service"
//	}
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "query-service",
	})
}
