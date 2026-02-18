package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"query-service/internal/cache"
	"query-service/internal/config"
	"query-service/internal/models"
	"query-service/internal/repository"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Consumer represents a Kafka consumer for cache invalidation and update
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	cache         cache.Cache
	repository    repository.ReadRepository
	logger        *zap.Logger
	config        *config.Config
	topics        []string
	cacheTTL      time.Duration
}

// NewConsumer creates a new Kafka consumer for cache invalidation and update
func NewConsumer(cfg *config.Config, cacheClient cache.Cache, repo repository.ReadRepository, logger *zap.Logger) (*Consumer, error) {
	logger.Info("🔌 Creating Kafka consumer",
		zap.Strings("brokers", cfg.KafkaBrokers),
		zap.String("group_id", cfg.KafkaGroupID),
	)

	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Version = sarama.V2_8_0_0

	// Important: Configure Net settings to handle Docker hostname resolution
	// This ensures that even if Kafka returns internal hostnames, we can connect
	saramaConfig.Net.DialTimeout = 10 * time.Second
	saramaConfig.Net.ReadTimeout = 10 * time.Second
	saramaConfig.Net.WriteTimeout = 10 * time.Second

	// IMPORTANT: Sarama uses broker addresses from Kafka metadata directly.
	// If Kafka returns "kafka:9093" in metadata, Sarama will try to connect to that hostname.
	// This cannot be intercepted at the application level without modifying Sarama.
	// The solution is to configure Kafka correctly in Docker:
	//   KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9093
	//   KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9093

	// Disable metadata refresh that might cause hostname resolution issues
	// We'll use the initial broker list directly
	saramaConfig.Metadata.RefreshFrequency = 10 * time.Minute
	saramaConfig.Metadata.Retry.Max = 3
	saramaConfig.Metadata.Retry.Backoff = 250 * time.Millisecond

	consumerGroup, err := sarama.NewConsumerGroup(cfg.KafkaBrokers, cfg.KafkaGroupID, saramaConfig)
	if err != nil {
		logger.Error("❌ Failed to create Kafka consumer group",
			zap.Strings("brokers", cfg.KafkaBrokers),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	logger.Info("✅ Kafka consumer group created successfully",
		zap.Strings("brokers", cfg.KafkaBrokers),
		zap.String("group_id", cfg.KafkaGroupID),
	)

	topics := []string{cfg.KafkaTopicItems, cfg.KafkaTopicStock}

	return &Consumer{
		consumerGroup: consumerGroup,
		cache:         cacheClient,
		repository:    repo,
		logger:        logger,
		config:        cfg,
		topics:        topics,
		cacheTTL:      time.Duration(cfg.CacheTTL) * time.Second,
	}, nil
}

// Start starts consuming messages for cache invalidation and update
func (c *Consumer) Start(ctx context.Context) error {
	handler := &cacheInvalidationHandler{
		cache:      c.cache,
		repository: c.repository,
		logger:     c.logger,
		cacheTTL:   c.cacheTTL,
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
				c.logger.Error("Error from consumer",
					zap.Error(err),
					zap.String("error_type", fmt.Sprintf("%T", err)),
				)
				// Log additional details about the error
				if err != nil {
					c.logger.Error("Consumer error details",
						zap.String("error_string", err.Error()),
					)
				}
				return
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// Handle errors with improved diagnostics
	go func() {
		for err := range c.consumerGroup.Errors() {
			errStr := err.Error()
			// Check if error is related to hostname resolution
			if strings.Contains(errStr, "lookup kafka") || strings.Contains(errStr, "no such host") {
				c.logger.Error("❌ Consumer error: Kafka hostname resolution failed",
					zap.Error(err),
					zap.String("problem", "Kafka metadata returns 'kafka:9093' but this hostname cannot be resolved from the host"),
					zap.String("solution", "Update KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9093 in Docker"),
					zap.String("action", "Update docker-compose.yml and restart Kafka: docker-compose restart kafka"),
				)
			} else {
				c.logger.Error("Consumer error", zap.Error(err))
			}
		}
	}()

	c.logger.Info("✅ Kafka consumer started for cache invalidation",
		zap.Strings("topics", c.topics),
		zap.String("group_id", c.config.KafkaGroupID),
	)

	wg.Wait()
	return nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.consumerGroup.Close()
}

// cacheInvalidationHandler handles Kafka messages for cache invalidation and update
type cacheInvalidationHandler struct {
	cache      cache.Cache
	repository repository.ReadRepository
	logger     *zap.Logger
	cacheTTL   time.Duration
}

// Setup is run at the beginning of a new session
func (h *cacheInvalidationHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session
func (h *cacheInvalidationHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages for cache invalidation
func (h *cacheInvalidationHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// Extract event type from headers
			eventType := h.extractEventType(message.Headers)
			if eventType == "" {
				h.logger.Warn("Message without event type, skipping",
					zap.String("topic", message.Topic),
					zap.Int("partition", int(message.Partition)),
					zap.Int64("offset", message.Offset),
				)
				session.MarkMessage(message, "")
				continue
			}

			// Update or invalidate cache based on event type
			if err := h.updateOrInvalidateCache(context.Background(), eventType, message.Value); err != nil {
				h.logger.Error("Failed to update/invalidate cache",
					zap.String("event_type", eventType),
					zap.String("topic", message.Topic),
					zap.Error(err),
				)
			} else {
				h.logger.Debug("Cache updated/invalidated",
					zap.String("event_type", eventType),
					zap.String("topic", message.Topic),
				)
			}

			// Mark message as processed
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// extractEventType extracts event type from Kafka message headers
func (h *cacheInvalidationHandler) extractEventType(headers []*sarama.RecordHeader) string {
	for _, header := range headers {
		if string(header.Key) == "event-type" {
			return string(header.Value)
		}
	}
	return ""
}

// updateOrInvalidateCache updates or invalidates cache based on event type
// For confirmation events (ending with "Confirmed"), it updates Redis with new data
// For regular events, it invalidates cache
func (h *cacheInvalidationHandler) updateOrInvalidateCache(ctx context.Context, eventType string, eventData []byte) error {
	// Check if this is a confirmation event (from listener-service)
	isConfirmationEvent := strings.HasSuffix(eventType, "Confirmed")

	// Parse event data to extract item ID or SKU
	var itemID, sku string
	var confirmationData map[string]interface{}

	if len(eventData) > 0 {
		var eventDataMap map[string]interface{}
		if err := json.Unmarshal(eventData, &eventDataMap); err == nil {
			// For confirmation events, data is in the "data" field
			if isConfirmationEvent {
				if data, ok := eventDataMap["data"].(map[string]interface{}); ok {
					confirmationData = data
					if id, ok := data["itemId"].(string); ok {
						itemID = id
					}
					if s, ok := data["sku"].(string); ok {
						sku = s
					}
				}
			} else {
				// For regular events, data is directly in the event
				if id, ok := eventDataMap["itemId"].(string); ok {
					itemID = id
				}
				if s, ok := eventDataMap["sku"].(string); ok {
					sku = s
				}
			}
		}
	}

	// Handle confirmation events: Update Redis with new data
	if isConfirmationEvent && h.cache != nil && h.repository != nil {
		return h.updateCacheWithData(ctx, eventType, itemID, sku, confirmationData)
	}

	// Handle regular events: Invalidate cache
	return h.invalidateCache(ctx, eventType, itemID, sku)
}

// updateCacheWithData updates Redis cache with new data from confirmation events
func (h *cacheInvalidationHandler) updateCacheWithData(ctx context.Context, eventType string, itemID, sku string, data map[string]interface{}) error {
	if itemID == "" {
		h.logger.Warn("No item ID in confirmation event, skipping cache update",
			zap.String("event_type", eventType),
		)
		return nil
	}

	h.logger.Info("Updating Redis cache with new data",
		zap.String("event_type", eventType),
		zap.String("item_id", itemID),
		zap.String("sku", sku),
	)

	// Try to read updated data from repository
	itemUUID, err := uuid.Parse(itemID)
	if err != nil {
		h.logger.Warn("Invalid item ID in confirmation event", zap.String("item_id", itemID), zap.Error(err))
		// Fallback to data from event
		return h.updateCacheFromEventData(ctx, itemID, sku, data)
	}

	// Read from repository to get latest data
	item, err := h.repository.FindByID(ctx, itemUUID)
	if err != nil {
		h.logger.Warn("Failed to read item from repository, using event data",
			zap.String("item_id", itemID),
			zap.Error(err),
		)
		// Fallback to data from event
		return h.updateCacheFromEventData(ctx, itemID, sku, data)
	}

	// Update cache with item data
	if err := h.updateItemCache(ctx, item); err != nil {
		h.logger.Warn("Failed to update item cache", zap.String("item_id", itemID), zap.Error(err))
	}

	// Invalidate list cache to ensure fresh data
	if err := h.cache.DeleteByPattern(ctx, "items:list:*"); err != nil {
		h.logger.Warn("Failed to delete list cache by pattern", zap.Error(err))
	}

	h.logger.Debug("Cache updated successfully",
		zap.String("event_type", eventType),
		zap.String("item_id", itemID),
		zap.String("sku", sku),
	)

	return nil
}

// updateCacheFromEventData updates cache using data from the event
func (h *cacheInvalidationHandler) updateCacheFromEventData(ctx context.Context, itemID, sku string, data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("no data in confirmation event")
	}

	// Build item from event data
	item := &models.InventoryItem{
		ID: itemID,
	}

	if skuVal, ok := data["sku"].(string); ok {
		item.SKU = skuVal
	}
	if nameVal, ok := data["name"].(string); ok {
		item.Name = nameVal
	}
	if descVal, ok := data["description"].(string); ok {
		item.Description = descVal
	}
	if qtyVal, ok := data["quantity"].(float64); ok {
		item.Quantity = int(qtyVal)
	}
	if resVal, ok := data["reserved"].(float64); ok {
		item.Reserved = int(resVal)
	}
	if availVal, ok := data["available"].(float64); ok {
		item.Available = int(availVal)
	}

	return h.updateItemCache(ctx, item)
}

// updateItemCache updates cache with item data
func (h *cacheInvalidationHandler) updateItemCache(ctx context.Context, item *models.InventoryItem) error {
	// Serialize item
	itemJSON, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	// Update item by ID cache
	if err := h.cache.Set(ctx, fmt.Sprintf("item:id:%s", item.ID), itemJSON, h.cacheTTL); err != nil {
		h.logger.Warn("Failed to update item cache by ID", zap.String("item_id", item.ID), zap.Error(err))
	}

	// Update item by SKU cache
	if item.SKU != "" {
		if err := h.cache.Set(ctx, fmt.Sprintf("item:sku:%s", item.SKU), itemJSON, h.cacheTTL); err != nil {
			h.logger.Warn("Failed to update item cache by SKU", zap.String("sku", item.SKU), zap.Error(err))
		}
	}

	// Update stock status cache
	stockStatus := &models.StockStatus{
		ID:        item.ID,
		SKU:       item.SKU,
		Quantity:  item.Quantity,
		Reserved:  item.Reserved,
		Available: item.Available,
		UpdatedAt: item.UpdatedAt,
	}
	stockJSON, err := json.Marshal(stockStatus)
	if err == nil {
		if err := h.cache.Set(ctx, fmt.Sprintf("stock:%s", item.ID), stockJSON, h.cacheTTL); err != nil {
			h.logger.Warn("Failed to update stock cache", zap.String("item_id", item.ID), zap.Error(err))
		}
	}

	return nil
}

// invalidateCache invalidates cache based on event type
func (h *cacheInvalidationHandler) invalidateCache(ctx context.Context, eventType string, itemID, sku string) error {
	switch eventType {
	case "InventoryItemCreated", "InventoryItemUpdated", "InventoryItemDeleted",
		"StockAdjusted", "StockReserved", "StockReleased":
		// Fast cache invalidation strategy:
		// 1. Invalidate specific item cache keys (if item ID/SKU available)
		// 2. Invalidate related cache keys (list, stock status)
		// 3. This ensures data is quickly synchronized

		h.logger.Info("Invalidating inventory cache",
			zap.String("event_type", eventType),
			zap.String("item_id", itemID),
			zap.String("sku", sku),
		)

		// Invalidate specific item cache if ID is available
		if itemID != "" && h.cache != nil {
			// Delete item by ID cache
			if err := h.cache.Delete(ctx, fmt.Sprintf("item:id:%s", itemID)); err != nil {
				h.logger.Warn("Failed to delete item cache by ID", zap.String("item_id", itemID), zap.Error(err))
			}
			// Delete stock status cache
			if err := h.cache.Delete(ctx, fmt.Sprintf("stock:%s", itemID)); err != nil {
				h.logger.Warn("Failed to delete stock cache", zap.String("item_id", itemID), zap.Error(err))
			}
		}

		// Invalidate item by SKU cache if SKU is available
		if sku != "" && h.cache != nil {
			if err := h.cache.Delete(ctx, fmt.Sprintf("item:sku:%s", sku)); err != nil {
				h.logger.Warn("Failed to delete item cache by SKU", zap.String("sku", sku), zap.Error(err))
			}
		}

		// Invalidate list cache (all pages) for fast synchronization
		// This ensures that list queries get fresh data quickly
		if h.cache != nil {
			if err := h.cache.DeleteByPattern(ctx, "items:list:*"); err != nil {
				h.logger.Warn("Failed to delete list cache by pattern", zap.Error(err))
			}
		}

		// If we don't have specific item info, invalidate all item-related cache
		if itemID == "" && sku == "" && h.cache != nil {
			h.logger.Debug("No item ID/SKU found, invalidating all item cache")
			// Invalidate all item-related cache patterns
			patterns := []string{
				"item:id:*",
				"item:sku:*",
				"stock:*",
				"items:list:*",
			}
			for _, pattern := range patterns {
				if err := h.cache.DeleteByPattern(ctx, pattern); err != nil {
					h.logger.Warn("Failed to delete cache by pattern", zap.String("pattern", pattern), zap.Error(err))
				}
			}
		}

		h.logger.Debug("Cache invalidation completed",
			zap.String("event_type", eventType),
			zap.String("item_id", itemID),
			zap.String("sku", sku),
		)

		return nil
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}
}
