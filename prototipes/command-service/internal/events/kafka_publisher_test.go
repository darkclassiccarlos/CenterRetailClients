package events

import (
	"context"
	"testing"
	"time"

	"command-service/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Note: MockSaramaSyncProducer is not used in current tests
// In a real implementation, we would need to mock sarama.SyncProducer
// For now, we test the event type mapping and topic selection logic

func TestKafkaEventPublisher_Publish_InventoryItemCreatedEvent(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	cfg := &config.Config{
		KafkaTopicItems: "inventory.items",
		KafkaTopicStock: "inventory.stock",
	}

	// Note: In a real test, we would need to mock sarama.NewSyncProducer
	// For now, we'll test the event type mapping logic
	event := InventoryItemCreatedEvent{
		ItemID:      uuid.New(),
		SKU:         "SKU-001",
		Name:        "Test Item",
		Description: "Description",
		Quantity:    100,
		OccurredAt:  time.Now(),
	}

	// Test event type
	publisher := &KafkaEventPublisher{
		logger: logger,
		config: cfg,
	}

	eventType := publisher.getEventType(event)
	assert.Equal(t, "InventoryItemCreated", eventType)

	topic, err := publisher.getTopicForEvent(event)
	assert.NoError(t, err)
	assert.Equal(t, "inventory.items", topic)
}

func TestKafkaEventPublisher_Publish_StockAdjustedEvent(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	cfg := &config.Config{
		KafkaTopicItems: "inventory.items",
		KafkaTopicStock: "inventory.stock",
	}

	event := StockAdjustedEvent{
		ItemID:     uuid.New(),
		SKU:        "SKU-001",
		Quantity:   10,
		NewTotal:   110,
		OccurredAt: time.Now(),
	}

	// Test event type
	publisher := &KafkaEventPublisher{
		logger: logger,
		config: cfg,
	}

	eventType := publisher.getEventType(event)
	assert.Equal(t, "StockAdjusted", eventType)

	topic, err := publisher.getTopicForEvent(event)
	assert.NoError(t, err)
	assert.Equal(t, "inventory.stock", topic)
}

func TestKafkaEventPublisher_GetEventType_AllTypes(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		KafkaTopicItems: "inventory.items",
		KafkaTopicStock: "inventory.stock",
	}

	publisher := &KafkaEventPublisher{
		logger: logger,
		config: cfg,
	}

	testCases := []struct {
		name     string
		event    interface{}
		expected string
	}{
		{"InventoryItemCreated", InventoryItemCreatedEvent{}, "InventoryItemCreated"},
		{"InventoryItemUpdated", InventoryItemUpdatedEvent{}, "InventoryItemUpdated"},
		{"InventoryItemDeleted", InventoryItemDeletedEvent{}, "InventoryItemDeleted"},
		{"StockAdjusted", StockAdjustedEvent{}, "StockAdjusted"},
		{"StockReserved", StockReservedEvent{}, "StockReserved"},
		{"StockReleased", StockReleasedEvent{}, "StockReleased"},
		{"Unknown", "unknown", "Unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := publisher.getEventType(tc.event)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestKafkaEventPublisher_GetTopicForEvent_AllTypes(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		KafkaTopicItems: "inventory.items",
		KafkaTopicStock: "inventory.stock",
	}

	publisher := &KafkaEventPublisher{
		logger: logger,
		config: cfg,
	}

	testCases := []struct {
		name        string
		event       interface{}
		expected    string
		expectError bool
	}{
		{"InventoryItemCreated", InventoryItemCreatedEvent{}, "inventory.items", false},
		{"InventoryItemUpdated", InventoryItemUpdatedEvent{}, "inventory.items", false},
		{"InventoryItemDeleted", InventoryItemDeletedEvent{}, "inventory.items", false},
		{"StockAdjusted", StockAdjustedEvent{}, "inventory.stock", false},
		{"StockReserved", StockReservedEvent{}, "inventory.stock", false},
		{"StockReleased", StockReleasedEvent{}, "inventory.stock", false},
		{"Unknown", "unknown", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			topic, err := publisher.getTopicForEvent(tc.event)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, topic)
			}
		})
	}
}

func TestKafkaEventPublisher_GetPartitionKey_UUID(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		KafkaTopicItems: "inventory.items",
		KafkaTopicStock: "inventory.stock",
	}

	publisher := &KafkaEventPublisher{
		logger: logger,
		config: cfg,
	}

	itemID := uuid.New()
	event := InventoryItemCreatedEvent{
		ItemID: itemID,
		SKU:    "SKU-001",
	}

	partitionKey := publisher.getPartitionKey(event)
	assert.Equal(t, itemID.String(), partitionKey)
}

func TestKafkaEventPublisher_GetPartitionKey_String(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		KafkaTopicItems: "inventory.items",
		KafkaTopicStock: "inventory.stock",
	}

	publisher := &KafkaEventPublisher{
		logger: logger,
		config: cfg,
	}

	itemIDStr := uuid.New().String()
	event := InventoryItemCreatedEvent{
		ItemID: itemIDStr,
		SKU:    "SKU-001",
	}

	partitionKey := publisher.getPartitionKey(event)
	assert.Equal(t, itemIDStr, partitionKey)
}

func TestInMemoryEventPublisher_Publish(t *testing.T) {
	publisher := NewEventPublisher()

	event := InventoryItemCreatedEvent{
		ItemID:      uuid.New(),
		SKU:         "SKU-001",
		Name:        "Test Item",
		Description: "Description",
		Quantity:    100,
		OccurredAt:  time.Now(),
	}

	err := publisher.Publish(context.Background(), event)
	assert.NoError(t, err)
}

