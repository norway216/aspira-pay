package middleware

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/pkg/crypto"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// IdempotencyMiddleware checks for idempotency key in request headers.
// In production, this would check against a database.
func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			// Auto-generate if not provided (for Sandbox convenience)
			key = idgen.RequestID()
			c.Header("Idempotency-Key", key)
		}

		// Read and hash request body for idempotency check
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		bodyHash := crypto.SHA256(string(bodyBytes))
		_ = bodyHash // In production: check against idempotency_keys table

		// Store in context for handler use
		c.Set("idempotency_key", key)
		c.Set("request_hash", bodyHash)

		c.Next()
	}
}
