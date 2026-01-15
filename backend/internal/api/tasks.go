package api

import "github.com/gin-gonic/gin"

// TaskHandler handles task-related requests
type TaskHandler struct {
	// Add dependencies here (task service, etc.)
}

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{}
}

// ListTasks returns all tasks for a project
func (h *TaskHandler) ListTasks(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// GetTask returns a specific task
func (h *TaskHandler) GetTask(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// UpdateTask updates a task
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// DeleteTask deletes a task
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// ExecuteTask executes a task with OpenCode
func (h *TaskHandler) ExecuteTask(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}
