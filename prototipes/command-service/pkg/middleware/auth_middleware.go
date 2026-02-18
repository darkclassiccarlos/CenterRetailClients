package middleware

import (
	"net/http"
	"strings"

	"command-service/internal/auth"
	"command-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtManager *auth.JWTManager, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Missing authorization header",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			c.JSON(http.StatusUnauthorized, errors.NewStandardError("Unauthorized", "missing authorization header", "Header: Authorization"))
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Warn("Invalid authorization header format",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			c.JSON(http.StatusUnauthorized, errors.NewStandardError("Unauthorized", "invalid authorization header format", "Expected: Bearer <token>"))
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				logger.Warn("Token expired",
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				c.JSON(http.StatusUnauthorized, errors.NewStandardError("Unauthorized", "token expired", "Token has expired, please login again"))
				c.Abort()
				return
			}

			logger.Warn("Invalid token",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Error(err),
			)
			c.JSON(http.StatusUnauthorized, errors.NewStandardError("Unauthorized", "invalid token", err.Error()))
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("username", claims.Username)
		c.Set("user_id", claims.Subject)

		logger.Debug("Token validated",
			zap.String("username", claims.Username),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)

		c.Next()
	}
}

