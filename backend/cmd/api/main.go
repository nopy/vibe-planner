package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/npinot/vibe/backend/internal/config"
	"github.com/npinot/vibe/backend/internal/db"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	database, err := db.NewPostgresConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations (auto-migrate GORM models)
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Gin router
	router := setupRouter(cfg)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	// API v1 routes
	v1 := router.Group("/api")
	{
		// Auth routes (to be implemented)
		auth := v1.Group("/auth")
		{
			auth.POST("/oidc/login", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			auth.POST("/oidc/callback", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			auth.GET("/me", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			auth.POST("/logout", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
		}

		// Projects routes (to be implemented)
		projects := v1.Group("/projects")
		{
			projects.GET("", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.POST("", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.GET("/:id", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.PATCH("/:id", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.DELETE("/:id", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})

			// Tasks routes (to be implemented)
			projects.GET("/:id/tasks", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.POST("/:id/tasks", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.GET("/:id/tasks/:taskId", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.PATCH("/:id/tasks/:taskId", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.DELETE("/:id/tasks/:taskId", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
			projects.POST("/:id/tasks/:taskId/execute", func(c *gin.Context) {
				c.JSON(501, gin.H{"error": "Not implemented yet"})
			})
		}
	}

	return router
}
