// Package auth verifies RS256 tokens issued by auth.marketmaker.cc.
package auth

import "github.com/golang-jwt/jwt/v5"

// Claims mirrors the structure produced by auth.marketmaker.cc.
// `services` is a map of service-name -> role; the popularity backend
// checks for membership of its own service name before authorising.
type Claims struct {
	UserID   string            `json:"sub"`
	Email    string            `json:"email"`
	Username string            `json:"username"`
	Services map[string]string `json:"services"`
	jwt.RegisteredClaims
}
