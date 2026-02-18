package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"query-service/internal/cache"
	"query-service/internal/models"
	"query-service/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockCache is a mock implementation of cache.Cache
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCache) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

// MockRepository is a mock implementation of repository.ReadRepository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.InventoryItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InventoryItem), args.Error(1)
}

func (m *MockRepository) FindBySKU(ctx context.Context, sku string) (*models.InventoryItem, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InventoryItem), args.Error(1)
}

func (m *MockRepository) ListItems(ctx context.Context, page, pageSize int) ([]models.InventoryItem, int, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]models.InventoryItem), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetStockStatus(ctx context.Context, id uuid.UUID) (*models.StockStatus, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StockStatus), args.Error(1)
}

// Helper function to create a test handler
func createTestHandler(cacheClient cache.Cache, repo repository.ReadRepository) *InventoryHandler {
	logger := zap.NewNop()
	return &InventoryHandler{
		logger:     logger,
		repository: repo,
		cache:      cacheClient,
		cacheTTL:   300,
	}
}

// Helper function to create a test item
func createTestItem(id uuid.UUID, sku string) *models.InventoryItem {
	now := time.Now()
	return &models.InventoryItem{
		ID:          id.String(),
		SKU:         sku,
		Name:        "Test Item",
		Description: "Test Description",
		Quantity:    100,
		Reserved:    20,
		Available:   80,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Helper function to setup Gin router for testing
func setupTestRouter(handler *InventoryHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		inventory := v1.Group("/inventory")
		{
			inventory.GET("/items", handler.ListItems)
			inventory.GET("/items/:id", handler.GetItemByID)
			inventory.GET("/items/sku/:sku", handler.GetItemBySKU)
			inventory.GET("/items/:id/stock", handler.GetStockStatus)
		}
	}
	return router
}

func TestListItems_CacheHit(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	// Expected cached response
	cachedResponse := ListItemsResponse{
		Items: []InventoryItemResponse{
			{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				SKU:         "SKU-001",
				Name:        "Test Item",
				Description: "Test Description",
				Quantity:    100,
				Reserved:    20,
				Available:   80,
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   10,
		TotalPages: 1,
	}

	cachedData, _ := json.Marshal(cachedResponse)
	mockCache.On("Get", mock.Anything, "items:list:1:10").Return(cachedData, nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "ListItems") // Repository should not be called on cache hit

	var response ListItemsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.Items, 1)
}

func TestListItems_CacheMiss(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "items:list:1:10").Return(nil, cache.ErrCacheMiss)

	// Mock repository response
	testItem := createTestItem(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), "SKU-001")
	mockRepo.On("ListItems", mock.Anything, 1, 10).Return([]models.InventoryItem{*testItem}, 1, nil)

	// Mock cache set
	mockCache.On("Set", mock.Anything, "items:list:1:10", mock.Anything, mock.Anything).Return(nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)

	var response ListItemsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
	assert.Len(t, response.Items, 1)
}

func TestListItems_NoCache(t *testing.T) {
	// Setup - handler without cache
	mockRepo := new(MockRepository)
	handler := createTestHandler(nil, mockRepo)
	router := setupTestRouter(handler)

	// Mock repository response
	testItem := createTestItem(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), "SKU-001")
	mockRepo.On("ListItems", mock.Anything, 1, 10).Return([]models.InventoryItem{*testItem}, 1, nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)

	var response ListItemsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 1, response.Total)
}

func TestGetItemByID_CacheHit(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	testItem := createTestItem(itemID, "SKU-001")

	cachedData, _ := json.Marshal(testItem)
	mockCache.On("Get", mock.Anything, "item:id:550e8400-e29b-41d4-a716-446655440000").Return(cachedData, nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "FindByID")

	var response InventoryItemResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, testItem.SKU, response.SKU)
	assert.Equal(t, testItem.Quantity, response.Quantity)
}

func TestGetItemByID_CacheMiss(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	testItem := createTestItem(itemID, "SKU-001")

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "item:id:550e8400-e29b-41d4-a716-446655440000").Return(nil, cache.ErrCacheMiss)
	mockRepo.On("FindByID", mock.Anything, itemID).Return(testItem, nil)
	mockCache.On("Set", mock.Anything, "item:id:550e8400-e29b-41d4-a716-446655440000", mock.Anything, mock.Anything).Return(nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)

	var response InventoryItemResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, testItem.SKU, response.SKU)
}

