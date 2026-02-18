package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"listener-service/internal/config"
	"listener-service/internal/database"
	"listener-service/internal/events"
	"listener-service/internal/kafka"
	"listener-service/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	appLogger := logger.New(cfg.Environment)
	defer appLogger.Sync()

	appLogger.Info("🚀 Starting Listener Service (Legacy Mode)",
		zap.String("environment", cfg.Environment),
		zap.String("sqlite_path", cfg.SQLitePath),
		zap.Strings("kafka_brokers", cfg.KafkaBrokers),
		zap.String("kafka_group_id", cfg.KafkaGroupID),
	)

	appLogger.Info("📡 Kafka Configuration",
		zap.Strings("brokers", cfg.KafkaBrokers),
		zap.String("topic_items", cfg.KafkaTopicItems),
		zap.String("topic_stock", cfg.KafkaTopicStock),
		zap.String("group_id", cfg.KafkaGroupID),
		zap.Bool("auto_commit", cfg.KafkaAutoCommit),
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

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		appLogger.Fatal("Consumer error", zap.Error(err))
	case sig := <-quit:
		appLogger.Info("Shutting down listener service", zap.String("signal", sig.String()))
		cancel()
	}

	appLogger.Info("Listener service exited")
}

