package commands

import (
	"github.com/google/uuid"
)

// CreateItemCommand represents a command to create a new inventory item
type CreateItemCommand struct {
	SKU         string
	Name        string
	Description string
	Quantity    int
}

// UpdateItemCommand represents a command to update an inventory item
type UpdateItemCommand struct {
	ID          uuid.UUID
	Name        string
	Description string
}

// AdjustStockCommand represents a command to adjust stock
type AdjustStockCommand struct {
	ID       uuid.UUID
	Quantity int
}

// ReserveStockCommand represents a command to reserve stock
type ReserveStockCommand struct {
	ID       uuid.UUID
	Quantity int
}

// ReleaseStockCommand represents a command to release reserved stock
type ReleaseStockCommand struct {
	ID       uuid.UUID
	Quantity int
}

// DeleteItemCommand represents a command to delete an inventory item
type DeleteItemCommand struct {
	ID uuid.UUID
}

