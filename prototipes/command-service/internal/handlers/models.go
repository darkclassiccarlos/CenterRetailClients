package handlers

// ErrorResponse represents an error response
// @Description Error response with error message
type ErrorResponse struct {
	// Error message describing what went wrong
	// @Example "invalid request: sku is required"
	// @Example "item not found"
	// @Example "insufficient stock available"
	Error string `json:"error" example:"invalid request: sku is required"`
}

// SuccessResponse represents a success response
// @Description Success response with message
type SuccessResponse struct {
	// Success message
	// @Example "item deleted successfully"
	Message string `json:"message" example:"item deleted successfully"`
}

// CreateItemRequest represents the request body for creating an item
// @Description Request to create a new inventory item
type CreateItemRequest struct {
	// SKU (Stock Keeping Unit) - unique identifier for the product
	// @Example "SKU-001"
	// @Example "PROD-2024-ABC"
	SKU string `json:"sku" binding:"required" example:"SKU-001"`
	
	// Product name
	// @Example "Laptop Dell XPS 15"
	// @Example "iPhone 15 Pro Max"
	Name string `json:"name" binding:"required" example:"Laptop Dell XPS 15"`
	
	// Product description (optional)
	// @Example "High-performance laptop with 16GB RAM and 512GB SSD"
	// @Example ""
	Description string `json:"description" example:"High-performance laptop with 16GB RAM and 512GB SSD"`
	
	// Initial stock quantity (must be >= 0)
	// @Example 100
	// @Example 0
	// @Example 500
	Quantity int `json:"quantity" binding:"required,min=0" example:"100"`
}

// CreateItemResponse represents the response after creating an item
// @Description Response after successfully creating an inventory item
type CreateItemResponse struct {
	// Unique item identifier (UUID)
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	
	// SKU (Stock Keeping Unit)
	SKU string `json:"sku" example:"SKU-001"`
	
	// Product name
	Name string `json:"name" example:"Laptop Dell XPS 15"`
	
	// Product description
	Description string `json:"description" example:"High-performance laptop with 16GB RAM and 512GB SSD"`
	
	// Current stock quantity
	Quantity int `json:"quantity" example:"100"`
	
	// Creation timestamp (ISO 8601 format)
	CreatedAt string `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// UpdateItemRequest represents the request body for updating an item
// @Description Request to update an existing inventory item
type UpdateItemRequest struct {
	// Updated product name
	// @Example "Laptop Dell XPS 15 - Updated"
	Name string `json:"name" binding:"required" example:"Laptop Dell XPS 15 - Updated"`
	
	// Updated product description (optional)
	// @Example "High-performance laptop with 32GB RAM and 1TB SSD"
	Description string `json:"description" example:"High-performance laptop with 32GB RAM and 1TB SSD"`
}

// UpdateItemResponse represents the response after updating an item
// @Description Response after successfully updating an inventory item
type UpdateItemResponse struct {
	// Unique item identifier (UUID)
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	
	// SKU (Stock Keeping Unit)
	SKU string `json:"sku" example:"SKU-001"`
	
	// Updated product name
	Name string `json:"name" example:"Laptop Dell XPS 15 - Updated"`
	
	// Updated product description
	Description string `json:"description" example:"High-performance laptop with 32GB RAM and 1TB SSD"`
	
	// Current stock quantity
	Quantity int `json:"quantity" example:"100"`
	
	// Last update timestamp (ISO 8601 format)
	UpdatedAt string `json:"updated_at" example:"2024-01-15T11:45:00Z"`
}

// AdjustStockRequest represents the request body for adjusting stock
// @Description Request to adjust stock quantity (can be positive or negative)
type AdjustStockRequest struct {
	// Quantity adjustment (positive to add, negative to subtract)
	// @Example 10
	// @Example -5
	// @Example 50
	Quantity int `json:"quantity" binding:"required" example:"10"`
}

// StockResponse represents the response for stock operations
// @Description Response for stock-related operations
type StockResponse struct {
	// Unique item identifier (UUID)
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	
	// Total stock quantity
	Quantity int `json:"quantity" example:"100"`
	
	// Available stock (total - reserved)
	Available int `json:"available" example:"80"`
	
	// Reserved stock quantity
	Reserved int `json:"reserved" example:"20"`
	
	// Last update timestamp (ISO 8601 format)
	UpdatedAt string `json:"updated_at" example:"2024-01-15T12:00:00Z"`
}

// ReserveStockRequest represents the request body for reserving stock
// @Description Request to reserve stock for an order
type ReserveStockRequest struct {
	// Quantity to reserve (must be >= 1)
	// @Example 5
	// @Example 10
	// @Example 1
	Quantity int `json:"quantity" binding:"required,min=1" example:"5"`
}

// ReleaseStockRequest represents the request body for releasing stock
// @Description Request to release previously reserved stock
type ReleaseStockRequest struct {
	// Quantity to release (must be >= 1)
	// @Example 5
	// @Example 10
	// @Example 1
	Quantity int `json:"quantity" binding:"required,min=1" example:"5"`
}

