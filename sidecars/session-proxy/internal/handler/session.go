package handler

import (
	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	opencodeURL string
}

func NewSessionHandler(opencodeURL string) *SessionHandler {
	return &SessionHandler{
		opencodeURL: opencodeURL,
	}
}

func (h *SessionHandler) ListSessions(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (h *SessionHandler) CreateSession(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (h *SessionHandler) GetSession(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (h *SessionHandler) StreamEvents(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (h *SessionHandler) ConnectPTY(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (h *SessionHandler) SendPrompt(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}
