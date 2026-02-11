package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const BearerPrefix = "Bearer "

// RequireAuth returns a Gin middleware that checks JWT and sets claims in context.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, BearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization"})
			return
		}
		token := strings.TrimPrefix(auth, BearerPrefix)
		claims, err := ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		role := claims.Role
		if role == "" {
			role = "user"
		}
		c.Set("role", role)
		c.Next()
	}
}

// RequireAdmin aborts with 403 if the user's role is not admin. Must be used after RequireAuth.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		c.Next()
	}
}
