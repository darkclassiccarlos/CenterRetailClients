package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware configura los headers CORS para permitir todas las peticiones
// desde cualquier origen. Esto es necesario cuando el dashboard HTML se sirve
// desde un puerto diferente (8000) y necesita hacer peticiones directas al
// Query Service (8081).
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Configurar headers CORS
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "3600")

		// Manejar preflight requests (OPTIONS)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204) // No Content
			return
		}

		// Continuar con el siguiente handler
		c.Next()
	}
}
