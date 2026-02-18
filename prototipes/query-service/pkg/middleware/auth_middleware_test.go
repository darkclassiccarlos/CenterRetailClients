package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"query-service/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupAuthMiddlewareTestRouter(jwtManager *auth.JWTManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Protected route
	protected := router.Group("/api/v1")
	protected.Use(AuthMiddleware(jwtManager, zap.NewNop()))
	{
		protected.GET("/test", func(c *gin.Context) {
			username, _ := c.Get("username")
			c.JSON(http.StatusOK, gin.H{
				"message":  "success",
				"username": username,
			})
		})
	}

	return router
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := auth.NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	router := setupAuthMiddlewareTestRouter(jwtManager)

	// Generate token
	token, err := jwtManager.GenerateToken("admin")
	assert.NoError(t, err)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := auth.NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	router := setupAuthMiddlewareTestRouter(jwtManager)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := auth.NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	router := setupAuthMiddlewareTestRouter(jwtManager)

	testCases := []struct {
		name   string
		header string
	}{
		{"no Bearer prefix", "invalid-token"},
		{"wrong prefix", "Token invalid-token"},
		{"empty token", "Bearer "},
		{"multiple spaces", "Bearer  token"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := auth.NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	router := setupAuthMiddlewareTestRouter(jwtManager)

	testCases := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid token", "invalid.token.here"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIn0.wrong"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestAuthMiddleware_ContextValues(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := auth.NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	protected := router.Group("/api/v1")
	protected.Use(AuthMiddleware(jwtManager, logger))
	{
		protected.GET("/test", func(c *gin.Context) {
			username, exists := c.Get("username")
			assert.True(t, exists)

			userID, exists := c.Get("user_id")
			assert.True(t, exists)

			c.JSON(http.StatusOK, gin.H{
				"username": username,
				"user_id":  userID,
			})
		})
	}

	// Generate token
	token, err := jwtManager.GenerateToken("admin")
	assert.NoError(t, err)

	// Execute
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}
