package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"listener-service/internal/database"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EventPublisher defines the interface for publishing confirmation events
type EventPublisher interface {
	PublishConfirmationEvent(ctx context.Context, eventType string, itemID, sku string, data interface{}) error
}

// EventProcessor processes domain events and updates the database
type EventProcessor struct {
	db       *database.SingleWriterDB
	producer EventPublisher
	logger   *zap.Logger
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(db *database.SingleWriterDB, producer EventPublisher, logger *zap.Logger) *EventProcessor {
	return &EventProcessor{
		db:       db,
		producer: producer,
		logger:   logger,
	}
}

// ProcessEvent processes a single event
func (p *EventProcessor) ProcessEvent(ctx context.Context, eventType string, eventData []byte) error {
	switch eventType {
	case "InventoryItemCreated":
		return p.processItemCreated(ctx, eventData)
	case "InventoryItemUpdated":
		return p.processItemUpdated(ctx, eventData)
	case "InventoryItemDeleted":
		return p.processItemDeleted(ctx, eventData)
	case "StockAdjusted":
		return p.processStockAdjusted(ctx, eventData)
	case "StockReserved":
		return p.processStockReserved(ctx, eventData)
	case "StockReleased":
		return p.processStockReleased(ctx, eventData)
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}
}

// processItemCreated processes InventoryItemCreated event
func (p *EventProcessor) processItemCreated(ctx context.Context, eventData []byte) error {
	var event struct {
		ItemID      string `json:"itemId"`
		SKU         string `json:"sku"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Quantity    int    `json:"quantity"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	itemID, err := uuid.Parse(event.ItemID)
	if err != nil {
		return fmt.Errorf("invalid item ID: %w", err)
	}

	dbItem := &database.InventoryItem{
		ID:          itemID.String(),
		SKU:         event.SKU,
		Name:        event.Name,
		Description: event.Description,
		Quantity:    event.Quantity,
		Reserved:    0,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := p.db.CreateItem(ctx, dbItem); err != nil {
		return fmt.Errorf("failed to create item: %w", err)
	}

	p.logger.Info("Item created", zap.String("item_id", itemID.String()), zap.String("sku", event.SKU))

	// Publish confirmation event to update Redis in query-service
	if p.producer != nil {
		confirmationData := map[string]interface{}{
			"itemId":      itemID.String(),
			"sku":         event.SKU,
			"name":        event.Name,
			"description": event.Description,
			"quantity":    event.Quantity,
			"reserved":    0,
			"available":   event.Quantity,
		}
		if err := p.producer.PublishConfirmationEvent(ctx, "InventoryItemCreated", itemID.String(), event.SKU, confirmationData); err != nil {
			p.logger.Warn("Failed to publish confirmation event", zap.Error(err))
			// Don't fail the operation if confirmation event fails
		}
	}

	return nil
}

// processItemUpdated processes InventoryItemUpdated event
func (p *EventProcessor) processItemUpdated(ctx context.Context, eventData []byte) error {
	var event struct {
		ItemID      string `json:"itemId"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	itemID, err := uuid.Parse(event.ItemID)
	if err != nil {
		return fmt.Errorf("invalid item ID: %w", err)
	}

	// Get current item to get version
	currentItem, err := p.db.GetItem(ctx, itemID.String())
	if err != nil {
		return fmt.Errorf("failed to get item for update: %w", err)
	}

	dbItem := &database.InventoryItem{
		ID:          itemID.String(),
		Name:        event.Name,
		Description: event.Description,
		Version:     currentItem.Version,
	}

	if err := p.db.UpdateItem(ctx, dbItem); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	p.logger.Info("Item updated", zap.String("item_id", itemID.String()))

	// Get updated item to publish confirmation event
	updatedItem, err := p.db.GetItem(ctx, itemID.String())
	if err == nil && p.producer != nil {
		confirmationData := map[string]interface{}{
			"itemId":      itemID.String(),
			"sku":         updatedItem.SKU,
			"name":        updatedItem.Name,
			"description": updatedItem.Description,
			"quantity":    updatedItem.Quantity,
			"reserved":    updatedItem.Reserved,
			"available":   updatedItem.Available,
		}
		if err := p.producer.PublishConfirmationEvent(ctx, "InventoryItemUpdated", itemID.String(), updatedItem.SKU, confirmationData); err != nil {
			p.logger.Warn("Failed to publish confirmation event", zap.Error(err))
		}
	}

	return nil
}

// processItemDeleted processes InventoryItemDeleted event
func (p *EventProcessor) processItemDeleted(ctx context.Context, eventData []byte) error {
	var event struct {
		ItemID string `json:"itemId"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	itemID, err := uuid.Parse(event.ItemID)
	if err != nil {
		return fmt.Errorf("invalid item ID: %w", err)
	}

	// Get item before deletion to publish confirmation event
	var sku string
	itemToDelete, err := p.db.GetItem(ctx, itemID.String())
	if err == nil {
		sku = itemToDelete.SKU
	}

	if err := p.db.DeleteItem(ctx, itemID.String()); err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	p.logger.Info("Item deleted", zap.String("item_id", itemID.String()))

	// Publish confirmation event
	if p.producer != nil {
		confirmationData := map[string]interface{}{
			"itemId": itemID.String(),
			"sku":    sku,
		}
		if err := p.producer.PublishConfirmationEvent(ctx, "InventoryItemDeleted", itemID.String(), sku, confirmationData); err != nil {
			p.logger.Warn("Failed to publish confirmation event", zap.Error(err))
		}
	}

	return nil
}

// processStockAdjusted processes StockAdjusted event
func (p *EventProcessor) processStockAdjusted(ctx context.Context, eventData []byte) error {
	var event struct {
		ItemID   string `json:"itemId"`
		Quantity int    `json:"quantity"` // This is the adjustment (difference), not the new total
		NewTotal int    `json:"newTotal"`  // This is the new total quantity after adjustment
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	itemID, err := uuid.Parse(event.ItemID)
	if err != nil {
		return fmt.Errorf("invalid item ID: %w", err)
	}

	// Get current item to get version
	currentItem, err := p.db.GetItem(ctx, itemID.String())
	if err != nil {
		return fmt.Errorf("failed to get item for stock adjustment: %w", err)
	}

	// The event.Quantity field contains the adjustment (difference), not the new total
	// For example: if stock was 100 and we adjust by +25, event.Quantity = 25
	adjustment := event.Quantity

	if err := p.db.AdjustStock(ctx, itemID.String(), adjustment, currentItem.Version); err != nil {
		return fmt.Errorf("failed to adjust stock: %w", err)
	}

	p.logger.Info("Stock adjusted", zap.String("item_id", itemID.String()), zap.Int("adjustment", adjustment))

	// Get updated item to publish confirmation event
	updatedItem, err := p.db.GetItem(ctx, itemID.String())
	if err == nil && p.producer != nil {
		confirmationData := map[string]interface{}{
			"itemId":   itemID.String(),
			"sku":      updatedItem.SKU,
			"quantity": updatedItem.Quantity,
			"reserved": updatedItem.Reserved,
			"available": updatedItem.Available,
		}
		if err := p.producer.PublishConfirmationEvent(ctx, "StockAdjusted", itemID.String(), updatedItem.SKU, confirmationData); err != nil {
			p.logger.Warn("Failed to publish confirmation event", zap.Error(err))
		}
	}

	return nil
}

// processStockReserved processes StockReserved event
func (p *EventProcessor) processStockReserved(ctx context.Context, eventData []byte) error {
	var event struct {
		ItemID   string `json:"itemId"`
		Quantity int    `json:"quantity"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	itemID, err := uuid.Parse(event.ItemID)
	if err != nil {
		return fmt.Errorf("invalid item ID: %w", err)
	}

	// Get current item to get version
	currentItem, err := p.db.GetItem(ctx, itemID.String())
	if err != nil {
		return fmt.Errorf("failed to get item for stock reservation: %w", err)
	}

	if err := p.db.ReserveStock(ctx, itemID.String(), event.Quantity, currentItem.Version); err != nil {
		return fmt.Errorf("failed to reserve stock: %w", err)
	}

	p.logger.Info("Stock reserved", zap.String("item_id", itemID.String()), zap.Int("quantity", event.Quantity))

	// Get updated item to publish confirmation event
	updatedItem, err := p.db.GetItem(ctx, itemID.String())
	if err == nil && p.producer != nil {
		confirmationData := map[string]interface{}{
			"itemId":   itemID.String(),
			"sku":      updatedItem.SKU,
			"quantity": updatedItem.Quantity,
			"reserved": updatedItem.Reserved,
			"available": updatedItem.Available,
		}
		if err := p.producer.PublishConfirmationEvent(ctx, "StockReserved", itemID.String(), updatedItem.SKU, confirmationData); err != nil {
			p.logger.Warn("Failed to publish confirmation event", zap.Error(err))
		}
	}

	return nil
}

// processStockReleased processes StockReleased event
func (p *EventProcessor) processStockReleased(ctx context.Context, eventData []byte) error {
	var event struct {
		ItemID   string `json:"itemId"`
		Quantity int    `json:"quantity"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	itemID, err := uuid.Parse(event.ItemID)
	if err != nil {
		return fmt.Errorf("invalid item ID: %w", err)
	}

	// Get current item to get version
	currentItem, err := p.db.GetItem(ctx, itemID.String())
	if err != nil {
		return fmt.Errorf("failed to get item for stock release: %w", err)
	}

	if err := p.db.ReleaseStock(ctx, itemID.String(), event.Quantity, currentItem.Version); err != nil {
		return fmt.Errorf("failed to release stock: %w", err)
	}

	p.logger.Info("Stock released", zap.String("item_id", itemID.String()), zap.Int("quantity", event.Quantity))

	// Get updated item to publish confirmation event
	updatedItem, err := p.db.GetItem(ctx, itemID.String())
	if err == nil && p.producer != nil {
		confirmationData := map[string]interface{}{
			"itemId":   itemID.String(),
			"sku":      updatedItem.SKU,
			"quantity": updatedItem.Quantity,
			"reserved": updatedItem.Reserved,
			"available": updatedItem.Available,
		}
		if err := p.producer.PublishConfirmationEvent(ctx, "StockReleased", itemID.String(), updatedItem.SKU, confirmationData); err != nil {
			p.logger.Warn("Failed to publish confirmation event", zap.Error(err))
		}
	}

	return nil
}