func TestGetItemByID_NotFound(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "item:id:550e8400-e29b-41d4-a716-446655440000").Return(nil, cache.ErrCacheMiss)
	mockRepo.On("FindByID", mock.Anything, itemID).Return(nil, repository.ErrItemNotFound)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetItemByID_InvalidUUID(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockCache.AssertNotCalled(t, "Get")
	mockRepo.AssertNotCalled(t, "FindByID")
}

func TestGetItemBySKU_CacheHit(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	testItem := createTestItem(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), "SKU-001")

	cachedData, _ := json.Marshal(testItem)
	mockCache.On("Get", mock.Anything, "item:sku:SKU-001").Return(cachedData, nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/sku/SKU-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "FindBySKU")

	var response InventoryItemResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "SKU-001", response.SKU)
}

func TestGetItemBySKU_CacheMiss(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	testItem := createTestItem(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), "SKU-001")

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "item:sku:SKU-001").Return(nil, cache.ErrCacheMiss)
	mockRepo.On("FindBySKU", mock.Anything, "SKU-001").Return(testItem, nil)
	mockCache.On("Set", mock.Anything, "item:sku:SKU-001", mock.Anything, mock.Anything).Return(nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/sku/SKU-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)

	var response InventoryItemResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "SKU-001", response.SKU)
}

func TestGetStockStatus_CacheHit(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	stockStatus := &models.StockStatus{
		ID:        itemID.String(),
		SKU:       "SKU-001",
		Quantity:  100,
		Reserved:  20,
		Available: 80,
		UpdatedAt: time.Now(),
	}

	cachedData, _ := json.Marshal(stockStatus)
	mockCache.On("Get", mock.Anything, "stock:550e8400-e29b-41d4-a716-446655440000").Return(cachedData, nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000/stock", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "GetStockStatus")

	var response StockStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 100, response.Quantity)
	assert.Equal(t, 20, response.Reserved)
	assert.Equal(t, 80, response.Available)
}

func TestGetStockStatus_CacheMiss(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	stockStatus := &models.StockStatus{
		ID:        itemID.String(),
		SKU:       "SKU-001",
		Quantity:  100,
		Reserved:  20,
		Available: 80,
		UpdatedAt: time.Now(),
	}

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "stock:550e8400-e29b-41d4-a716-446655440000").Return(nil, cache.ErrCacheMiss)
	mockRepo.On("GetStockStatus", mock.Anything, itemID).Return(stockStatus, nil)
	mockCache.On("Set", mock.Anything, "stock:550e8400-e29b-41d4-a716-446655440000", mock.Anything, mock.Anything).Return(nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000/stock", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)

	var response StockStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 100, response.Quantity)
}

func TestListItems_Pagination(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	// Create test items
	items := make([]models.InventoryItem, 0)
	for i := 0; i < 25; i++ {
		item := createTestItem(uuid.New(), "SKU-"+string(rune(i+48)))
		items = append(items, *item)
	}

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "items:list:2:10").Return(nil, cache.ErrCacheMiss)
	mockRepo.On("ListItems", mock.Anything, 2, 10).Return(items[10:20], 25, nil)
	mockCache.On("Set", mock.Anything, "items:list:2:10", mock.Anything, mock.Anything).Return(nil)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/inventory/items?page=2&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response ListItemsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 25, response.Total)
	assert.Equal(t, 2, response.Page)
	assert.Equal(t, 10, response.PageSize)
	assert.Equal(t, 3, response.TotalPages) // 25 items / 10 per page = 3 pages
	assert.Len(t, response.Items, 10)
}

func TestListItems_InvalidPagination(t *testing.T) {
	// Setup
	mockCache := new(MockCache)
	mockRepo := new(MockRepository)
	handler := createTestHandler(mockCache, mockRepo)
	router := setupTestRouter(handler)

	testCases := []struct {
		name     string
		query    string
		expected int // expected page/pageSize after normalization
	}{
		{"negative page", "page=-1&page_size=10", 1},
		{"zero page", "page=0&page_size=10", 1},
		{"negative page size", "page=1&page_size=-5", 10},
		{"zero page size", "page=1&page_size=0", 10},
		{"page size too large", "page=1&page_size=200", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mocks for each test case
			mockCache.ExpectedCalls = nil
			mockRepo.ExpectedCalls = nil

			// Mock cache miss
			mockCache.On("Get", mock.Anything, mock.Anything).Return(nil, cache.ErrCacheMiss)
			mockRepo.On("ListItems", mock.Anything, mock.Anything, mock.Anything).Return([]models.InventoryItem{}, 0, nil)
			// Mock cache set (handler will try to cache the response)
			mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			req := httptest.NewRequest("GET", "/api/v1/inventory/items?"+tc.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
