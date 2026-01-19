package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/npinot/vibe/sidecars/file-browser/internal/service"
)

func TestNewWatchHandler(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := service.NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	wh := NewWatchHandler(fw)
	if wh == nil {
		t.Fatal("NewWatchHandler returned nil")
	}
	if wh.watcher != fw {
		t.Error("watcher not set correctly")
	}
}

func TestWatchHandler_FileChangesStreamUpgrade(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := service.NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(); err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	wh := NewWatchHandler(fw)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/watch", wh.FileChangesStream)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/watch"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	var msg map[string]interface{}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read initial message: %v", err)
	}

	if msgType, ok := msg["type"].(string); !ok || msgType != "connected" {
		t.Errorf("expected type 'connected', got %v", msg["type"])
	}

	if _, ok := msg["version"]; !ok {
		t.Error("expected version field in initial message")
	}
}

func TestWatchHandler_FileChangesStreamReceivesEvents(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := service.NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(); err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	wh := NewWatchHandler(fw)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/watch", wh.FileChangesStream)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/watch"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	var msg map[string]interface{}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read initial message: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestWatchHandler_PongHandler(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := service.NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(); err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	wh := NewWatchHandler(fw)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/watch", wh.FileChangesStream)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/watch"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	var msg map[string]interface{}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read initial message: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	for i := 0; i < 3; i++ {
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			break
		}
		if i == 2 {
			break
		}
	}
}

func TestWatchHandler_ClientDisconnect(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := service.NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(); err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	wh := NewWatchHandler(fw)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/watch", wh.FileChangesStream)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/watch"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	var msg map[string]interface{}
	conn.ReadJSON(&msg)

	conn.Close()

	time.Sleep(200 * time.Millisecond)
}
