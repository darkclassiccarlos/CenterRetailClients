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
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	// JWT Configuration
	JWTSecret string
	// Kafka Configuration
	KafkaBrokers    []string
	KafkaTopicItems string
	KafkaTopicStock string
	KafkaClientID   string
	KafkaAcks       string
	KafkaRetries    int
	KafkaBatchSize  int
	KafkaLingerMs   int
}

func Load() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Parse Kafka brokers (comma-separated)
	kafkaBrokersStr := getEnv("KAFKA_BROKERS", "localhost:9093")
	kafkaBrokers := strings.Split(kafkaBrokersStr, ",")
	for i, broker := range kafkaBrokers {
		kafkaBrokers[i] = strings.TrimSpace(broker)
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", "postgres"),
		DBName:      getEnv("DB_NAME", "inventory_db"),
		// JWT Configuration
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production-min-32-chars"),
		// Kafka Configuration
		KafkaBrokers:    kafkaBrokers,
		KafkaTopicItems: getEnv("KAFKA_TOPIC_ITEMS", "inventory.items"),
		KafkaTopicStock: getEnv("KAFKA_TOPIC_STOCK", "inventory.stock"),
		KafkaClientID:   getEnv("KAFKA_CLIENT_ID", "command-service"),
		KafkaAcks:       getEnv("KAFKA_ACKS", "all"),
		KafkaRetries:    getEnvAsInt("KAFKA_RETRIES", 3),
		KafkaBatchSize:  getEnvAsInt("KAFKA_BATCH_SIZE", 16384),
		KafkaLingerMs:   getEnvAsInt("KAFKA_LINGER_MS", 10),
	}
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
