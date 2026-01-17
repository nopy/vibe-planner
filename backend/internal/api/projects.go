package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/npinot/vibe/backend/internal/middleware"
	"github.com/npinot/vibe/backend/internal/service"
)

// ProjectHandler handles project-related requests
type ProjectHandler struct {
	projectService service.ProjectService
}

func NewProjectHandler(projectService service.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	RepoURL     string `json:"repo_url"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	RepoURL     *string `json:"repo_url"`
}

// ListProjects returns all projects for the current user
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	projects, err := h.projectService.ListProjects(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, projects)
}

// CreateProject creates a new project
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	project, err := h.projectService.CreateProject(
		c.Request.Context(),
		user.ID,
		req.Name,
		req.Description,
		req.RepoURL,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidProjectName):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidRepoURL):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		}
		return
	}

	c.JSON(http.StatusCreated, project)
}

// GetProject returns a specific project
func (h *ProjectHandler) GetProject(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	project, err := h.projectService.GetProject(c.Request.Context(), projectID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project"})
		}
		return
	}

	c.JSON(http.StatusOK, project)
}

// UpdateProject updates a project
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.RepoURL != nil {
		updates["repo_url"] = *req.RepoURL
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	project, err := h.projectService.UpdateProject(c.Request.Context(), projectID, user.ID, updates)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		case errors.Is(err, service.ErrInvalidProjectName):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidRepoURL):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		}
		return
	}

	c.JSON(http.StatusOK, project)
}

// DeleteProject deletes a project
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	err = h.projectService.DeleteProject(c.Request.Context(), projectID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *ProjectHandler) ProjectStatus(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	project, err := h.projectService.GetProject(c.Request.Context(), projectID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project"})
		}
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	if project.PodName == "" || project.PodNamespace == "" {
		conn.WriteJSON(gin.H{
			"status": string(project.Status),
			"error":  project.PodError,
		})
		return
	}

	conn.WriteJSON(gin.H{
		"status": project.PodStatus,
		"error":  project.PodError,
	})
}
