package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestGetRequestID(t *testing.T) {
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
	
	// Verify GetRequestID returns the correct ID
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, providedID, response["request_id"])
}

