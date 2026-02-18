package handlers

// ErrorResponse represents an error response
// @Description Error response with error message
type ErrorResponse struct {
	// Error message describing what went wrong
	// @Example "item not found"
	// @Example "invalid page number"
	Error string `json:"error" example:"item not found"`
}

// InventoryItemResponse represents an inventory item response
// @Description Response with inventory item details
type InventoryItemResponse struct {
	// Unique item identifier (UUID)
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	
	// SKU (Stock Keeping Unit)
	SKU string `json:"sku" example:"SKU-001"`
	
	// Product name
	Name string `json:"name" example:"Laptop Dell XPS 15"`
	
	// Product description
	Description string `json:"description" example:"High-performance laptop with 16GB RAM and 512GB SSD"`
	
	// Total stock quantity
	Quantity int `json:"quantity" example:"100"`
	
	// Reserved stock quantity
	Reserved int `json:"reserved" example:"20"`
	
	// Available stock (total - reserved)
	Available int `json:"available" example:"80"`
	
	// Creation timestamp (ISO 8601 format)
	CreatedAt string `json:"created_at" example:"2024-01-15T10:30:00Z"`
	
	// Last update timestamp (ISO 8601 format)
	UpdatedAt string `json:"updated_at" example:"2024-01-15T11:45:00Z"`
}

// StockStatusResponse represents stock status response
// @Description Response with stock status information
type StockStatusResponse struct {
	// Unique item identifier (UUID)
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	
	// SKU (Stock Keeping Unit)
	SKU string `json:"sku" example:"SKU-001"`
	
	// Total stock quantity
	Quantity int `json:"quantity" example:"100"`
	
	// Reserved stock quantity
	Reserved int `json:"reserved" example:"20"`
	
	// Available stock (total - reserved)
	Available int `json:"available" example:"80"`
	
	// Last update timestamp (ISO 8601 format)
	UpdatedAt string `json:"updated_at" example:"2024-01-15T12:00:00Z"`
}

// ListItemsResponse represents the response for listing items
// @Description Response with paginated list of inventory items
type ListItemsResponse struct {
	// List of inventory items
	Items []InventoryItemResponse `json:"items"`
	
	// Total number of items
	Total int `json:"total" example:"100"`
	
	// Current page number
	Page int `json:"page" example:"1"`
	
	// Number of items per page
	PageSize int `json:"page_size" example:"10"`
	
	// Total number of pages
	TotalPages int `json:"total_pages" example:"10"`
}

