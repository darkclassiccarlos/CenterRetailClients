package middleware

import (
	"net/http"

	"listener-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler is a global error handling middleware
func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// RecoveryHandler is a panic recovery middleware
func RecoveryHandler(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("Panic recovered",
			zap.Any("panic", recovered),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
		c.JSON(http.StatusInternalServerError, errors.NewInternalError("internal server error", nil))
	})
}

