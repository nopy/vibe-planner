package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/npinot/vibe/sidecars/file-browser/internal/handler"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	workspaceDir := os.Getenv("WORKSPACE_DIR")
	if workspaceDir == "" {
		workspaceDir = "/workspace"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "debug" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	if logLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	fileHandler := handler.NewFileHandler(workspaceDir)

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.GET("/ready", func(c *gin.Context) {
		if _, err := os.Stat(workspaceDir); err != nil {
			slog.Error("Workspace directory not accessible", "error", err, "path", workspaceDir)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "workspace not accessible"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	files := router.Group("/files")
	{
		files.GET("/tree", fileHandler.GetTree)
		files.GET("/content", fileHandler.GetContent)
		files.GET("/info", fileHandler.GetFileInfo)
		files.POST("/write", fileHandler.WriteFile)
		files.DELETE("", fileHandler.DeleteFile)
		files.POST("/mkdir", fileHandler.CreateDirectory)
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		slog.Info("File Browser Sidecar starting", "port", port, "workspace", workspaceDir)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down File Browser Sidecar...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("File Browser Sidecar stopped gracefully")
}
