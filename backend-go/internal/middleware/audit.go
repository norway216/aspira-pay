package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditLog logs all API requests for audit purposes.
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()

		log.Printf("[AUDIT] %s %s | %d | %v | %s",
			method, path, status, latency, clientIP)
	}
}
