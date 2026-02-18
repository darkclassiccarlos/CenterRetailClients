package repository

import (
	"context"

	"query-service/internal/models"

	"github.com/google/uuid"
)

// ReadRepository defines the interface for read operations
type ReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.InventoryItem, error)
	FindBySKU(ctx context.Context, sku string) (*models.InventoryItem, error)
	ListItems(ctx context.Context, page, pageSize int) ([]models.InventoryItem, int, error)
	GetStockStatus(ctx context.Context, id uuid.UUID) (*models.StockStatus, error)
}

// InMemoryReadRepository is a placeholder implementation
// TODO: Replace with actual database implementation that reads from Read Model
type InMemoryReadRepository struct {
	items map[uuid.UUID]*models.InventoryItem
}

func NewReadRepository() ReadRepository {
	return &InMemoryReadRepository{
		items: make(map[uuid.UUID]*models.InventoryItem),
	}
}

func (r *InMemoryReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.InventoryItem, error) {
	item, exists := r.items[id]
	if !exists {
		return nil, ErrItemNotFound
	}
	return item, nil
}

func (r *InMemoryReadRepository) FindBySKU(ctx context.Context, sku string) (*models.InventoryItem, error) {
	for _, item := range r.items {
		if item.SKU == sku {
			return item, nil
		}
	}
	return nil, ErrItemNotFound
}

func (r *InMemoryReadRepository) ListItems(ctx context.Context, page, pageSize int) ([]models.InventoryItem, int, error) {
	items := make([]models.InventoryItem, 0)
	for _, item := range r.items {
		items = append(items, *item)
	}

	total := len(items)

	// Simple pagination
	start := (page - 1) * pageSize
	if start > total {
		return []models.InventoryItem{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	if start >= total {
		return []models.InventoryItem{}, total, nil
	}

	return items[start:end], total, nil
}

func (r *InMemoryReadRepository) GetStockStatus(ctx context.Context, id uuid.UUID) (*models.StockStatus, error) {
	item, exists := r.items[id]
	if !exists {
		return nil, ErrItemNotFound
	}

	return &models.StockStatus{
		ID:        item.ID,
		SKU:       item.SKU,
		Quantity:  item.Quantity,
		Reserved:  item.Reserved,
		Available: item.Available,
		UpdatedAt: item.UpdatedAt,
	}, nil
}

var (
	ErrItemNotFound = &RepositoryError{Message: "item not found"}
)

type RepositoryError struct {
	Message string
}

func (e *RepositoryError) Error() string {
	return e.Message
}

