package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// RequestIDHeader is the HTTP header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDContextKey is the context key for request ID
	RequestIDContextKey = "request_id"
)

// RequestIDStore stores processed request IDs for idempotency
type RequestIDStore interface {
	// Store stores a request ID with its response
	Store(ctx context.Context, requestID string, response []byte, ttl time.Duration) error
	// Get retrieves a stored response by request ID
	Get(ctx context.Context, requestID string) ([]byte, error)
	// Exists checks if a request ID exists
	Exists(ctx context.Context, requestID string) (bool, error)
}

// InMemoryRequestIDStore is an in-memory implementation of RequestIDStore
type InMemoryRequestIDStore struct {
	mu      sync.RWMutex
	store   map[string]requestIDEntry
	cleanup *time.Ticker
}

type requestIDEntry struct {
	response  []byte
	expiresAt time.Time
}

// NewInMemoryRequestIDStore creates a new in-memory request ID store
func NewInMemoryRequestIDStore() *InMemoryRequestIDStore {
	store := &InMemoryRequestIDStore{
		store:   make(map[string]requestIDEntry),
		cleanup: time.NewTicker(1 * time.Minute), // Cleanup every minute
	}

	// Start cleanup goroutine
	go store.cleanupExpired()

	return store
}

func (s *InMemoryRequestIDStore) Store(ctx context.Context, requestID string, response []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[requestID] = requestIDEntry{
		response:  response,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (s *InMemoryRequestIDStore) Get(ctx context.Context, requestID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.store[requestID]
	if !exists {
		return nil, ErrRequestIDNotFound
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		delete(s.store, requestID)
		return nil, ErrRequestIDNotFound
	}

	return entry.response, nil
}

func (s *InMemoryRequestIDStore) Exists(ctx context.Context, requestID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.store[requestID]
	if !exists {
		return false, nil
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		delete(s.store, requestID)
		return false, nil
	}

	return true, nil
}

func (s *InMemoryRequestIDStore) cleanupExpired() {
	for range s.cleanup.C {
		s.mu.Lock()
		now := time.Now()
		for id, entry := range s.store {
			if now.After(entry.expiresAt) {
				delete(s.store, id)
			}
		}
		s.mu.Unlock()
	}
}

var (
	ErrRequestIDNotFound = &RequestIDError{Message: "request ID not found"}
)

type RequestIDError struct {
	Message string
}

func (e *RequestIDError) Error() string {
	return e.Message
}

// RequestIDMiddleware extracts or generates X-Request-ID header
func RequestIDMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract or generate request ID
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			// Generate new request ID if not provided
			requestID = uuid.New().String()
			logger.Debug("Generated new request ID",
				zap.String("request_id", requestID),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
		} else {
			logger.Debug("Using provided request ID",
				zap.String("request_id", requestID),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
		}

		// Store request ID in context
		c.Set(RequestIDContextKey, requestID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), RequestIDContextKey, requestID))

		// Add request ID to response header
		c.Header(RequestIDHeader, requestID)

		// Add request ID to logger context
		c.Next()
	}
}

// GetRequestID retrieves the request ID from the Gin context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDContextKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// IdempotencyMiddleware checks for duplicate requests based on X-Request-ID
func IdempotencyMiddleware(store RequestIDStore, logger *zap.Logger, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply idempotency to write operations (POST, PUT, DELETE, PATCH)
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		requestID := GetRequestID(c)
		if requestID == "" {
			// No request ID, continue normally
			c.Next()
			return
		}

		// Check if request ID already exists
		exists, err := store.Exists(c.Request.Context(), requestID)
		if err != nil {
			logger.Warn("Error checking request ID existence",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
			// Continue on error (fail open)
			c.Next()
			return
		}

		if exists {
			// Request ID exists, retrieve cached response
			cachedResponse, err := store.Get(c.Request.Context(), requestID)
			if err == nil && len(cachedResponse) > 0 {
				logger.Info("Duplicate request detected, returning cached response",
					zap.String("request_id", requestID),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// Set content type and return cached response
				c.Header("Content-Type", "application/json")
				c.Data(200, "application/json", cachedResponse)
				c.Abort()
				return
			}
		}

		// Request ID doesn't exist, continue processing
		// We'll store the response after processing
		c.Next()
	}
}

// StoreResponseMiddleware stores the response for idempotency
func StoreResponseMiddleware(store RequestIDStore, logger *zap.Logger, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only store responses for write operations
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		requestID := GetRequestID(c)
		if requestID == "" {
			c.Next()
			return
		}

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
		}
		c.Writer = writer

		c.Next()

		// Only store successful responses (2xx)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			if len(writer.body) > 0 {
				// Store response for idempotency
				if err := store.Store(c.Request.Context(), requestID, writer.body, ttl); err != nil {
					logger.Warn("Failed to store response for idempotency",
						zap.String("request_id", requestID),
						zap.Error(err),
					)
				} else {
					logger.Debug("Stored response for idempotency",
						zap.String("request_id", requestID),
						zap.String("path", c.Request.URL.Path),
						zap.String("method", c.Request.Method),
						zap.Int("status", c.Writer.Status()),
					)
				}
			}
		}
	}
}

// responseWriter captures the response body
type responseWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body = append(w.body, []byte(s)...)
	return w.ResponseWriter.WriteString(s)
}
