package repository

import (
	"context"

	"command-service/internal/domain"

	"github.com/google/uuid"
)

// InventoryRepository defines the interface for inventory persistence
type InventoryRepository interface {
	Save(ctx context.Context, item *domain.InventoryItem) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.InventoryItem, error)
	FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// InMemoryInventoryRepository is a placeholder implementation
// TODO: Replace with actual database implementation (PostgreSQL, etc.)
type InMemoryInventoryRepository struct {
	items map[uuid.UUID]*domain.InventoryItem
}

func NewInventoryRepository() InventoryRepository {
	return &InMemoryInventoryRepository{
		items: make(map[uuid.UUID]*domain.InventoryItem),
	}
}

func (r *InMemoryInventoryRepository) Save(ctx context.Context, item *domain.InventoryItem) error {
	r.items[item.ID] = item
	return nil
}

func (r *InMemoryInventoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.InventoryItem, error) {
	item, exists := r.items[id]
	if !exists {
		return nil, domain.ErrItemNotFound
	}
	return item, nil
}

func (r *InMemoryInventoryRepository) FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error) {
	for _, item := range r.items {
		if item.SKU == sku {
			return item, nil
		}
	}
	return nil, domain.ErrItemNotFound
}

func (r *InMemoryInventoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, exists := r.items[id]; !exists {
		return domain.ErrItemNotFound
	}
	delete(r.items, id)
	return nil
}

