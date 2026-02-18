package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"listener-service/internal/config"
	"listener-service/internal/database"
	"listener-service/internal/events"
	"listener-service/internal/handlers"
	"listener-service/internal/kafka"
	"listener-service/pkg/logger"
	"listener-service/pkg/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "listener-service/docs" // Import docs for Swagger
)

// @title           Listener Service API
// @version         1.0
// @description     API de monitoreo para el Listener Service. Procesa eventos de Kafka y actualiza la base de datos SQLite.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  carlosand_01@hotmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8082
// @BasePath  /api/v1

// @schemes   http https
func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	appLogger := logger.New(cfg.Environment)
	defer appLogger.Sync()

	appLogger.Info("🚀 Starting Listener Service",
		zap.String("environment", cfg.Environment),
		zap.String("port", cfg.Port),
	)

	appLogger.Info("📡 Kafka Configuration",
		zap.Strings("brokers", cfg.KafkaBrokers),
		zap.String("topic_items", cfg.KafkaTopicItems),
		zap.String("topic_stock", cfg.KafkaTopicStock),
		zap.String("group_id", cfg.KafkaGroupID),
		zap.Bool("auto_commit", cfg.KafkaAutoCommit),
		zap.String("brokers_raw", strings.Join(cfg.KafkaBrokers, ",")),
	)

	appLogger.Info("💾 Database Configuration",
		zap.String("sqlite_path", cfg.SQLitePath),
	)

	// Initialize database (Single Writer)
	appLogger.Info("🔧 Initializing database...")
	db, err := database.NewSingleWriterDB(cfg, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()
	appLogger.Info("✅ Database initialized successfully")

	// Initialize Kafka producer for confirmation events
	appLogger.Info("🔧 Initializing Kafka producer for confirmation events...")
	producer, err := kafka.NewProducer(cfg, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize Kafka producer", zap.Error(err))
	}
	defer producer.Close()
	appLogger.Info("✅ Kafka producer initialized successfully")

	// Initialize event processor
	appLogger.Info("🔧 Initializing event processor...")
	processor := events.NewEventProcessor(db, producer, appLogger)
	appLogger.Info("✅ Event processor initialized successfully")

	// Initialize Kafka consumer
	appLogger.Info("🔧 Initializing Kafka consumer...")
	consumer, err := kafka.NewConsumer(cfg, processor, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize Kafka consumer", zap.Error(err))
	}
	defer consumer.Close()
	appLogger.Info("✅ Kafka consumer initialized successfully",
		zap.Strings("topics", []string{cfg.KafkaTopicItems, cfg.KafkaTopicStock}),
	)

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.New()
	router.Use(middleware.RecoveryHandler(appLogger))
	router.Use(logger.GinMiddleware(appLogger))
	router.Use(middleware.ErrorHandler(appLogger))

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Initialize handlers
	appLogger.Info("🔧 Initializing handlers...")
	monitoringHandler := handlers.NewMonitoringHandler(db, appLogger)
	appLogger.Info("✅ Handlers initialized successfully")

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint
		v1.GET("/health", healthCheck)

		// Monitoring endpoints
		monitoring := v1.Group("/monitoring")
		{
			monitoring.GET("/stats", monitoringHandler.GetStats)
			monitoring.GET("/database/status", monitoringHandler.GetDatabaseStatus)
		}
	}

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start HTTP server in a goroutine
	go func() {
		appLogger.Info("🌐 Starting HTTP server",
			zap.String("address", ":"+cfg.Port),
			zap.String("swagger_url", "http://localhost:"+cfg.Port+"/swagger/index.html"),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming Kafka messages in a goroutine
	errChan := make(chan error, 1)
	go func() {
		appLogger.Info("📨 Starting Kafka consumer...")
		if err := consumer.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		appLogger.Fatal("Consumer error", zap.Error(err))
	case sig := <-quit:
		appLogger.Info("Shutting down listener service", zap.String("signal", sig.String()))
		cancel()

		// Graceful shutdown HTTP server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			appLogger.Error("Server forced to shutdown", zap.Error(err))
		}
	}

	appLogger.Info("Listener service exited")
}

// healthCheck godoc
// @Summary      Health check endpoint
// @Description  Verifica el estado del servicio. Retorna el estado del servicio y su nombre.
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string  "Servicio operativo"
// @Router       /health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "listener-service",
	})
}
