package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"command-service/internal/config"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// KafkaEventPublisher implements EventPublisher using Kafka
type KafkaEventPublisher struct {
	producer sarama.SyncProducer
	logger   *zap.Logger
	config   *config.Config
}

// NewKafkaEventPublisher creates a new Kafka event publisher
func NewKafkaEventPublisher(cfg *config.Config, logger *zap.Logger) (EventPublisher, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = cfg.KafkaRetries
	config.Producer.Idempotent = true
	config.Net.MaxOpenRequests = 1

	// Parse acks
	switch cfg.KafkaAcks {
	case "0":
		config.Producer.RequiredAcks = sarama.NoResponse
	case "1":
		config.Producer.RequiredAcks = sarama.WaitForLocal
	case "all":
		config.Producer.RequiredAcks = sarama.WaitForAll
	default:
		config.Producer.RequiredAcks = sarama.WaitForAll
	}

	producer, err := sarama.NewSyncProducer(cfg.KafkaBrokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &KafkaEventPublisher{
		producer: producer,
		logger:   logger,
		config:   cfg,
	}, nil
}

// Publish publishes an event to Kafka with retries and exponential backoff
func (p *KafkaEventPublisher) Publish(ctx context.Context, event interface{}) error {
	// Determine topic based on event type
	topic, err := p.getTopicForEvent(event)
	if err != nil {
		return fmt.Errorf("failed to determine topic: %w", err)
	}

	// Serialize event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message
	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(eventJSON),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event-type"),
				Value: []byte(p.getEventType(event)),
			},
			{
				Key:   []byte("event-id"),
				Value: []byte(uuid.New().String()),
			},
			{
				Key:   []byte("timestamp"),
				Value: []byte(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	}

	// Set partition key if available
	if partitionKey := p.getPartitionKey(event); partitionKey != "" {
		message.Key = sarama.StringEncoder(partitionKey)
	}

	// Retry with exponential backoff
	maxRetries := 3
	baseDelay := 100 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check context timeout
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		// Send message with timeout
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		done := make(chan error, 1)
		
		go func() {
			partition, offset, err := p.producer.SendMessage(message)
			if err != nil {
				done <- err
				return
			}
			p.logger.Info("Event published to Kafka",
				zap.String("topic", topic),
				zap.Int32("partition", partition),
				zap.Int64("offset", offset),
				zap.String("event-type", p.getEventType(event)),
				zap.Int("attempt", attempt+1),
			)
			done <- nil
		}()

		select {
		case err := <-done:
			cancel()
			if err == nil {
				return nil // Success
			}
			// Log error and retry
			p.logger.Warn("Failed to publish event to Kafka, retrying",
				zap.String("topic", topic),
				zap.Error(err),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
			)
		case <-sendCtx.Done():
			cancel()
			err := fmt.Errorf("timeout publishing event to Kafka: %w", sendCtx.Err())
			p.logger.Warn("Timeout publishing event to Kafka, retrying",
				zap.String("topic", topic),
				zap.Error(err),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries),
			)
		}

		// Exponential backoff before retry
		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(attempt)) // Exponential: 100ms, 200ms, 400ms
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	// All retries failed
	return fmt.Errorf("failed to publish event to Kafka after %d attempts", maxRetries)
}

// Close closes the Kafka producer
func (p *KafkaEventPublisher) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}

// getTopicForEvent determines the Kafka topic based on event type
func (p *KafkaEventPublisher) getTopicForEvent(event interface{}) (string, error) {
	switch event.(type) {
	case InventoryItemCreatedEvent, InventoryItemUpdatedEvent, InventoryItemDeletedEvent:
		return p.config.KafkaTopicItems, nil
	case StockAdjustedEvent, StockReservedEvent, StockReleasedEvent:
		return p.config.KafkaTopicStock, nil
	default:
		return "", fmt.Errorf("unknown event type: %T", event)
	}
}

// getEventType returns the event type as string
func (p *KafkaEventPublisher) getEventType(event interface{}) string {
	switch event.(type) {
	case InventoryItemCreatedEvent:
		return "InventoryItemCreated"
	case InventoryItemUpdatedEvent:
		return "InventoryItemUpdated"
	case InventoryItemDeletedEvent:
		return "InventoryItemDeleted"
	case StockAdjustedEvent:
		return "StockAdjusted"
	case StockReservedEvent:
		return "StockReserved"
	case StockReleasedEvent:
		return "StockReleased"
	default:
		return "Unknown"
	}
}

// getPartitionKey returns the partition key for the event (usually the item ID)
func (p *KafkaEventPublisher) getPartitionKey(event interface{}) string {
	switch e := event.(type) {
	case InventoryItemCreatedEvent:
		if id, ok := e.ItemID.(string); ok {
			return id
		}
		if id, ok := e.ItemID.(uuid.UUID); ok {
			return id.String()
		}
	case InventoryItemUpdatedEvent:
		if id, ok := e.ItemID.(string); ok {
			return id
		}
		if id, ok := e.ItemID.(uuid.UUID); ok {
			return id.String()
		}
	case InventoryItemDeletedEvent:
		if id, ok := e.ItemID.(string); ok {
			return id
		}
		if id, ok := e.ItemID.(uuid.UUID); ok {
			return id.String()
		}
	case StockAdjustedEvent:
		if id, ok := e.ItemID.(string); ok {
			return id
		}
		if id, ok := e.ItemID.(uuid.UUID); ok {
			return id.String()
		}
	case StockReservedEvent:
		if id, ok := e.ItemID.(string); ok {
			return id
		}
		if id, ok := e.ItemID.(uuid.UUID); ok {
			return id.String()
		}
	case StockReleasedEvent:
		if id, ok := e.ItemID.(string); ok {
			return id
		}
		if id, ok := e.ItemID.(uuid.UUID); ok {
			return id.String()
		}
	}
	return ""
}

