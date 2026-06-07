package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit implements a simple in-memory rate limiter.
// In production, this should use Redis for distributed rate limiting.
func RateLimit(maxRequests int, windowSeconds int) gin.HandlerFunc {
	type clientState struct {
		count    int
		windowStart time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*clientState)
	)

	// Cleanup goroutine
	go func() {
		for {
			time.Sleep(time.Duration(windowSeconds) * time.Second)
			mu.Lock()
			now := time.Now()
			for ip, state := range clients {
				if now.Sub(state.windowStart) > time.Duration(windowSeconds)*time.Second {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		state, exists := clients[ip]
		now := time.Now()

		if !exists || now.Sub(state.windowStart) > time.Duration(windowSeconds)*time.Second {
			clients[ip] = &clientState{count: 1, windowStart: now}
			mu.Unlock()
			c.Next()
			return
		}

		state.count++
		if state.count > maxRequests {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"retry_after_seconds": windowSeconds,
			})
			return
		}
		mu.Unlock()

		c.Next()
	}
}
