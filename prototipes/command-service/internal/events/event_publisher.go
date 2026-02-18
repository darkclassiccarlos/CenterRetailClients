package events

import (
	"context"

	"go.uber.org/zap"
)

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	Publish(ctx context.Context, event interface{}) error
}

// Event represents a base event structure
type Event struct {
	EventType  string
	OccurredAt interface{}
}

// Inventory domain events
type InventoryItemCreatedEvent struct {
	ItemID      interface{}
	SKU         string
	Name        string
	Description string
	Quantity    int
	OccurredAt  interface{}
}

type InventoryItemUpdatedEvent struct {
	ItemID      interface{}
	Name        string
	Description string
	OccurredAt  interface{}
}

type InventoryItemDeletedEvent struct {
	ItemID     interface{}
	SKU        string
	OccurredAt interface{}
}

type StockAdjustedEvent struct {
	ItemID     interface{}
	SKU        string
	Quantity   int
	NewTotal   int
	OccurredAt interface{}
}

type StockReservedEvent struct {
	ItemID     interface{}
	SKU        string
	Quantity   int
	Reserved   int
	Available  int
	OccurredAt interface{}
}

type StockReleasedEvent struct {
	ItemID     interface{}
	SKU        string
	Quantity   int
	Reserved   int
	Available  int
	OccurredAt interface{}
}

// InMemoryEventPublisher is a placeholder implementation
// TODO: Replace with actual event broker implementation (Kafka, RabbitMQ, etc.)
type InMemoryEventPublisher struct {
	logger *zap.Logger
	events []interface{}
}

func NewEventPublisher() EventPublisher {
	return &InMemoryEventPublisher{
		logger: zap.NewNop(),
		events: make([]interface{}, 0),
	}
}

func (p *InMemoryEventPublisher) Publish(ctx context.Context, event interface{}) error {
	// TODO: Implement actual event publishing to Kafka/RabbitMQ/etc.
	p.events = append(p.events, event)
	p.logger.Info("Event published (in-memory)", zap.Any("event", event))
	return nil
}

