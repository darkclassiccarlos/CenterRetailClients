package domain

import (
	"time"

	"github.com/google/uuid"
)

// InventoryItem represents the aggregate root for inventory
type InventoryItem struct {
	ID          uuid.UUID
	SKU         string
	Name        string
	Description string
	Quantity    int
	Reserved    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int // For optimistic locking
}

// NewInventoryItem creates a new inventory item
func NewInventoryItem(sku, name, description string, initialQuantity int) *InventoryItem {
	return &InventoryItem{
		ID:          uuid.New(),
		SKU:         sku,
		Name:        name,
		Description: description,
		Quantity:    initialQuantity,
		Reserved:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
}

// AvailableQuantity returns the available quantity (total - reserved)
func (i *InventoryItem) AvailableQuantity() int {
	return i.Quantity - i.Reserved
}

// AdjustStock adjusts the stock quantity
func (i *InventoryItem) AdjustStock(quantity int) error {
	newQuantity := i.Quantity + quantity
	if newQuantity < 0 {
		return ErrInsufficientStock
	}
	i.Quantity = newQuantity
	i.UpdatedAt = time.Now()
	i.Version++
	return nil
}

// ReserveStock reserves stock
func (i *InventoryItem) ReserveStock(quantity int) error {
	if i.AvailableQuantity() < quantity {
		return ErrInsufficientStock
	}
	i.Reserved += quantity
	i.UpdatedAt = time.Now()
	i.Version++
	return nil
}

// ReleaseStock releases reserved stock
func (i *InventoryItem) ReleaseStock(quantity int) error {
	if i.Reserved < quantity {
		return ErrInvalidReleaseQuantity
	}
	i.Reserved -= quantity
	i.UpdatedAt = time.Now()
	i.Version++
	return nil
}

// FulfillReservation fulfills a reservation and reduces stock
func (i *InventoryItem) FulfillReservation(quantity int) error {
	if i.Reserved < quantity {
		return ErrInvalidReleaseQuantity
	}
	i.Reserved -= quantity
	i.Quantity -= quantity
	i.UpdatedAt = time.Now()
	i.Version++
	return nil
}

// Domain errors
var (
	ErrInsufficientStock      = &DomainError{Message: "insufficient stock available"}
	ErrInvalidReleaseQuantity = &DomainError{Message: "invalid release quantity"}
	ErrItemNotFound           = &DomainError{Message: "item not found"}
)

// DomainError represents a domain-level error
type DomainError struct {
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

