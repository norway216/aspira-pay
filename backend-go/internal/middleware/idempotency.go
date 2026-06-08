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
//
// Optimization: Skip body hashing for GET/HEAD/OPTIONS requests since they
// have no request body and are naturally idempotent per HTTP semantics.
func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			// Auto-generate if not provided (for Sandbox convenience)
			key = idgen.RequestID()
			c.Header("Idempotency-Key", key)
		}

		// GET/HEAD/OPTIONS are naturally idempotent — skip expensive body hash
		method := c.Request.Method
		skipHash := method == "GET" || method == "HEAD" || method == "OPTIONS"

		var bodyHash string
		if !skipHash {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if len(bodyBytes) > 0 {
				bodyHash = crypto.SHA256(string(bodyBytes))
			}
		}

		// Store in context for handler use
		c.Set("idempotency_key", key)
		c.Set("request_hash", bodyHash)

		c.Next()
	}
}
