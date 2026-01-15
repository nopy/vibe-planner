package api

import "github.com/gin-gonic/gin"

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	// Add dependencies here (auth service, etc.)
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// OIDCLogin initiates OIDC login flow
func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// OIDCCallback handles OIDC callback
func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// Logout logs out the current user
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}
