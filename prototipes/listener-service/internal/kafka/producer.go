package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"listener-service/internal/config"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Producer publishes confirmation events to Kafka
type Producer struct {
	producer sarama.SyncProducer
	logger   *zap.Logger
	config   *config.Config
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.Config, logger *zap.Logger) (*Producer, error) {
	logger.Info("🔌 Creating Kafka producer",
		zap.Strings("brokers", cfg.KafkaBrokers),
	)

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 3
	saramaConfig.Producer.Retry.Backoff = 100 * time.Millisecond
	saramaConfig.Version = sarama.V2_8_0_0

	// Network settings
	saramaConfig.Net.DialTimeout = 10 * time.Second
	saramaConfig.Net.ReadTimeout = 10 * time.Second
	saramaConfig.Net.WriteTimeout = 10 * time.Second

	producer, err := sarama.NewSyncProducer(cfg.KafkaBrokers, saramaConfig)
	if err != nil {
		logger.Error("❌ Failed to create Kafka producer",
			zap.Strings("brokers", cfg.KafkaBrokers),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	logger.Info("✅ Kafka producer created successfully",
		zap.Strings("brokers", cfg.KafkaBrokers),
	)

	return &Producer{
		producer: producer,
		logger:   logger,
		config:   cfg,
	}, nil
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.producer.Close()
}

// PublishConfirmationEvent publishes a confirmation event after processing
func (p *Producer) PublishConfirmationEvent(ctx context.Context, eventType string, itemID, sku string, data interface{}) error {
	// Create confirmation event
	confirmationEvent := map[string]interface{}{
		"eventType":   eventType + "Confirmed",
		"eventId":     uuid.New().String(),
		"aggregateId": itemID,
		"occurredAt":  time.Now().UTC().Format(time.RFC3339),
		"version":     1,
		"data":        data,
	}

	// Add SKU if available
	if sku != "" {
		confirmationEvent["sku"] = sku
	}

	// Serialize event
	eventData, err := json.Marshal(confirmationEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal confirmation event: %w", err)
	}

	// Determine topic based on event type
	topic := p.config.KafkaTopicStock
	if eventType == "InventoryItemCreated" || eventType == "InventoryItemUpdated" || eventType == "InventoryItemDeleted" {
		topic = p.config.KafkaTopicItems
	}

	// Create message
	message := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(itemID),
		Value: sarama.ByteEncoder(eventData),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event-type"),
				Value: []byte(eventType + "Confirmed"),
			},
		},
	}

	// Publish message
	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		p.logger.Error("Failed to publish confirmation event",
			zap.String("event_type", eventType+"Confirmed"),
			zap.String("topic", topic),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish confirmation event: %w", err)
	}

	p.logger.Info("Confirmation event published",
		zap.String("event_type", eventType+"Confirmed"),
		zap.String("topic", topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
		zap.String("item_id", itemID),
		zap.String("sku", sku),
	)

	return nil
}

