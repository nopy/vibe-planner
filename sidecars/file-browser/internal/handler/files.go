package handler

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/npinot/vibe/sidecars/file-browser/internal/service"
)

type FileHandler struct {
	fileService *service.FileService
}

func NewFileHandler(workspaceDir string) *FileHandler {
	return &FileHandler{
		fileService: service.NewFileService(workspaceDir),
	}
}

func (h *FileHandler) GetTree(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	maxDepth := 5

	tree, err := h.fileService.GetTree(path, maxDepth)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, tree)
}

func (h *FileHandler) GetContent(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(400, gin.H{"error": "path parameter is required"})
		return
	}

	content, err := h.fileService.ReadFile(path)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"content": content, "path": path})
}

func (h *FileHandler) WriteFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileService.WriteFile(req.Path, req.Content); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "File written successfully", "path": req.Path})
}

func (h *FileHandler) DeleteFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(400, gin.H{"error": "path parameter is required"})
		return
	}

	if err := h.fileService.DeleteFile(path); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "File deleted successfully"})
}

func (h *FileHandler) CreateDirectory(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := os.MkdirAll(filepath.Join(h.fileService.WorkspaceDir, req.Path), 0755); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Directory created successfully"})
}
