package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"query-service/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Cache defines the interface for cache operations
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	// DeleteByPattern deletes all keys matching a pattern (for cache invalidation)
	DeleteByPattern(ctx context.Context, pattern string) error
}

// RedisCache implements Cache using Redis
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
}

// InMemoryCache is a fallback implementation when Redis is not available
type InMemoryCache struct {
	logger *zap.Logger
	data   map[string]cacheEntry
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewCache creates a new cache instance (Redis or InMemory fallback)
func NewCache(cfg *config.Config, logger *zap.Logger) Cache {
	// Try to initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		// Connection pool settings for high performance
		PoolSize:     10, // Number of connections in the pool
		MinIdleConns: 5,  // Minimum number of idle connections
		// Timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		// Retry settings
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn("Failed to connect to Redis, using in-memory cache",
			zap.String("host", cfg.RedisHost),
			zap.String("port", cfg.RedisPort),
			zap.Error(err),
		)
		rdb.Close()
		// Fallback to in-memory cache
		return &InMemoryCache{
			logger: logger,
			data:   make(map[string]cacheEntry),
		}
	}

	logger.Info("Redis cache initialized successfully",
		zap.String("host", cfg.RedisHost),
		zap.String("port", cfg.RedisPort),
		zap.Int("db", cfg.RedisDB),
	)

	return &RedisCache{
		client: rdb,
		logger: logger,
	}
}

func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	entry, exists := c.data[key]
	if !exists {
		return nil, ErrCacheMiss
	}

	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return nil, ErrCacheMiss
	}

	return entry.value, nil
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}

func (c *InMemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	entry, exists := c.data[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return false, nil
	}

	return true, nil
}

func (c *InMemoryCache) DeleteByPattern(ctx context.Context, pattern string) error {
	// Simple pattern matching for in-memory cache
	// In production, this would use proper pattern matching
	for key := range c.data {
		// Simple prefix matching
		if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
			prefix := pattern[:len(pattern)-1]
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				delete(c.data, key)
			}
		} else if key == pattern {
			delete(c.data, key)
		}
	}
	return nil
}

// RedisCache implementation

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheMiss
	}
	if err != nil {
		c.logger.Warn("Redis Get error", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("redis get error: %w", err)
	}
	return val, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := c.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		c.logger.Warn("Redis Set error", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("redis set error: %w", err)
	}
	return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		c.logger.Warn("Redis Delete error", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("redis delete error: %w", err)
	}
	return nil
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		c.logger.Warn("Redis Exists error", zap.String("key", key), zap.Error(err))
		return false, fmt.Errorf("redis exists error: %w", err)
	}
	return count > 0, nil
}

// DeleteByPattern deletes all keys matching a pattern (for cache invalidation)
func (c *RedisCache) DeleteByPattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	keys := make([]string, 0)
	
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		c.logger.Warn("Redis Scan error", zap.String("pattern", pattern), zap.Error(err))
		return fmt.Errorf("redis scan error: %w", err)
	}
	
	if len(keys) > 0 {
		err := c.client.Del(ctx, keys...).Err()
		if err != nil {
			c.logger.Warn("Redis DeleteByPattern error", zap.String("pattern", pattern), zap.Error(err))
			return fmt.Errorf("redis delete by pattern error: %w", err)
		}
		c.logger.Debug("Deleted keys by pattern", zap.String("pattern", pattern), zap.Int("count", len(keys)))
	}
	
	return nil
}

// Helper functions for JSON serialization
func GetJSON(ctx context.Context, cache Cache, key string, dest interface{}) error {
	data, err := cache.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func SetJSON(ctx context.Context, cache Cache, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return cache.Set(ctx, key, data, ttl)
}

// TTL returns a time.Duration from seconds
func TTL(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)

