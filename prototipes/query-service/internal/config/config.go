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
	// SQLite Configuration (Read Model - same database as Listener Service)
	SQLitePath string
	// JWT Configuration
	JWTSecret string
	// Redis Configuration (optional - for cache)
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	CacheTTL      int  // Cache TTL in seconds
	UseCache      bool // Whether to use cache (Redis) or not
	// Kafka Configuration (for cache invalidation - optional)
	KafkaBrokers    []string
	KafkaTopicItems string
	KafkaTopicStock string
	KafkaGroupID    string
	KafkaAutoCommit bool
	UseKafka        bool // Whether to use Kafka for cache invalidation
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
		Port:        getEnv("PORT", "8081"),
		Environment: getEnv("ENVIRONMENT", "development"),
		// SQLite Configuration (Read Model - same database as Listener Service)
		SQLitePath: getEnv("SQLITE_PATH", "./inventory.db"),
		// JWT Configuration
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production-min-32-chars"),
		// Redis Configuration (optional)
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		CacheTTL:      getEnvAsInt("CACHE_TTL", 300),    // 5 minutes default
		UseCache:      getEnvAsBool("USE_CACHE", false), // Cache is optional, default false
		// Kafka Configuration (optional - for cache invalidation)
		KafkaBrokers:    kafkaBrokers,
		KafkaTopicItems: getEnv("KAFKA_TOPIC_ITEMS", "inventory.items"),
		KafkaTopicStock: getEnv("KAFKA_TOPIC_STOCK", "inventory.stock"),
		KafkaGroupID:    getEnv("KAFKA_GROUP_ID", "query-service"),
		KafkaAutoCommit: getEnvAsBool("KAFKA_AUTO_COMMIT", true),
		UseKafka:        getEnvAsBool("USE_KAFKA", false), // Kafka is optional, default false
	}
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
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
