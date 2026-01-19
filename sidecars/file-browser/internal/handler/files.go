package handler

import (
	"errors"
	"net/http"

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
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, tree)
}

func (h *FileHandler) GetContent(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	content, err := h.fileService.ReadFile(path)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content, "path": path})
}

func (h *FileHandler) GetFileInfo(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	info, err := h.fileService.GetFileInfo(path)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, info)
}

func (h *FileHandler) WriteFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileService.WriteFile(req.Path, req.Content); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File written successfully", "path": req.Path})
}

func (h *FileHandler) DeleteFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	if err := h.fileService.DeleteFile(path); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

func (h *FileHandler) CreateDirectory(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileService.CreateDirectory(req.Path); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Directory created successfully", "path": req.Path})
}

func (h *FileHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidPath):
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid path: directory traversal detected"})
	case errors.Is(err, service.ErrPathRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "File or directory not found"})
	case errors.Is(err, service.ErrNotDirectory):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is not a directory"})
	case errors.Is(err, service.ErrMaxDepthZero):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Max depth must be greater than zero"})
	case errors.Is(err, service.ErrFileTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File exceeds maximum size limit (10MB)"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}
