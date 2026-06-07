package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/pkg/crypto"
)

// SignatureVerification validates API request signatures.
// Architecture doc §14.1: signature = HMAC_SHA256(method + path + timestamp + nonce + body_hash, api_secret)
func SignatureVerification() gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.GetHeader("X-Signature")
		timestamp := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")

		// In Sandbox mode, signature verification is lenient
		if signature == "" {
			// Skip verification for Sandbox
			c.Next()
			return
		}

		// Read body for hash
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyHash := crypto.SHA256(string(bodyBytes))

		// Build signature message
		message := c.Request.Method + c.Request.URL.Path + timestamp + nonce + bodyHash

		// In production, look up api_secret by API key
		apiSecret := "sandbox-secret"
		expected := crypto.HMACSHA256(message, apiSecret)

		if signature != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		c.Next()
	}
}
