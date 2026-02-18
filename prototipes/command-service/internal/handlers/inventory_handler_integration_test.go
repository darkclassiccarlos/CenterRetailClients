package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"command-service/internal/domain"
	"command-service/internal/events"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// IntegrationTestEventPublisher captures published events for verification
type IntegrationTestEventPublisher struct {
	events []interface{}
	logger *zap.Logger
}

func NewIntegrationTestEventPublisher(logger *zap.Logger) *IntegrationTestEventPublisher {
	return &IntegrationTestEventPublisher{
		events: make([]interface{}, 0),
		logger: logger,
	}
}

func (p *IntegrationTestEventPublisher) Publish(ctx context.Context, event interface{}) error {
	p.events = append(p.events, event)
	p.logger.Info("Event captured for integration test", zap.Any("event", event))
	return nil
}

func (p *IntegrationTestEventPublisher) GetEvents() []interface{} {
	return p.events
}

func (p *IntegrationTestEventPublisher) ClearEvents() {
	p.events = make([]interface{}, 0)
}

// TestAdjustStock_Integration_VerifiesEventData tests that AdjustStock publishes correct event data
func TestAdjustStock_Integration_VerifiesEventData(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	eventPublisher := NewIntegrationTestEventPublisher(logger)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   eventPublisher,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": 25, // Aumentar stock en 25
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/adjust", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(125), response["quantity"]) // 100 + 25

	// Assert event was published
	assert.Equal(t, 1, len(eventPublisher.GetEvents()))

	// Verify event data
	event := eventPublisher.GetEvents()[0]
	// Convert to JSON and back to verify structure
	eventJSON, err := json.Marshal(event)
	assert.NoError(t, err)
	
	var stockAdjustedEvent events.StockAdjustedEvent
	err = json.Unmarshal(eventJSON, &stockAdjustedEvent)
	assert.NoError(t, err)

	// Verify event fields
	assert.Equal(t, itemID.String(), fmt.Sprintf("%v", stockAdjustedEvent.ItemID))
	assert.Equal(t, "TEST-001", stockAdjustedEvent.SKU)
	assert.Equal(t, 25, stockAdjustedEvent.Quantity) // Ajuste (diferencia)
	assert.Equal(t, 125, stockAdjustedEvent.NewTotal) // Nueva cantidad total

	mockRepo.AssertExpectations(t)
}

