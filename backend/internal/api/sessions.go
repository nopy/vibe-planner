package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/npinot/vibe/backend/internal/service"
)

type SessionHandler struct {
	sessionService service.SessionService
}

func NewSessionHandler(sessionService service.SessionService) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
	}
}

type ActiveSessionsResponse struct {
	Sessions []ActiveSessionDTO `json:"sessions"`
}

type ActiveSessionDTO struct {
	ID              string `json:"id"`
	TaskID          string `json:"task_id"`
	Status          string `json:"status"`
	Prompt          string `json:"prompt,omitempty"`
	RemoteSessionID string `json:"remote_session_id,omitempty"`
	LastEventID     string `json:"last_event_id,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type UpdateSessionStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Error  string `json:"error"`
}

func (h *SessionHandler) GetActiveSessions(c *gin.Context) {
	sessions, err := h.sessionService.GetAllActiveSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch active sessions",
		})
		return
	}

	responseSessions := make([]ActiveSessionDTO, 0, len(sessions))
	for _, session := range sessions {
		dto := ActiveSessionDTO{
			ID:              session.ID.String(),
			TaskID:          session.TaskID.String(),
			Status:          string(session.Status),
			Prompt:          session.Prompt,
			RemoteSessionID: session.RemoteSessionID,
			LastEventID:     session.LastEventID,
			CreatedAt:       session.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		responseSessions = append(responseSessions, dto)
	}

	c.JSON(http.StatusOK, ActiveSessionsResponse{
		Sessions: responseSessions,
	})
}

func (h *SessionHandler) UpdateSessionStatus(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid session ID",
		})
		return
	}

	var req UpdateSessionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if err := h.sessionService.UpdateSessionStatus(c.Request.Context(), sessionID, req.Status, req.Error); err != nil {
		if err == service.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update session status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session status updated",
	})
}

type UpdateLastEventIDRequest struct {
	LastEventID string `json:"last_event_id" binding:"required"`
}

func (h *SessionHandler) UpdateLastEventID(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid session ID",
		})
		return
	}

	var req UpdateLastEventIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if err := h.sessionService.UpdateLastEventID(c.Request.Context(), sessionID, req.LastEventID); err != nil {
		if err == service.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update last event ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Last event ID updated",
	})
}
