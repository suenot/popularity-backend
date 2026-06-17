// Package middleware contains gin middlewares.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/suenot/popularity-backend/internal/auth"
)

// Context keys for downstream handlers.
const (
	CtxUserID   = "auth.user_id"
	CtxEmail    = "auth.email"
	CtxUsername = "auth.username"
	CtxClaims   = "auth.claims"
)

// JWT returns a middleware that requires a Bearer token verified by `v`.
// If `serviceName` is non-empty, the token's `services` map must contain it.
func JWT(v *auth.Verifier, serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		raw := strings.TrimPrefix(h, "Bearer ")
		claims, err := v.Parse(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if serviceName != "" {
			if _, ok := claims.Services[serviceName]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "service not granted"})
				return
			}
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxEmail, claims.Email)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxClaims, claims)
		c.Next()
	}
}

// OptionalJWT returns a middleware that validates a Bearer token when present,
// but falls back to a synthetic dev user when the header is absent.
func OptionalJWT(v *auth.Verifier, serviceName, devUserID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.Set(CtxUserID, devUserID)
			c.Set(CtxEmail, "dev@local")
			c.Set(CtxUsername, "dev")
			c.Next()
			return
		}
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		raw := strings.TrimPrefix(h, "Bearer ")
		claims, err := v.Parse(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if serviceName != "" {
			if _, ok := claims.Services[serviceName]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "service not granted"})
				return
			}
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxEmail, claims.Email)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxClaims, claims)
		c.Next()
	}
}

// UserID returns the authenticated user ID from gin context.
func UserID(c *gin.Context) string {
	v, _ := c.Get(CtxUserID)
	s, _ := v.(string)
	return s
}
