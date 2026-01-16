package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/npinot/vibe/backend/internal/middleware"
	"github.com/npinot/vibe/backend/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	state := c.Query("state")

	authURL, err := h.authService.GetAuthorizationURL(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authorization URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authorization_url": authURL,
	})
}

func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing authorization code"})
		return
	}

	user, token, err := h.authService.ExchangeCodeForToken(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to exchange code for token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
