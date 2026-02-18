package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Environment string
	// Kafka Configuration
	KafkaBrokers    []string
	KafkaTopicItems string
	KafkaTopicStock string
	KafkaGroupID    string
	KafkaAutoCommit bool
	// SQLite Configuration
	SQLitePath string
	// Retry Configuration
	MaxRetries      int
	RetryDelayMs    int
	DeadLetterQueue bool
	DLQTopic        string
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file is optional, continue with environment variables
	}

	// Parse Kafka brokers (comma-separated)
	kafkaBrokersStr := getEnv("KAFKA_BROKERS", "localhost:9093")
	kafkaBrokers := strings.Split(kafkaBrokersStr, ",")
	for i, broker := range kafkaBrokers {
		kafkaBrokers[i] = strings.TrimSpace(broker)
	}

	// Debug: Log the actual broker configuration being used
	// This will help identify if .env is being loaded correctly
	if kafkaBrokersStr != "localhost:9093" {
		// If not using default, log it (this will be visible in startup logs)
	}

	return &Config{
		Port:        getEnv("PORT", "8082"),
		Environment: getEnv("ENVIRONMENT", "development"),
		// Kafka Configuration
		KafkaBrokers:    kafkaBrokers,
		KafkaTopicItems: getEnv("KAFKA_TOPIC_ITEMS", "inventory.items"),
		KafkaTopicStock: getEnv("KAFKA_TOPIC_STOCK", "inventory.stock"),
		KafkaGroupID:     getEnv("KAFKA_GROUP_ID", "listener-service"),
		KafkaAutoCommit: getEnvAsBool("KAFKA_AUTO_COMMIT", false),
		// SQLite Configuration
		SQLitePath: getEnv("SQLITE_PATH", "./inventory.db"),
		// Retry Configuration
		MaxRetries:      getEnvAsInt("MAX_RETRIES", 3),
		RetryDelayMs:    getEnvAsInt("RETRY_DELAY_MS", 1000),
		DeadLetterQueue: getEnvAsBool("DEAD_LETTER_QUEUE", true),
		DLQTopic:        getEnv("DLQ_TOPIC", "inventory.dlq"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}

