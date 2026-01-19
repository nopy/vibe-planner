package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/npinot/vibe/sidecars/file-browser/internal/service"
)

var watchUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, validate origin against allowed domains
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WatchHandler handles WebSocket connections for real-time file change notifications
type WatchHandler struct {
	watcher *service.FileWatcher
}

// NewWatchHandler creates a new watch handler with a file watcher
func NewWatchHandler(watcher *service.FileWatcher) *WatchHandler {
	return &WatchHandler{
		watcher: watcher,
	}
}

// FileChangesStream handles WebSocket connections for file watching
func (h *WatchHandler) FileChangesStream(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := watchUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection to WebSocket", "error", err)
		return
	}

	// Register client with file watcher
	h.watcher.Register(conn)
	defer h.watcher.Unregister(conn)

	// Configure WebSocket read deadline and pong handler
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send initial snapshot with current version
	if err := conn.WriteJSON(gin.H{
		"type":    "connected",
		"version": h.watcher.GetVersion(),
		"message": "File watcher connected",
	}); err != nil {
		slog.Error("Failed to send initial snapshot", "error", err)
		return
	}

	slog.Info("WebSocket client connected for file watching")

	// Create ticker for periodic pings (30s interval)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Create channel to signal completion
	done := make(chan struct{})

	// Start read pump goroutine to detect client disconnect
	go func() {
		defer close(done)
		for {
			// Read messages from client (we don't expect any, but this detects disconnects)
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					slog.Warn("WebSocket unexpected close", "error", err)
				} else {
					slog.Debug("WebSocket client disconnected", "error", err)
				}
				break
			}
		}
	}()

	// Main event loop: send periodic pings
	for {
		select {
		case <-done:
			// Client disconnected
			slog.Info("WebSocket client disconnected")
			return
		case <-ticker.C:
			// Send ping to keep connection alive
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Warn("Failed to send ping, closing connection", "error", err)
				return
			}
		}
	}
}
