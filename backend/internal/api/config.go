package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/middleware"
	"github.com/npinot/vibe/backend/internal/model"
)

type ConfigService interface {
	GetActiveConfig(ctx context.Context, projectID uuid.UUID) (*model.OpenCodeConfig, error)
	CreateOrUpdateConfig(ctx context.Context, config *model.OpenCodeConfig, apiKey string) error
	RollbackToVersion(ctx context.Context, projectID uuid.UUID, version int) error
	GetConfigHistory(ctx context.Context, projectID uuid.UUID) ([]model.OpenCodeConfig, error)
	GetDecryptedAPIKey(ctx context.Context, projectID uuid.UUID) (string, error)
}

type ConfigHandler struct {
	configService ConfigService
}

func NewConfigHandler(configService ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// GetActiveConfig godoc
// @Summary Get active configuration
// @Description Retrieves the currently active OpenCode configuration for a project
// @Tags config
// @Produce json
// @Param id path string true "Project ID"
// @Success 200 {object} model.OpenCodeConfig
// @Failure 404 {object} gin.H{"error": "config not found"}
// @Router /api/projects/{id}/config [get]
func (h *ConfigHandler) GetActiveConfig(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	config, err := h.configService.GetActiveConfig(c.Request.Context(), projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// CreateOrUpdateConfig godoc
// @Summary Create or update configuration
// @Description Creates a new configuration version (deactivating the old one)
// @Tags config
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Param config body CreateConfigRequest true "Configuration"
// @Success 201 {object} model.OpenCodeConfig
// @Failure 400 {object} gin.H{"error": "validation error"}
// @Router /api/projects/{id}/config [post]
func (h *ConfigHandler) CreateOrUpdateConfig(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from context (set by auth middleware)
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	config := &model.OpenCodeConfig{
		ProjectID:      projectID,
		ModelProvider:  req.ModelProvider,
		ModelName:      req.ModelName,
		ModelVersion:   req.ModelVersion,
		APIEndpoint:    req.APIEndpoint,
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
		EnabledTools:   req.EnabledTools,
		ToolsConfig:    req.ToolsConfig,
		SystemPrompt:   req.SystemPrompt,
		MaxIterations:  req.MaxIterations,
		TimeoutSeconds: req.TimeoutSeconds,
		CreatedBy:      user.ID,
	}

	if err := h.configService.CreateOrUpdateConfig(c.Request.Context(), config, req.APIKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// GetConfigHistory godoc
// @Summary List configuration versions
// @Description Retrieves all configuration versions for a project
// @Tags config
// @Produce json
// @Param id path string true "Project ID"
// @Success 200 {array} model.OpenCodeConfig
// @Router /api/projects/{id}/config/versions [get]
func (h *ConfigHandler) GetConfigHistory(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	configs, err := h.configService.GetConfigHistory(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// RollbackConfig godoc
// @Summary Rollback to previous version
// @Description Activates a previous configuration version by creating a new version
// @Tags config
// @Produce json
// @Param id path string true "Project ID"
// @Param version path int true "Version to rollback to"
// @Success 200 {object} gin.H{"message": "config rolled back"}
// @Failure 404 {object} gin.H{"error": "version not found"}
// @Router /api/projects/{id}/config/rollback/{version} [post]
func (h *ConfigHandler) RollbackConfig(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil || version < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	if err := h.configService.RollbackToVersion(c.Request.Context(), projectID, version); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config rolled back successfully"})
}

// Request/Response types
type CreateConfigRequest struct {
	ModelProvider  string      `json:"model_provider" binding:"required,oneof=openai anthropic custom"`
	ModelName      string      `json:"model_name" binding:"required"`
	ModelVersion   *string     `json:"model_version,omitempty"`
	APIEndpoint    *string     `json:"api_endpoint,omitempty"`
	APIKey         string      `json:"api_key,omitempty"` // Not stored in response
	Temperature    float64     `json:"temperature" binding:"min=0,max=2"`
	MaxTokens      int         `json:"max_tokens" binding:"min=1,max=128000"`
	EnabledTools   []string    `json:"enabled_tools" binding:"required"`
	ToolsConfig    model.JSONB `json:"tools_config,omitempty"`
	SystemPrompt   *string     `json:"system_prompt,omitempty"`
	MaxIterations  int         `json:"max_iterations" binding:"min=1,max=50"`
	TimeoutSeconds int         `json:"timeout_seconds" binding:"min=60,max=3600"`
}
