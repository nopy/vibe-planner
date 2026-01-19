package api

import (
	"context"
	"errors"
	"fmt"
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

type InteractionHandler struct {
	interactionService service.InteractionService
	upgrader           websocket.Upgrader
	sessions           *sessionManager
}

func NewInteractionHandler(interactionService service.InteractionService) *InteractionHandler {
	return &InteractionHandler{
		interactionService: interactionService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
		},
		sessions: &sessionManager{
			connections: make(map[uuid.UUID]map[*websocket.Conn]bool),
		},
	}
}

type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type sessionManager struct {
	mu          sync.RWMutex
	connections map[uuid.UUID]map[*websocket.Conn]bool
}

func (sm *sessionManager) register(taskID uuid.UUID, conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.connections[taskID] == nil {
		sm.connections[taskID] = make(map[*websocket.Conn]bool)
	}
	sm.connections[taskID][conn] = true
	log.Printf("[InteractionHandler] Registered connection for task %s (total: %d)", taskID, len(sm.connections[taskID]))
}

func (sm *sessionManager) unregister(taskID uuid.UUID, conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if conns, exists := sm.connections[taskID]; exists {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(sm.connections, taskID)
		}
		log.Printf("[InteractionHandler] Unregistered connection for task %s (remaining: %d)", taskID, len(conns))
	}
}

func (sm *sessionManager) broadcast(taskID uuid.UUID, msg WebSocketMessage) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	conns, exists := sm.connections[taskID]
	if !exists {
		return
	}

	for conn := range conns {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[InteractionHandler] Failed to broadcast to connection: %v", err)
		}
	}
}

func (h *InteractionHandler) TaskInteractionWebSocket(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	taskIDParam := c.Param("id")
	taskID, err := uuid.Parse(taskIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.interactionService.ValidateTaskOwnership(ctx, taskID, user.ID); err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case errors.Is(err, service.ErrTaskNotOwnedByUser):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate task ownership"})
		}
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[InteractionHandler] Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	h.sessions.register(taskID, conn)
	defer h.sessions.unregister(taskID, conn)

	conn.SetReadDeadline(time.Now().Add(300 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(300 * time.Second))
		return nil
	})

	history, err := h.interactionService.GetTaskHistory(ctx, taskID, user.ID)
	if err != nil {
		log.Printf("[InteractionHandler] Failed to load history: %v", err)
	} else {
		historyMessages := make([]WebSocketMessage, len(history))
		for i, interaction := range history {
			historyMessages[i] = WebSocketMessage{
				Type:      interaction.MessageType,
				Content:   interaction.Content,
				Metadata:  interaction.Metadata,
				Timestamp: interaction.CreatedAt,
			}
		}
		if err := conn.WriteJSON(gin.H{
			"type":     "history",
			"messages": historyMessages,
		}); err != nil {
			log.Printf("[InteractionHandler] Failed to send history: %v", err)
		}
	}

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[InteractionHandler] WebSocket error: %v", err)
			}
			break
		}

		conn.SetReadDeadline(time.Now().Add(300 * time.Second))

		switch msg.Type {
		case "user_message":
			if err := h.handleUserMessage(ctx, taskID, user.ID, msg); err != nil {
				h.sendError(conn, fmt.Sprintf("Failed to process message: %v", err))
				continue
			}

		default:
			h.sendError(conn, fmt.Sprintf("Unknown message type: %s", msg.Type))
		}
	}
}

func (h *InteractionHandler) handleUserMessage(ctx context.Context, taskID, userID uuid.UUID, msg WebSocketMessage) error {
	metadataJSON := model.JSONB(msg.Metadata)

	interaction, err := h.interactionService.CreateUserMessage(ctx, taskID, userID, msg.Content, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to store user message: %w", err)
	}

	broadcastMsg := WebSocketMessage{
		Type:      interaction.MessageType,
		Content:   interaction.Content,
		Metadata:  interaction.Metadata,
		Timestamp: interaction.CreatedAt,
	}
	h.sessions.broadcast(taskID, broadcastMsg)

	return nil
}

func (h *InteractionHandler) sendError(conn *websocket.Conn, message string) {
	errMsg := WebSocketMessage{
		Type:      "error",
		Content:   message,
		Timestamp: time.Now(),
	}
	if err := conn.WriteJSON(errMsg); err != nil {
		log.Printf("[InteractionHandler] Failed to send error message: %v", err)
	}
}

func (h *InteractionHandler) GetTaskHistory(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	taskIDParam := c.Param("id")
	taskID, err := uuid.Parse(taskIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	ctx := c.Request.Context()
	history, err := h.interactionService.GetTaskHistory(ctx, taskID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case errors.Is(err, service.ErrTaskNotOwnedByUser):
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch interaction history"})
		}
		return
	}

	type InteractionResponse struct {
		ID          uuid.UUID              `json:"id"`
		TaskID      uuid.UUID              `json:"task_id"`
		SessionID   *uuid.UUID             `json:"session_id"`
		UserID      uuid.UUID              `json:"user_id"`
		MessageType string                 `json:"message_type"`
		Content     string                 `json:"content"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
		CreatedAt   time.Time              `json:"created_at"`
	}

	responses := make([]InteractionResponse, len(history))
	for i, interaction := range history {
		responses[i] = InteractionResponse{
			ID:          interaction.ID,
			TaskID:      interaction.TaskID,
			SessionID:   interaction.SessionID,
			UserID:      interaction.UserID,
			MessageType: interaction.MessageType,
			Content:     interaction.Content,
			Metadata:    interaction.Metadata,
			CreatedAt:   interaction.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"interactions": responses})
}

func (h *InteractionHandler) BroadcastAgentResponse(taskID uuid.UUID, content string, metadata model.JSONB) {
	msg := WebSocketMessage{
		Type:      "agent_response",
		Content:   content,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
	h.sessions.broadcast(taskID, msg)
}

func (h *InteractionHandler) BroadcastSystemNotification(taskID uuid.UUID, content string, metadata model.JSONB) {
	msg := WebSocketMessage{
		Type:      "system_notification",
		Content:   content,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
	h.sessions.broadcast(taskID, msg)
}
