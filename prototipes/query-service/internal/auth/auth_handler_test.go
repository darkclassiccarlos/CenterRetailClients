package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"query-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupAuthTestRouter(handler *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := zap.NewNop()
	// Error handler middleware (inline to avoid import cycle)
	router.Use(func(c *gin.Context) {
		c.Next()
		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			// Check if it's a StandardError
			if stdErr, ok := err.(*errors.StandardError); ok {
				logger.Warn("Request error",
					zap.String("error_code", stdErr.Code),
					zap.String("message", stdErr.Message),
					zap.String("details", stdErr.Details),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				c.JSON(stdErr.HTTPStatus(), stdErr)
				return
			}
			// Handle other errors
			logger.Error("Unhandled error",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			c.JSON(http.StatusInternalServerError, errors.NewInternalError("internal server error", err))
		}
	})
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", handler.Login)
		}
	}
	return router
}

func TestLogin_Success(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	handler := NewAuthHandler(jwtManager, logger)
	router := setupAuthTestRouter(handler)

	// Test data
	loginReq := LoginRequest{
		Username: "admin",
		Password: "admin123",
	}

	body, _ := json.Marshal(loginReq)

	// Execute
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "Bearer", response.Type)
	assert.Equal(t, 600, response.ExpiresIn) // 10 minutes in seconds
}

func TestLogin_InvalidCredentials(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	handler := NewAuthHandler(jwtManager, logger)
	router := setupAuthTestRouter(handler)

	testCases := []struct {
		name         string
		username     string
		password     string
		expectedCode int
	}{
		{"wrong password", "admin", "wrongpassword", http.StatusUnauthorized},
		{"wrong username", "wronguser", "admin123", http.StatusUnauthorized},
		{"both wrong", "wronguser", "wrongpassword", http.StatusUnauthorized},
		{"empty username", "", "admin123", http.StatusBadRequest}, // Empty fields are validation errors (400)
		{"empty password", "admin", "", http.StatusBadRequest},    // Empty fields are validation errors (400)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loginReq := LoginRequest{
				Username: tc.username,
				Password: tc.password,
			}

			body, _ := json.Marshal(loginReq)

			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code, "Expected HTTP %d but got %d for test case: %s", tc.expectedCode, w.Code, tc.name)
		})
	}
}

func TestLogin_ValidUsers(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	handler := NewAuthHandler(jwtManager, logger)
	router := setupAuthTestRouter(handler)

	validUsers := []struct {
		username string
		password string
	}{
		{"admin", "admin123"},
		{"user", "user123"},
		{"operator", "operator123"},
	}

	for _, user := range validUsers {
		t.Run(user.username, func(t *testing.T) {
			loginReq := LoginRequest{
				Username: user.username,
				Password: user.password,
			}

			body, _ := json.Marshal(loginReq)

			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response LoginResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.NotEmpty(t, response.Token)
		})
	}
}

func TestLogin_InvalidRequest(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)
	handler := NewAuthHandler(jwtManager, logger)
	router := setupAuthTestRouter(handler)

	testCases := []struct {
		name         string
		body         string
		expectedCode int
	}{
		{"empty body", "", http.StatusBadRequest},
		{"invalid JSON", `{"username":}`, http.StatusBadRequest},
		{"missing username", `{"password":"admin123"}`, http.StatusBadRequest},
		{"missing password", `{"username":"admin"}`, http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should return 400 for invalid requests
			assert.Equal(t, tc.expectedCode, w.Code, "Expected HTTP %d but got %d for test case: %s", tc.expectedCode, w.Code, tc.name)
		})
	}
}

func TestJWTManager_GenerateAndValidateToken(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)

	// Generate token
	token, err := jwtManager.GenerateToken("admin")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token
	claims, err := jwtManager.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Username)
	assert.Equal(t, "admin", claims.Subject)
	assert.Equal(t, "query-service", claims.Issuer)
}

func TestJWTManager_InvalidToken(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager := NewJWTManager("test-secret-key-min-32-chars-for-testing", logger)

	testCases := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "invalid.token.format"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIn0.wrongsignature"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := jwtManager.ValidateToken(tc.token)
			assert.Error(t, err)
			assert.Equal(t, ErrInvalidToken, err)
		})
	}
}

func TestJWTManager_TokenWithDifferentSecret(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	jwtManager1 := NewJWTManager("secret-key-1-min-32-chars-for-testing", logger)
	jwtManager2 := NewJWTManager("secret-key-2-min-32-chars-for-testing", logger)

	// Generate token with manager 1
	token, err := jwtManager1.GenerateToken("admin")
	require.NoError(t, err)

	// Try to validate with manager 2 (different secret)
	_, err = jwtManager2.ValidateToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}
