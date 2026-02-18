package auth

import (
	"net/http"
	"time"

	"query-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	jwtManager *JWTManager
	logger     *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtManager *JWTManager, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// LoginRequest represents the login request
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"admin123"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token     string    `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Type      string    `json:"type" example:"Bearer"`
	ExpiresIn int       `json:"expires_in" example:"600"` // 10 minutes in seconds
	ExpiresAt time.Time `json:"expires_at" example:"2024-01-15T12:00:00Z"`
}

// Login handles POST /api/v1/auth/login
// @Summary      Login and get JWT token
// @Description  Autentica un usuario y retorna un token JWT válido por 10 minutos. Usuarios disponibles: admin/admin123, user/user123, operator/operator123
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login credentials"
// @Success      200      {object}  LoginResponse  "Token generado exitosamente"
// @Failure      400      {object}  map[string]string  "Request inválido - credenciales faltantes"
// @Failure      401      {object}  map[string]string  "Credenciales inválidas"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request", zap.Error(err))
		c.Error(errors.NewValidationError("invalid request", "username or password"))
		c.Abort()
		return
	}

	// Simple authentication (for prototype)
	// In production, this should validate against a user database
	if !h.validateCredentials(req.Username, req.Password) {
		h.logger.Warn("Invalid credentials",
			zap.String("username", req.Username),
		)
		c.Error(errors.NewStandardError("Unauthorized", "invalid credentials", "username or password incorrect"))
		c.Abort()
		c.JSON(http.StatusUnauthorized, errors.NewStandardError("Unauthorized", "invalid credentials", "username or password incorrect"))
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.GenerateToken(req.Username)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		c.Error(errors.NewInternalError("failed to generate token", err))
		c.Abort()
		return
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	response := LoginResponse{
		Token:     token,
		Type:      "Bearer",
		ExpiresIn: 600, // 10 minutes in seconds
		ExpiresAt: expiresAt,
	}

	h.logger.Info("User logged in successfully",
		zap.String("username", req.Username),
		zap.Time("expires_at", expiresAt),
	)

	c.JSON(http.StatusOK, response)
}

// validateCredentials validates user credentials
// For prototype: simple hardcoded validation
// In production: validate against user database
func (h *AuthHandler) validateCredentials(username, password string) bool {
	// Simple validation for prototype
	// In production, this should query a user database
	validUsers := map[string]string{
		"admin":    "admin123",
		"user":     "user123",
		"operator": "operator123",
	}

	expectedPassword, exists := validUsers[username]
	if !exists {
		return false
	}

	return password == expectedPassword
}

