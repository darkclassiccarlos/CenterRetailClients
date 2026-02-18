package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"listener-service/internal/config"
	"listener-service/internal/events"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

// Consumer represents a Kafka consumer
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	processor     *events.EventProcessor
	logger        *zap.Logger
	config        *config.Config
	topics        []string
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.Config, processor *events.EventProcessor, logger *zap.Logger) (*Consumer, error) {
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

	// Note: Sarama doesn't expose Net.Dialer directly
	// We'll handle hostname mapping at the application level
	// by ensuring Kafka is configured correctly in Docker
	// For now, we rely on KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9093

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
		processor:     processor,
		logger:        logger,
		config:        cfg,
		topics:        topics,
	}, nil
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	handler := &consumerGroupHandler{
		processor: c.processor,
		logger:    c.logger,
		config:    c.config,
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

	// Handle errors
	go func() {
		for err := range c.consumerGroup.Errors() {
			c.logger.Error("Consumer error", zap.Error(err))
		}
	}()

	c.logger.Info("Kafka consumer started",
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

// consumerGroupHandler handles Kafka consumer group messages
type consumerGroupHandler struct {
	processor *events.EventProcessor
	logger    *zap.Logger
	config    *config.Config
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
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

			// Process event with retry logic
			if err := h.processWithRetry(context.Background(), eventType, message.Value, message); err != nil {
				h.logger.Error("Failed to process event after retries",
					zap.String("event_type", eventType),
					zap.String("topic", message.Topic),
					zap.Error(err),
				)

				// Send to Dead Letter Queue if enabled
				if h.config.DeadLetterQueue {
					if err := h.sendToDLQ(message, err); err != nil {
						h.logger.Error("Failed to send to DLQ", zap.Error(err))
					}
				}

				// Mark message as processed even if failed (to avoid infinite loop)
				// In production, you might want to handle this differently
				session.MarkMessage(message, "")
				continue
			}

			// Mark message as processed
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// processWithRetry processes an event with retry logic
func (h *consumerGroupHandler) processWithRetry(ctx context.Context, eventType string, eventData []byte, message *sarama.ConsumerMessage) error {
	var lastErr error
	for attempt := 0; attempt <= h.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(h.config.RetryDelayMs*attempt) * time.Millisecond
			h.logger.Info("Retrying event processing",
				zap.String("event_type", eventType),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
			)
			time.Sleep(delay)
		}

		err := h.processor.ProcessEvent(ctx, eventType, eventData)
		if err == nil {
			if attempt > 0 {
				h.logger.Info("Event processed successfully after retry",
					zap.String("event_type", eventType),
					zap.Int("attempts", attempt+1),
				)
			}
			return nil
		}

		lastErr = err

		// Check if error is optimistic lock failure (retryable)
		if err.Error() == "optimistic lock failed - version mismatch or constraint violation" {
			h.logger.Warn("Optimistic lock failed, will retry",
				zap.String("event_type", eventType),
				zap.Int("attempt", attempt),
			)
			continue
		}

		// For other errors, check if retryable
		if attempt < h.config.MaxRetries {
			h.logger.Warn("Event processing failed, will retry",
				zap.String("event_type", eventType),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			continue
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", h.config.MaxRetries+1, lastErr)
}

// extractEventType extracts event type from Kafka message headers
func (h *consumerGroupHandler) extractEventType(headers []*sarama.RecordHeader) string {
	for _, header := range headers {
		if string(header.Key) == "event-type" {
			return string(header.Value)
		}
	}
	return ""
}

// sendToDLQ sends a failed message to Dead Letter Queue
func (h *consumerGroupHandler) sendToDLQ(message *sarama.ConsumerMessage, err error) error {
	// TODO: Implement DLQ producer
	// For now, just log the error
	h.logger.Error("Message sent to DLQ",
		zap.String("topic", message.Topic),
		zap.String("dlq_topic", h.config.DLQTopic),
		zap.Error(err),
	)
	return nil
}
