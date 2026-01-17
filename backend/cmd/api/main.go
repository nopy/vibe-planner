package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/npinot/vibe/backend/internal/api"
	"github.com/npinot/vibe/backend/internal/config"
	"github.com/npinot/vibe/backend/internal/db"
	"github.com/npinot/vibe/backend/internal/middleware"
	"github.com/npinot/vibe/backend/internal/repository"
	"github.com/npinot/vibe/backend/internal/service"
	"github.com/npinot/vibe/backend/internal/static"
)

func main() {
	// Load .env from parent directory (project root)
	if err := godotenv.Load("../.env"); err != nil {
		// Try current directory as fallback
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	cfg := config.Load()

	database, err := db.NewPostgresConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	userRepo := repository.NewUserRepository(database)
	projectRepo := repository.NewProjectRepository(database)

	k8sService, err := service.NewKubernetesService(
		cfg.Kubeconfig,
		cfg.K8SNamespace,
		nil,
	)
	if err != nil {
		log.Printf("Warning: Failed to initialize Kubernetes service: %v", err)
		log.Println("Project management features will be limited")
	}

	projectService := service.NewProjectService(projectRepo, k8sService)

	authService, err := service.NewAuthService(cfg, userRepo)
	if err != nil {
		log.Fatalf("Failed to create auth service: %v", err)
	}

	authMiddleware := middleware.NewAuthMiddleware(cfg, userRepo)
	authHandler := api.NewAuthHandler(authService)
	projectHandler := api.NewProjectHandler(projectService)

	router := setupRouter(cfg, authHandler, projectHandler, authMiddleware)

	// Setup static file serving for production (embedded frontend)
	if cfg.Environment == "production" {
		if err := static.ServeEmbeddedSPA(router); err != nil {
			log.Fatalf("Failed to setup static file serving: %v", err)
		}
		log.Println("Static file serving enabled (embedded SPA)")
	}

	port := cfg.Port
	if port == "" {
		port = "8090"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(cfg *config.Config, authHandler *api.AuthHandler, projectHandler *api.ProjectHandler, authMiddleware *middleware.AuthMiddleware) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.SecurityHeaders())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	v1 := router.Group("/api")
	{
		auth := v1.Group("/auth")
		{
			auth.GET("/oidc/login", authHandler.OIDCLogin)
			auth.GET("/oidc/callback", authHandler.OIDCCallback)
			auth.GET("/me", authMiddleware.JWTAuth(), authHandler.GetCurrentUser)
			auth.POST("/logout", authHandler.Logout)
		}

		projects := v1.Group("/projects", authMiddleware.JWTAuth())
		{
			projects.GET("", projectHandler.ListProjects)
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/:id", projectHandler.GetProject)
			projects.PATCH("/:id", projectHandler.UpdateProject)
			projects.DELETE("/:id", projectHandler.DeleteProject)
			projects.GET("/:id/status", projectHandler.ProjectStatus)

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
