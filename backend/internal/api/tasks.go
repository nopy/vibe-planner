package api

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/npinot/vibe/backend/internal/middleware"
	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/service"
)

// TaskHandler handles task-related requests
type TaskHandler struct {
	taskService     service.TaskService
	taskBroadcaster *TaskBroadcaster
}

// TaskEvent represents a task update event for WebSocket streaming
type TaskEvent struct {
	Type    string      `json:"type"` // "created", "updated", "moved", "deleted"
	Task    *model.Task `json:"task,omitempty"`
	TaskID  string      `json:"task_id,omitempty"`
	Version int64       `json:"version"` // Monotonic counter for ordering
}

// TaskBroadcaster manages WebSocket connections and broadcasts task events
type TaskBroadcaster struct {
	mu             sync.RWMutex
	projectClients map[uuid.UUID]map[*websocket.Conn]bool
	version        int64
}

func NewTaskHandler(taskService service.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService:     taskService,
		taskBroadcaster: NewTaskBroadcaster(),
	}
}

// NewTaskBroadcaster creates a new task broadcaster
func NewTaskBroadcaster() *TaskBroadcaster {
	return &TaskBroadcaster{
		projectClients: make(map[uuid.UUID]map[*websocket.Conn]bool),
		version:        0,
	}
}

// Register adds a WebSocket connection for a project
func (tb *TaskBroadcaster) Register(projectID uuid.UUID, conn *websocket.Conn) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.projectClients[projectID] == nil {
		tb.projectClients[projectID] = make(map[*websocket.Conn]bool)
	}
	tb.projectClients[projectID][conn] = true
	log.Printf("[TaskBroadcaster] Registered connection for project %s (total: %d)", projectID, len(tb.projectClients[projectID]))
}

// Unregister removes a WebSocket connection
func (tb *TaskBroadcaster) Unregister(projectID uuid.UUID, conn *websocket.Conn) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if clients, ok := tb.projectClients[projectID]; ok {
		delete(clients, conn)
		log.Printf("[TaskBroadcaster] Unregistered connection for project %s (remaining: %d)", projectID, len(clients))
		if len(clients) == 0 {
			delete(tb.projectClients, projectID)
		}
	}
	conn.Close()
}

// Broadcast sends a task event to all connected clients for a project
func (tb *TaskBroadcaster) Broadcast(projectID uuid.UUID, event TaskEvent) {
	tb.mu.Lock()
	tb.version++
	event.Version = tb.version
	tb.mu.Unlock()

	tb.mu.RLock()
	clients := tb.projectClients[projectID]
	tb.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	log.Printf("[TaskBroadcaster] Broadcasting %s event (version %d) to %d clients for project %s", event.Type, event.Version, len(clients), projectID)

	for conn := range clients {
		if err := conn.WriteJSON(event); err != nil {
			log.Printf("[TaskBroadcaster] Failed to send to client: %v", err)
			go tb.Unregister(projectID, conn)
		}
	}
}

type CreateTaskRequest struct {
	Title       string             `json:"title" binding:"required"`
	Description string             `json:"description"`
	Priority    model.TaskPriority `json:"priority"`
}

type UpdateTaskRequest struct {
	Title    *string             `json:"title"`
	Priority *model.TaskPriority `json:"priority"`
}

type MoveTaskRequest struct {
	Status   model.TaskStatus `json:"status" binding:"required"`
	Position int              `json:"position"`
}

// ListTasks returns all tasks for a project
func (h *TaskHandler) ListTasks(c *gin.Context) {
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

	tasks, err := h.taskService.ListProjectTasks(c.Request.Context(), projectID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		}
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(c *gin.Context) {
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

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Default priority if not specified
	priority := req.Priority
	if priority == "" {
		priority = model.TaskPriorityMedium
	}

	task, err := h.taskService.CreateTask(
		c.Request.Context(),
		projectID,
		user.ID,
		req.Title,
		req.Description,
		priority,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		case errors.Is(err, service.ErrInvalidTaskTitle):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidTaskPriority):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		}
		return
	}

	h.taskBroadcaster.Broadcast(projectID, TaskEvent{
		Type: "created",
		Task: task,
	})

	c.JSON(http.StatusCreated, task)
}

// GetTask returns a specific task
func (h *TaskHandler) GetTask(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	taskParam := c.Param("taskId")
	taskID, err := uuid.Parse(taskParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task"})
		}
		return
	}

	c.JSON(http.StatusOK, task)
}

// UpdateTask updates task fields
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	taskParam := c.Param("taskId")
	taskID, err := uuid.Parse(taskParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Build updates map from pointer fields
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	task, err := h.taskService.UpdateTask(c.Request.Context(), taskID, user.ID, updates)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		case errors.Is(err, service.ErrInvalidTaskTitle):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidTaskPriority):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		}
		return
	}

	h.taskBroadcaster.Broadcast(task.ProjectID, TaskEvent{
		Type: "updated",
		Task: task,
	})

	c.JSON(http.StatusOK, task)
}

// MoveTask moves a task to a new state and/or position
func (h *TaskHandler) MoveTask(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	taskParam := c.Param("taskId")
	taskID, err := uuid.Parse(taskParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req MoveTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	task, err := h.taskService.MoveTask(c.Request.Context(), taskID, user.ID, req.Status, req.Position)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		case errors.Is(err, service.ErrInvalidStateTransition):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move task"})
		}
		return
	}

	h.taskBroadcaster.Broadcast(task.ProjectID, TaskEvent{
		Type: "moved",
		Task: task,
	})

	c.JSON(http.StatusOK, task)
}

// DeleteTask soft deletes a task
func (h *TaskHandler) DeleteTask(c *gin.Context) {
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

	taskParam := c.Param("taskId")
	taskID, err := uuid.Parse(taskParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	err = h.taskService.DeleteTask(c.Request.Context(), taskID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		}
		return
	}

	h.taskBroadcaster.Broadcast(projectID, TaskEvent{
		Type:   "deleted",
		TaskID: taskID.String(),
	})

	c.Status(http.StatusNoContent)
}

// ExecuteTask executes a task with OpenCode (Phase 5 - not implemented yet)
func (h *TaskHandler) ExecuteTask(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Task execution coming in Phase 5"})
}

var taskUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// TaskUpdatesStream handles WebSocket connections for real-time task updates
func (h *TaskHandler) TaskUpdatesStream(c *gin.Context) {
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

	tasks, err := h.taskService.ListProjectTasks(c.Request.Context(), projectID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProjectNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		case errors.Is(err, service.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		}
		return
	}

	conn, err := taskUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[TaskUpdatesStream] Failed to upgrade connection: %v", err)
		return
	}

	h.taskBroadcaster.Register(projectID, conn)
	defer h.taskBroadcaster.Unregister(projectID, conn)

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	if err := conn.WriteJSON(gin.H{
		"type":    "snapshot",
		"tasks":   tasks,
		"version": h.taskBroadcaster.version,
	}); err != nil {
		log.Printf("[TaskUpdatesStream] Failed to send initial snapshot: %v", err)
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("[TaskUpdatesStream] Unexpected close: %v", err)
				}
				break
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
