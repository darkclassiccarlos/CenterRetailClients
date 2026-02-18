package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewInventoryItem(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)

	assert.NotEqual(t, uuid.Nil, item.ID)
	assert.Equal(t, "SKU-001", item.SKU)
	assert.Equal(t, "Test Item", item.Name)
	assert.Equal(t, "Description", item.Description)
	assert.Equal(t, 100, item.Quantity)
	assert.Equal(t, 0, item.Reserved)
	assert.Equal(t, 1, item.Version)
	assert.False(t, item.CreatedAt.IsZero())
	assert.False(t, item.UpdatedAt.IsZero())
}

func TestAvailableQuantity(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	item.Reserved = 30

	assert.Equal(t, 70, item.AvailableQuantity())
}

func TestAdjustStock_Success_Increase(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	originalVersion := item.Version

	err := item.AdjustStock(50)

	assert.NoError(t, err)
	assert.Equal(t, 150, item.Quantity)
	assert.Equal(t, originalVersion+1, item.Version)
}

func TestAdjustStock_Success_Decrease(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	originalVersion := item.Version

	err := item.AdjustStock(-30)

	assert.NoError(t, err)
	assert.Equal(t, 70, item.Quantity)
	assert.Equal(t, originalVersion+1, item.Version)
}

func TestAdjustStock_Error_NegativeResult(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 10)
	originalQuantity := item.Quantity
	originalVersion := item.Version

	err := item.AdjustStock(-20)

	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientStock, err)
	assert.Equal(t, originalQuantity, item.Quantity)
	assert.Equal(t, originalVersion, item.Version)
}

func TestReserveStock_Success(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	item.Reserved = 20
	originalVersion := item.Version

	err := item.ReserveStock(30)

	assert.NoError(t, err)
	assert.Equal(t, 50, item.Reserved)
	assert.Equal(t, 50, item.AvailableQuantity())
	assert.Equal(t, originalVersion+1, item.Version)
}

func TestReserveStock_Error_InsufficientStock(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	item.Reserved = 80
	originalReserved := item.Reserved
	originalVersion := item.Version

	err := item.ReserveStock(30) // Only 20 available, trying to reserve 30

	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientStock, err)
	assert.Equal(t, originalReserved, item.Reserved)
	assert.Equal(t, originalVersion, item.Version)
}

func TestReleaseStock_Success(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	item.Reserved = 50
	originalVersion := item.Version

	err := item.ReleaseStock(20)

	assert.NoError(t, err)
	assert.Equal(t, 30, item.Reserved)
	assert.Equal(t, 70, item.AvailableQuantity())
	assert.Equal(t, originalVersion+1, item.Version)
}

func TestReleaseStock_Error_InvalidQuantity(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	item.Reserved = 30
	originalReserved := item.Reserved
	originalVersion := item.Version

	err := item.ReleaseStock(50) // Trying to release 50 when only 30 reserved

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidReleaseQuantity, err)
	assert.Equal(t, originalReserved, item.Reserved)
	assert.Equal(t, originalVersion, item.Version)
}

func TestFulfillReservation_Success(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	item.Reserved = 30
	originalVersion := item.Version

	err := item.FulfillReservation(20)

	assert.NoError(t, err)
	assert.Equal(t, 10, item.Reserved)
	assert.Equal(t, 80, item.Quantity)
	assert.Equal(t, originalVersion+1, item.Version)
}

func TestFulfillReservation_Error_InvalidQuantity(t *testing.T) {
	item := NewInventoryItem("SKU-001", "Test Item", "Description", 100)
	item.Reserved = 30
	originalReserved := item.Reserved
	originalQuantity := item.Quantity
	originalVersion := item.Version

	err := item.FulfillReservation(50) // Trying to fulfill 50 when only 30 reserved

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidReleaseQuantity, err)
	assert.Equal(t, originalReserved, item.Reserved)
	assert.Equal(t, originalQuantity, item.Quantity)
	assert.Equal(t, originalVersion, item.Version)
}

