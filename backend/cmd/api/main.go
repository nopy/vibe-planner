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
	taskRepo := repository.NewTaskRepository(database)
	sessionRepo := repository.NewSessionRepository(database)
	configRepo := repository.NewConfigRepository(database)
	interactionRepo := repository.NewInteractionRepository(database)

	k8sService, err := service.NewKubernetesService(
		cfg.Kubeconfig,
		cfg.K8SNamespace,
		&service.KubernetesConfig{
			Namespace:         cfg.K8SNamespace,
			OpenCodeImage:     cfg.OpenCodeServerImage,
			FileBrowserImage:  cfg.FileBrowserImage,
			SessionProxyImage: cfg.SessionProxyImage,
			WorkspaceSize:     "1Gi",
			CPULimit:          "1000m",
			MemoryLimit:       "1Gi",
			CPURequest:        "100m",
			MemoryRequest:     "256Mi",
		},
	)
	if err != nil {
		log.Printf("Warning: Failed to initialize Kubernetes service: %v", err)
		log.Println("Project management features will be limited")
	}

	configService, err := service.NewConfigService(configRepo, cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize config service: %v", err)
	}

	sessionService := service.NewSessionService(sessionRepo, taskRepo, projectRepo, k8sService, configService, cfg.OpenCodeSharedSecret)
	projectService := service.NewProjectService(projectRepo, k8sService)
	taskService := service.NewTaskService(taskRepo, projectRepo, sessionService)
	interactionService := service.NewInteractionService(interactionRepo, taskRepo, projectRepo, sessionRepo)

	authService, err := service.NewAuthService(cfg, userRepo)
	if err != nil {
		log.Printf("Warning: Failed to create auth service: %v", err)
		log.Println("Authentication features will be disabled")
	}

	authMiddleware := middleware.NewAuthMiddleware(cfg, userRepo)
	authHandler := api.NewAuthHandler(authService)
	projectHandler := api.NewProjectHandler(projectService)
	taskHandler := api.NewTaskHandler(taskService, projectRepo, k8sService)
	fileHandler := api.NewFileHandler(projectRepo, k8sService)
	configHandler := api.NewConfigHandler(configService)
	interactionHandler := api.NewInteractionHandler(interactionService)
	sessionHandler := api.NewSessionHandler(sessionService)

	router := setupRouter(cfg, authHandler, projectHandler, taskHandler, fileHandler, configHandler, interactionHandler, sessionHandler, authMiddleware)

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

func setupRouter(cfg *config.Config, authHandler *api.AuthHandler, projectHandler *api.ProjectHandler, taskHandler *api.TaskHandler, fileHandler *api.FileHandler, configHandler *api.ConfigHandler, interactionHandler *api.InteractionHandler, sessionHandler *api.SessionHandler, authMiddleware *middleware.AuthMiddleware) *gin.Engine {
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

		sessions := v1.Group("/sessions")
		{
			sessions.GET("/active", sessionHandler.GetActiveSessions)
			sessions.PATCH("/:id/status", sessionHandler.UpdateSessionStatus)
			sessions.PATCH("/:id/event-id", sessionHandler.UpdateLastEventID)
		}

		projects := v1.Group("/projects", authMiddleware.JWTAuth())
		{
			projects.GET("", projectHandler.ListProjects)
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/:id", projectHandler.GetProject)
			projects.PATCH("/:id", projectHandler.UpdateProject)
			projects.DELETE("/:id", projectHandler.DeleteProject)
			projects.GET("/:id/status", projectHandler.ProjectStatus)

			projects.GET("/:id/tasks", taskHandler.ListTasks)
			projects.POST("/:id/tasks", taskHandler.CreateTask)
			projects.GET("/:id/tasks/stream", taskHandler.TaskUpdatesStream)
			projects.GET("/:id/tasks/:taskId", taskHandler.GetTask)
			projects.PATCH("/:id/tasks/:taskId", taskHandler.UpdateTask)
			projects.PATCH("/:id/tasks/:taskId/move", taskHandler.MoveTask)
			projects.DELETE("/:id/tasks/:taskId", taskHandler.DeleteTask)
			projects.POST("/:id/tasks/:taskId/execute", taskHandler.ExecuteTask)
			projects.POST("/:id/tasks/:taskId/stop", taskHandler.StopTask)
			projects.GET("/:id/tasks/:taskId/output", taskHandler.TaskOutputStream)
			projects.GET("/:id/tasks/:taskId/sessions", taskHandler.GetTaskSessions)
			projects.GET("/:id/tasks/:taskId/interactions", interactionHandler.GetTaskHistory)
			projects.GET("/:id/tasks/:taskId/interact", interactionHandler.TaskInteractionWebSocket)

			projects.GET("/:id/files/tree", fileHandler.GetTree)
			projects.GET("/:id/files/content", fileHandler.GetContent)
			projects.GET("/:id/files/info", fileHandler.GetFileInfo)
			projects.POST("/:id/files/write", fileHandler.WriteFile)
			projects.DELETE("/:id/files", fileHandler.DeleteFile)
			projects.POST("/:id/files/mkdir", fileHandler.CreateDirectory)
			projects.GET("/:id/files/watch", fileHandler.FileChangesStream)

			projects.GET("/:id/config", configHandler.GetActiveConfig)
			projects.POST("/:id/config", configHandler.CreateOrUpdateConfig)
			projects.GET("/:id/config/versions", configHandler.GetConfigHistory)
			projects.POST("/:id/config/rollback/:version", configHandler.RollbackConfig)
		}
	}

	return router
}
