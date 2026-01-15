package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/npinot/vibe/sidecars/file-browser/internal/handler"
)

func main() {
	workspaceDir := os.Getenv("WORKSPACE_DIR")
	if workspaceDir == "" {
		workspaceDir = "/workspace"
	}

	router := gin.Default()

	fileHandler := handler.NewFileHandler(workspaceDir)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	files := router.Group("/files")
	{
		files.GET("/tree", fileHandler.GetTree)
		files.GET("/content", fileHandler.GetContent)
		files.POST("/write", fileHandler.WriteFile)
		files.DELETE("", fileHandler.DeleteFile)
		files.POST("/mkdir", fileHandler.CreateDirectory)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("File Browser Sidecar starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
