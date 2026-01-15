package api

import "github.com/gin-gonic/gin"

// ProjectHandler handles project-related requests
type ProjectHandler struct {
	// Add dependencies here (project service, etc.)
}

func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{}
}

// ListProjects returns all projects for the current user
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// CreateProject creates a new project
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// GetProject returns a specific project
func (h *ProjectHandler) GetProject(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// UpdateProject updates a project
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// DeleteProject deletes a project
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}
