package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRequestIDMiddleware_GenerateID(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := zap.NewNop()
	router.Use(RequestIDMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	// Execute
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Check that request ID is in response header
	responseID := w.Header().Get(RequestIDHeader)
	assert.NotEmpty(t, responseID)
	
	// Verify it's a valid UUID
	_, err := uuid.Parse(responseID)
	assert.NoError(t, err)
}

func TestRequestIDMiddleware_UseProvidedID(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := zap.NewNop()
	router.Use(RequestIDMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	// Execute
	providedID := uuid.New().String()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, providedID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Check that the provided request ID is in response header
	responseID := w.Header().Get(RequestIDHeader)
	assert.Equal(t, providedID, responseID)
}

func TestIdempotencyMiddleware_DuplicateRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := zap.NewNop()
	store := NewInMemoryRequestIDStore()
	requestID := uuid.New().String()

	// Store a response for the request ID
	response := []byte(`{"message":"success"}`)
	err := store.Store(context.Background(), requestID, response, 5*time.Minute)
	assert.NoError(t, err)

	router.Use(RequestIDMiddleware(logger))
	router.Use(IdempotencyMiddleware(store, logger, 5*time.Minute))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "new response"})
	})

	// Execute - first request with stored ID
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set(RequestIDHeader, requestID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert - should return cached response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"message":"success"}`, w.Body.String())
}

func TestIdempotencyMiddleware_NewRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := zap.NewNop()
	store := NewInMemoryRequestIDStore()

	router.Use(RequestIDMiddleware(logger))
	router.Use(IdempotencyMiddleware(store, logger, 5*time.Minute))
	router.Use(StoreResponseMiddleware(store, logger, 5*time.Minute))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "new response"})
	})

	// Execute - new request
	requestID := uuid.New().String()
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set(RequestIDHeader, requestID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert - should process normally
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "new response")

	// Execute - duplicate request
	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set(RequestIDHeader, requestID)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Assert - should return cached response
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w.Body.String(), w2.Body.String())
}

func TestIdempotencyMiddleware_GETRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := zap.NewNop()
	store := NewInMemoryRequestIDStore()

	router.Use(RequestIDMiddleware(logger))
	router.Use(IdempotencyMiddleware(store, logger, 5*time.Minute))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "response"})
	})

	// Execute
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert - GET requests should not be checked for idempotency
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Verify request ID is still in header
	responseID := w.Header().Get(RequestIDHeader)
	assert.NotEmpty(t, responseID)
}

func TestInMemoryRequestIDStore_Expiration(t *testing.T) {
	// Setup
	store := NewInMemoryRequestIDStore()
	requestID := uuid.New().String()
	response := []byte(`{"message":"test"}`)

	// Store with short TTL
	err := store.Store(context.Background(), requestID, response, 100*time.Millisecond)
	assert.NoError(t, err)

	// Verify it exists
	exists, err := store.Exists(context.Background(), requestID)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Verify it no longer exists
	exists, err = store.Exists(context.Background(), requestID)
	assert.NoError(t, err)
	assert.False(t, exists)
}

