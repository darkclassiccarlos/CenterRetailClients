package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"command-service/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockInventoryRepository is a mock implementation of InventoryRepository
type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) Save(ctx context.Context, item *domain.InventoryItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockInventoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.InventoryItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.InventoryItem), args.Error(1)
}

func (m *MockInventoryRepository) FindBySKU(ctx context.Context, sku string) (*domain.InventoryItem, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.InventoryItem), args.Error(1)
}

func (m *MockInventoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockEventPublisher is a mock implementation of EventPublisher
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func setupTestRouter(handler *InventoryHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	{
		inventory := v1.Group("/inventory")
		{
			inventory.POST("/items", handler.CreateItem)
			inventory.PUT("/items/:id", handler.UpdateItem)
			inventory.DELETE("/items/:id", handler.DeleteItem)
			inventory.POST("/items/:id/adjust", handler.AdjustStock)
			inventory.POST("/items/:id/reserve", handler.ReserveStock)
			inventory.POST("/items/:id/release", handler.ReleaseStock)
		}
	}

	return router
}

func TestCreateItem_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	reqBody := map[string]interface{}{
		"sku":         "TEST-001",
		"name":        "Test Item",
		"description": "Test Description",
		"quantity":    100,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.InventoryItemCreatedEvent")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "TEST-001", response["sku"])
	assert.Equal(t, "Test Item", response["name"])
	assert.Equal(t, float64(100), response["quantity"])

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestCreateItem_InvalidRequest_MissingFields(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data - missing required fields
	reqBody := map[string]interface{}{
		"name": "Test Item",
		// Missing sku and quantity
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "required")

	mockRepo.AssertNotCalled(t, "Save")
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestCreateItem_InvalidRequest_NegativeQuantity(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data - negative quantity
	reqBody := map[string]interface{}{
		"sku":      "TEST-001",
		"name":     "Test Item",
		"quantity": -10,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "min")

	mockRepo.AssertNotCalled(t, "Save")
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestCreateItem_RepositoryError(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	reqBody := map[string]interface{}{
		"sku":      "TEST-001",
		"name":     "Test Item",
		"quantity": 100,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations - repository error
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(assert.AnError)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "failed to create item")

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestUpdateItem_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Old Name", "Old Description", 100)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"name":        "New Name",
		"description": "New Description",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/items/"+itemID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.InventoryItemUpdatedEvent")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "New Name", response["name"])
	assert.Equal(t, "New Description", response["description"])

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestUpdateItem_NotFound(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()

	reqBody := map[string]interface{}{
		"name": "New Name",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/api/v1/inventory/items/"+itemID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(nil, domain.ErrItemNotFound)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "not found")

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestAdjustStock_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": 10,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/adjust", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.StockAdjustedEvent")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(110), response["quantity"]) // 100 + 10

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestAdjustStock_InsufficientStock(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 10)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": -20, // Trying to reduce by 20 when only 10 available
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/adjust", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "insufficient")

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestReserveStock_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": 20,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.StockReservedEvent")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(20), response["reserved"]) // Reserved quantity
	assert.Equal(t, float64(80), response["available"]) // 100 - 20

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestReserveStock_InsufficientStock(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 10)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": 20, // Trying to reserve 20 when only 10 available
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "insufficient")

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestReserveStock_ItemNotFound(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()

	reqBody := map[string]interface{}{
		"quantity": 20,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(nil, domain.ErrItemNotFound)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "item not found", response["error"])

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestReserveStock_InvalidID(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	reqBody := map[string]interface{}{
		"quantity": 20,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/invalid-id/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "invalid item id", response["error"])

	mockRepo.AssertNotCalled(t, "FindByID")
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestReserveStock_InvalidQuantity(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()

	// Test with quantity = 0
	reqBody := map[string]interface{}{
		"quantity": 0,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockRepo.AssertNotCalled(t, "FindByID")
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestReserveStock_NegativeQuantity(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()

	// Test with negative quantity
	reqBody := map[string]interface{}{
		"quantity": -10,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockRepo.AssertNotCalled(t, "FindByID")
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestReserveStock_RepositoryError(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": 20,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(assert.AnError)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "failed to reserve stock", response["error"])

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestReserveStock_EventPublishError(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": 20,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.StockReservedEvent")).Return(assert.AnError)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	// Event publish error should not fail the request (item is already saved)
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(20), response["reserved"])
	assert.Equal(t, float64(80), response["available"])

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestReserveStock_MultipleReservations(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID
	existingItem.Reserved = 30 // Already has some reserved stock

	reqBody := map[string]interface{}{
		"quantity": 20, // Reserve additional 20
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.StockReservedEvent")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(50), response["reserved"]) // 30 + 20
	assert.Equal(t, float64(50), response["available"]) // 100 - 50

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestReleaseStock_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID
	existingItem.Reserved = 30 // Pre-reserved stock

	reqBody := map[string]interface{}{
		"quantity": 10,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/release", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.StockReleasedEvent")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(20), response["reserved"]) // 30 - 10
	assert.Equal(t, float64(80), response["available"]) // 100 - 20

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestReleaseStock_InvalidQuantity(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID
	existingItem.Reserved = 10 // Pre-reserved stock

	reqBody := map[string]interface{}{
		"quantity": 20, // Trying to release 20 when only 10 reserved
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/release", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "invalid")

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestDeleteItem_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	req, _ := http.NewRequest("DELETE", "/api/v1/inventory/items/"+itemID.String(), nil)
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Delete", mock.Anything, itemID).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.InventoryItemDeletedEvent")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["message"].(string), "deleted")

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestDeleteItem_NotFound(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	mockEventBus := new(MockEventPublisher)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   mockEventBus,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()

	req, _ := http.NewRequest("DELETE", "/api/v1/inventory/items/"+itemID.String(), nil)
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(nil, domain.ErrItemNotFound)

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"].(string), "not found")

	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

