package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows the given origin (the frontend's own base URL) to call this
// API — the frontend and backend are separate Traefik-routed subdomains
// (see docker-compose.yml), so every browser request is cross-origin.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
