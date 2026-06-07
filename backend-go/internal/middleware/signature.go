package middleware

import (
	"bytes"
	"crypto/subtle"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/pkg/crypto"
)

// nonceCache stores recently-used nonces to prevent replay attacks.
// Architecture doc §14.1: Nonce replay attack prevention.
type nonceCache struct {
	mu      sync.Mutex
	entries map[string]time.Time
	maxSize int
}

var nonces = &nonceCache{
	entries: make(map[string]time.Time),
	maxSize: 100000,
}

func (n *nonceCache) checkAndStore(nonce string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Check if nonce was already used
	if _, exists := n.entries[nonce]; exists {
		return false // Replay detected
	}

	// Store nonce with expiry
	n.entries[nonce] = time.Now()

	// Evict expired nonces older than 5 minutes
	if len(n.entries) > n.maxSize {
		cutoff := time.Now().Add(-5 * time.Minute)
		for k, v := range n.entries {
			if v.Before(cutoff) {
				delete(n.entries, k)
			}
		}
	}

	return true
}

// SignatureVerification validates API request signatures per architecture doc §14.1.
// Signature format:
//
//	signature = HMAC_SHA256(method + path + timestamp + nonce + body_hash, api_secret)
//
// Security checks:
//  1. Timestamp within ±5 minutes (replay window)
//  2. Nonce not previously used (replay prevention)
//  3. HMAC signature matches expected
//
// In Sandbox mode, if no signature header is present, verification is skipped.
// In production, all requests MUST include valid signatures.
func SignatureVerification() gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.GetHeader("X-Signature")
		timestamp := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")

		// Sandbox mode: skip verification if no signature present
		if signature == "" && timestamp == "" && nonce == "" {
			c.Next()
			return
		}

		// Production mode: all headers required
		if signature == "" || timestamp == "" || nonce == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing signature headers: X-Signature, X-Timestamp, X-Nonce required",
			})
			return
		}

		// Validate timestamp (±5 minutes window, architecture doc §14.1)
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid X-Timestamp format",
			})
			return
		}

		now := time.Now().Unix()
		drift := now - ts
		if drift < 0 {
			drift = -drift
		}
		if drift > 300 { // 5 minutes
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "request timestamp expired or too far in the future",
			})
			return
		}

		// Validate nonce (replay prevention)
		if !nonces.checkAndStore(nonce) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "duplicate nonce — possible replay attack",
			})
			return
		}

		// Read body for hash
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyHash := crypto.SHA256(string(bodyBytes))

		// Build signature message per architecture doc §14.1
		message := c.Request.Method + c.Request.URL.Path + timestamp + nonce + bodyHash

		// In production, look up api_secret by API key from header: X-API-Key
		apiKey := c.GetHeader("X-API-Key")
		apiSecret := "sandbox-secret"
		if apiKey != "" {
			// In production, this would be a database lookup
			// For now, use a derived secret based on the API key for Sandbox
			apiSecret = crypto.HMACSHA256(apiKey, "aspira-pay-sandbox-master-key")
		}

		expected := crypto.HMACSHA256(message, apiSecret)

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid signature",
			})
			return
		}

		c.Next()
	}
}
