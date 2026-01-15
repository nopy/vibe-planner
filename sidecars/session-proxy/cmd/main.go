package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/npinot/vibe/sidecars/session-proxy/internal/handler"
)

func main() {
	opencodeURL := os.Getenv("OPENCODE_URL")
	if opencodeURL == "" {
		opencodeURL = "http://localhost:3000"
	}

	router := gin.Default()

	sessionHandler := handler.NewSessionHandler(opencodeURL)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	session := router.Group("/session")
	{
		session.GET("", sessionHandler.ListSessions)
		session.POST("", sessionHandler.CreateSession)
		session.GET("/:id", sessionHandler.GetSession)
		session.GET("/:id/events", sessionHandler.StreamEvents)
		session.GET("/:id/pty/connect", sessionHandler.ConnectPTY)
		session.POST("/:id/prompt", sessionHandler.SendPrompt)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}

	log.Printf("Session Proxy Sidecar starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
