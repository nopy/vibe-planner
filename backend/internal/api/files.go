package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/npinot/vibe/backend/internal/middleware"
	"github.com/npinot/vibe/backend/internal/repository"
	"github.com/npinot/vibe/backend/internal/service"
)

// FileHandler handles file-related HTTP requests by proxying to the file-browser sidecar
type FileHandler struct {
	projectRepo repository.ProjectRepository
	k8sService  service.KubernetesService
	httpClient  *http.Client
	sidecarPort int
}

// NewFileHandler creates a new file handler
func NewFileHandler(projectRepo repository.ProjectRepository, k8sService service.KubernetesService) *FileHandler {
	return &FileHandler{
		projectRepo: projectRepo,
		k8sService:  k8sService,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		sidecarPort: 3001,
	}
}

// getSidecarURL resolves the file-browser sidecar URL for a given project
func (h *FileHandler) getSidecarURL(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (string, error) {
	project, err := h.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("failed to find project: %w", err)
	}

	if project.UserID != userID {
		return "", fmt.Errorf("unauthorized: user does not own project")
	}

	podIP, err := h.k8sService.GetPodIP(ctx, project.PodName, project.PodNamespace)
	if err != nil {
		return "", fmt.Errorf("failed to get pod IP: %w", err)
	}

	return fmt.Sprintf("http://%s:%d", podIP, h.sidecarPort), nil
}

func (h *FileHandler) GetTree(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sidecarURL, err := h.getSidecarURL(c.Request.Context(), projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Proxy request to sidecar
	path := c.Query("path")
	url := fmt.Sprintf("%s/files/tree?path=%s", sidecarURL, path)

	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach file-browser sidecar"})
		return
	}
	defer resp.Body.Close()

	// Copy response status and body
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// GetContent proxies GET /files/content to the sidecar
func (h *FileHandler) GetContent(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	sidecarURL, err := h.getSidecarURL(c.Request.Context(), projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("%s/files/content?path=%s", sidecarURL, path)

	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach file-browser sidecar"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// GetFileInfo proxies GET /files/info to the sidecar
func (h *FileHandler) GetFileInfo(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	sidecarURL, err := h.getSidecarURL(c.Request.Context(), projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("%s/files/info?path=%s", sidecarURL, path)

	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach file-browser sidecar"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// WriteFile proxies POST /files/write to the sidecar
func (h *FileHandler) WriteFile(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var reqBody map[string]interface{}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sidecarURL, err := h.getSidecarURL(c.Request.Context(), projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("%s/files/write", sidecarURL)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal request"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach file-browser sidecar"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// DeleteFile proxies DELETE /files to the sidecar
func (h *FileHandler) DeleteFile(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	sidecarURL, err := h.getSidecarURL(c.Request.Context(), projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("%s/files?path=%s", sidecarURL, path)

	req, err := http.NewRequestWithContext(c.Request.Context(), "DELETE", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach file-browser sidecar"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// CreateDirectory proxies POST /files/mkdir to the sidecar
func (h *FileHandler) CreateDirectory(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var reqBody map[string]interface{}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sidecarURL, err := h.getSidecarURL(c.Request.Context(), projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	url := fmt.Sprintf("%s/files/mkdir", sidecarURL)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal request"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach file-browser sidecar"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// FileChangesStream proxies WebSocket /files/watch to the sidecar
func (h *FileHandler) FileChangesStream(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sidecarURL, err := h.getSidecarURL(c.Request.Context(), projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Upgrade client connection to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO: validate origin in production
		},
	}
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	// Connect to sidecar WebSocket
	sidecarWSURL := "ws" + sidecarURL[4:] + "/files/watch" // Convert http:// to ws://
	sidecarConn, _, err := websocket.DefaultDialer.Dial(sidecarWSURL, nil)
	if err != nil {
		clientConn.WriteJSON(gin.H{"error": "Failed to connect to file-browser sidecar"})
		return
	}
	defer sidecarConn.Close()

	// Bidirectional proxy: client <-> sidecar
	done := make(chan struct{})

	// Sidecar -> Client
	go func() {
		defer close(done)
		for {
			messageType, message, err := sidecarConn.ReadMessage()
			if err != nil {
				return
			}
			if err := clientConn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	}()

	// Client -> Sidecar
	go func() {
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			if err := sidecarConn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	}()

	<-done
}
