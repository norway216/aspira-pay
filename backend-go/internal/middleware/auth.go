// Package middleware provides HTTP middleware for the Gin router.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/service"
)

// AuthRequired is a Gin middleware that validates JWT tokens.
func AuthRequired(jwtMgr *service.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := jwtMgr.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// Set user info in context
		userID, _ := claims["user_id"].(string)
		username, _ := claims["username"].(string)

		c.Set("user_id", userID)
		c.Set("username", username)
		c.Next()
	}
}

// AdminRequired is a Gin middleware that checks for admin role.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In Sandbox, all authenticated users have admin access
		userID := c.GetString("user_id")
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}
