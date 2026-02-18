package models

import "time"

// InventoryItem represents a read model for inventory items
type InventoryItem struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	Reserved    int       `json:"reserved"`
	Available   int       `json:"available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StockStatus represents the stock status of an item
type StockStatus struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Quantity    int       `json:"quantity"`
	Reserved    int       `json:"reserved"`
	Available   int       `json:"available"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListItemsResponse represents the response for listing items
type ListItemsResponse struct {
	Items      []InventoryItem `json:"items"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