// TestAdjustStock_Negative_VerifiesEventData tests decreasing stock
func TestAdjustStock_Negative_VerifiesEventData(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	eventPublisher := NewIntegrationTestEventPublisher(logger)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   eventPublisher,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	reqBody := map[string]interface{}{
		"quantity": -30, // Disminuir stock en 30
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/adjust", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(70), response["quantity"]) // 100 - 30

	// Assert event was published
	assert.Equal(t, 1, len(eventPublisher.GetEvents()))

	// Verify event data
	event := eventPublisher.GetEvents()[0]
	// Convert to JSON and back to verify structure
	eventJSON, err := json.Marshal(event)
	assert.NoError(t, err)
	
	var stockAdjustedEvent events.StockAdjustedEvent
	err = json.Unmarshal(eventJSON, &stockAdjustedEvent)
	assert.NoError(t, err)

	// Verify event fields
	assert.Equal(t, itemID.String(), fmt.Sprintf("%v", stockAdjustedEvent.ItemID))
	assert.Equal(t, "TEST-001", stockAdjustedEvent.SKU)
	assert.Equal(t, -30, stockAdjustedEvent.Quantity) // Ajuste (diferencia negativa)
	assert.Equal(t, 70, stockAdjustedEvent.NewTotal)  // Nueva cantidad total

	mockRepo.AssertExpectations(t)
}

// TestReserveStock_Integration_VerifiesEventData tests that ReserveStock publishes correct event data
func TestReserveStock_Integration_VerifiesEventData(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	eventPublisher := NewIntegrationTestEventPublisher(logger)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   eventPublisher,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID
	existingItem.Reserved = 10 // Ya tiene 10 reservados

	reqBody := map[string]interface{}{
		"quantity": 20, // Reservar 20 más
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(30), response["reserved"]) // 10 + 20
	assert.Equal(t, float64(70), response["available"]) // 100 - 30

	// Assert event was published
	assert.Equal(t, 1, len(eventPublisher.GetEvents()))

	// Verify event data
	event := eventPublisher.GetEvents()[0]
	// Convert to JSON and back to verify structure
	eventJSON, err := json.Marshal(event)
	assert.NoError(t, err)
	
	var stockReservedEvent events.StockReservedEvent
	err = json.Unmarshal(eventJSON, &stockReservedEvent)
	assert.NoError(t, err)

	// Verify event fields
	assert.Equal(t, itemID.String(), fmt.Sprintf("%v", stockReservedEvent.ItemID))
	assert.Equal(t, "TEST-001", stockReservedEvent.SKU)
	assert.Equal(t, 20, stockReservedEvent.Quantity)   // Cantidad reservada en esta operación
	assert.Equal(t, 30, stockReservedEvent.Reserved)   // Total reservado después
	assert.Equal(t, 70, stockReservedEvent.Available)  // Disponible después

	mockRepo.AssertExpectations(t)
}

// TestReleaseStock_Integration_VerifiesEventData tests that ReleaseStock publishes correct event data
func TestReleaseStock_Integration_VerifiesEventData(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	eventPublisher := NewIntegrationTestEventPublisher(logger)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   eventPublisher,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID
	existingItem.Reserved = 50 // Tiene 50 reservados

	reqBody := map[string]interface{}{
		"quantity": 20, // Liberar 20
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/release", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)

	// Execute
	router.ServeHTTP(w, req)

	// Assert HTTP response
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(30), response["reserved"]) // 50 - 20
	assert.Equal(t, float64(70), response["available"]) // 100 - 30

	// Assert event was published
	assert.Equal(t, 1, len(eventPublisher.GetEvents()))

	// Verify event data
	event := eventPublisher.GetEvents()[0]
	// Convert to JSON and back to verify structure
	eventJSON, err := json.Marshal(event)
	assert.NoError(t, err)
	
	var stockReleasedEvent events.StockReleasedEvent
	err = json.Unmarshal(eventJSON, &stockReleasedEvent)
	assert.NoError(t, err)

	// Verify event fields
	assert.Equal(t, itemID.String(), fmt.Sprintf("%v", stockReleasedEvent.ItemID))
	assert.Equal(t, "TEST-001", stockReleasedEvent.SKU)
	assert.Equal(t, 20, stockReleasedEvent.Quantity)   // Cantidad liberada en esta operación
	assert.Equal(t, 30, stockReleasedEvent.Reserved)   // Total reservado después
	assert.Equal(t, 70, stockReleasedEvent.Available)  // Disponible después

	mockRepo.AssertExpectations(t)
}

// TestStockOperations_Sequence_VerifiesMultipleEvents tests a sequence of stock operations
func TestStockOperations_Sequence_VerifiesMultipleEvents(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	mockRepo := new(MockInventoryRepository)
	eventPublisher := NewIntegrationTestEventPublisher(logger)

	handler := &InventoryHandler{
		logger:     logger,
		repository: mockRepo,
		eventBus:   eventPublisher,
	}

	router := setupTestRouter(handler)

	// Test data
	itemID := uuid.New()
	existingItem := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem.ID = itemID

	// Step 1: Adjust stock (increase by 50)
	existingItem1 := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 100)
	existingItem1.ID = itemID
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem1, nil).Once()
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil).Once()

	reqBody1 := map[string]interface{}{"quantity": 50}
	body1, _ := json.Marshal(reqBody1)
	req1, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/adjust", bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Step 2: Reserve stock (30 units)
	existingItem2 := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 150)
	existingItem2.ID = itemID
	existingItem2.Reserved = 0
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem2, nil).Once()
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil).Once()

	reqBody2 := map[string]interface{}{"quantity": 30}
	body2, _ := json.Marshal(reqBody2)
	req2, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/reserve", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Step 3: Release stock (10 units)
	existingItem3 := domain.NewInventoryItem("TEST-001", "Test Item", "Description", 150)
	existingItem3.ID = itemID
	existingItem3.Reserved = 30
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem3, nil).Once()
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil).Once()

	reqBody3 := map[string]interface{}{"quantity": 10}
	body3, _ := json.Marshal(reqBody3)
	req3, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/release", bytes.NewBuffer(body3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	// Verify all events were published
	events := eventPublisher.GetEvents()
	assert.Equal(t, 3, len(events), "Should have published 3 events")

	// Verify first event (AdjustStock)
	adjustEventJSON, _ := json.Marshal(events[0])
	var adjustEvent map[string]interface{}
	json.Unmarshal(adjustEventJSON, &adjustEvent)
	// Note: JSON serialization uses field names, not JSON tags for structs
	// The event struct fields are: Quantity, NewTotal (PascalCase)
	assert.NotNil(t, adjustEvent["Quantity"], "AdjustStock event should have Quantity field")
	assert.NotNil(t, adjustEvent["NewTotal"], "AdjustStock event should have NewTotal field")
	if quantity, ok := adjustEvent["Quantity"].(float64); ok {
		assert.Equal(t, float64(50), quantity)  // Ajuste
	}
	if newTotal, ok := adjustEvent["NewTotal"].(float64); ok {
		assert.Equal(t, float64(150), newTotal) // Nueva cantidad
	}

	// Verify second event (ReserveStock)
	reserveEventJSON, _ := json.Marshal(events[1])
	var reserveEvent map[string]interface{}
	json.Unmarshal(reserveEventJSON, &reserveEvent)
	assert.NotNil(t, reserveEvent["Quantity"], "ReserveStock event should have Quantity field")
	assert.NotNil(t, reserveEvent["Reserved"], "ReserveStock event should have Reserved field")
	assert.NotNil(t, reserveEvent["Available"], "ReserveStock event should have Available field")

	// Verify third event (ReleaseStock)
	releaseEventJSON, _ := json.Marshal(events[2])
	var releaseEvent map[string]interface{}
	json.Unmarshal(releaseEventJSON, &releaseEvent)
	assert.NotNil(t, releaseEvent["Quantity"], "ReleaseStock event should have Quantity field")
	assert.NotNil(t, releaseEvent["Reserved"], "ReleaseStock event should have Reserved field")
	assert.NotNil(t, releaseEvent["Available"], "ReleaseStock event should have Available field")

	mockRepo.AssertExpectations(t)
}

// TestAdjustStock_EventPublishingFailure_StillSaves tests that item is saved even if event publishing fails
func TestAdjustStock_EventPublishingFailure_StillSaves(t *testing.T) {
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
		"quantity": 25,
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/inventory/items/"+itemID.String()+"/adjust", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Mock expectations - event publishing fails
	mockRepo.On("FindByID", mock.Anything, itemID).Return(existingItem, nil)
	mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InventoryItem")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("events.StockAdjustedEvent")).Return(fmt.Errorf("kafka connection failed"))

	// Execute
	router.ServeHTTP(w, req)

	// Assert HTTP response - should still succeed even if event publishing fails
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(125), response["quantity"]) // Stock should still be updated

	// Verify item was saved
	mockRepo.AssertExpectations(t)
	// Event publishing was attempted
	mockEventBus.AssertExpectations(t)
}

