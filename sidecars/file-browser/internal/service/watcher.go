package service

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var (
	ErrWatcherClosed = errors.New("watcher is closed")
)

// FileChangeEvent represents a file system change event
type FileChangeEvent struct {
	Type      string    `json:"type"`               // created, modified, deleted, renamed
	Path      string    `json:"path"`               // Relative to workspace root
	OldPath   string    `json:"old_path,omitempty"` // For rename events
	Timestamp time.Time `json:"timestamp"`
}

// FileWatcher watches a directory tree for changes and broadcasts events to WebSocket clients
type FileWatcher struct {
	workspaceDir string
	watcher      *fsnotify.Watcher
	clients      map[*websocket.Conn]bool
	mu           sync.RWMutex
	debounce     map[string]*time.Timer
	debounceMu   sync.Mutex
	debounceTime time.Duration
	done         chan struct{}
	started      bool
	version      int64 // Monotonic counter for event versioning
}

// NewFileWatcher creates a new file watcher for the given workspace directory
func NewFileWatcher(workspaceDir string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw := &FileWatcher{
		workspaceDir: workspaceDir,
		watcher:      watcher,
		clients:      make(map[*websocket.Conn]bool),
		debounce:     make(map[string]*time.Timer),
		debounceTime: 100 * time.Millisecond,
		done:         make(chan struct{}),
		started:      false,
		version:      0,
	}

	return fw, nil
}

// Start begins watching the workspace directory recursively
func (fw *FileWatcher) Start() error {
	fw.mu.Lock()
	if fw.started {
		fw.mu.Unlock()
		return errors.New("watcher already started")
	}
	fw.started = true
	fw.mu.Unlock()

	// Add workspace root to watcher
	if err := fw.addRecursive(fw.workspaceDir); err != nil {
		return err
	}

	// Start event processing loop
	go fw.processEvents()

	slog.Info("FileWatcher started", "workspace", fw.workspaceDir)
	return nil
}

// addRecursive adds a directory and all subdirectories to the watcher
func (fw *FileWatcher) addRecursive(path string) error {
	if err := fw.watcher.Add(path); err != nil {
		return err
	}

	// Note: fsnotify watches directories recursively on some platforms (e.g., Windows)
	// but not on others (e.g., Linux). This implementation explicitly adds each directory
	// for cross-platform consistency.

	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() && walkPath != path {
			if err := fw.watcher.Add(walkPath); err != nil {
				slog.Warn("Failed to add directory to watcher", "path", walkPath, "error", err)
			}
		}
		return nil
	})
}

// processEvents processes file system events in a loop
func (fw *FileWatcher) processEvents() {
	for {
		select {
		case <-fw.done:
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("FileWatcher error", "error", err)
		}
	}
}

// handleEvent processes a single file system event with debouncing
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	relativePath := strings.TrimPrefix(event.Name, fw.workspaceDir)
	if relativePath == "" {
		relativePath = "/"
	}

	fw.debounceMu.Lock()
	defer fw.debounceMu.Unlock()

	// Cancel existing debounce timer for this path
	if timer, exists := fw.debounce[relativePath]; exists {
		timer.Stop()
	}

	// Create debounced event
	fw.debounce[relativePath] = time.AfterFunc(fw.debounceTime, func() {
		fw.debounceMu.Lock()
		delete(fw.debounce, relativePath)
		fw.debounceMu.Unlock()

		changeEvent := fw.mapFsnotifyEvent(event, relativePath)
		fw.broadcast(changeEvent)

		// If a directory was created, add it to the watcher
		if event.Op&fsnotify.Create != 0 {
			if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
				if err := fw.addRecursive(event.Name); err != nil {
					slog.Warn("Failed to watch new directory", "path", event.Name, "error", err)
				}
			}
		}
	})
}

// mapFsnotifyEvent converts fsnotify.Event to FileChangeEvent
func (fw *FileWatcher) mapFsnotifyEvent(event fsnotify.Event, relativePath string) FileChangeEvent {
	var eventType string

	switch {
	case event.Op&fsnotify.Create != 0:
		eventType = "created"
	case event.Op&fsnotify.Write != 0:
		eventType = "modified"
	case event.Op&fsnotify.Remove != 0:
		eventType = "deleted"
	case event.Op&fsnotify.Rename != 0:
		eventType = "renamed"
	case event.Op&fsnotify.Chmod != 0:
		eventType = "modified" // Treat chmod as modification
	default:
		eventType = "modified"
	}

	return FileChangeEvent{
		Type:      eventType,
		Path:      relativePath,
		Timestamp: time.Now(),
	}
}

// broadcast sends an event to all connected WebSocket clients
func (fw *FileWatcher) broadcast(event FileChangeEvent) {
	fw.mu.Lock()
	fw.version++
	fw.mu.Unlock()

	fw.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(fw.clients))
	for conn := range fw.clients {
		clients = append(clients, conn)
	}
	fw.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	// Add version to event
	type versionedEvent struct {
		FileChangeEvent
		Version int64 `json:"version"`
	}
	ve := versionedEvent{
		FileChangeEvent: event,
		Version:         fw.version,
	}

	slog.Debug("Broadcasting file change event", "type", event.Type, "path", event.Path, "clients", len(clients), "version", fw.version)

	// Broadcast to all clients
	for _, conn := range clients {
		if err := conn.WriteJSON(ve); err != nil {
			slog.Warn("Failed to send file change event to client", "error", err)
			go fw.Unregister(conn)
		}
	}
}

// Register adds a WebSocket client to receive file change events
func (fw *FileWatcher) Register(conn *websocket.Conn) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.clients[conn] = true
	slog.Info("Client registered for file watching", "total_clients", len(fw.clients))
}

// Unregister removes a WebSocket client
func (fw *FileWatcher) Unregister(conn *websocket.Conn) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if _, ok := fw.clients[conn]; ok {
		delete(fw.clients, conn)
		if conn != nil {
			conn.Close()
		}
		slog.Info("Client unregistered from file watching", "remaining_clients", len(fw.clients))
	}
}

// GetVersion returns the current event version counter
func (fw *FileWatcher) GetVersion() int64 {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.version
}

// Close stops the file watcher and cleans up resources
func (fw *FileWatcher) Close() error {
	fw.mu.Lock()
	if !fw.started {
		fw.mu.Unlock()
		return errors.New("watcher not started")
	}
	fw.mu.Unlock()

	// Signal shutdown
	close(fw.done)

	// Close all client connections
	fw.mu.Lock()
	for conn := range fw.clients {
		conn.Close()
	}
	fw.clients = make(map[*websocket.Conn]bool)
	fw.mu.Unlock()

	// Close fsnotify watcher
	if err := fw.watcher.Close(); err != nil {
		slog.Error("Failed to close fsnotify watcher", "error", err)
		return err
	}

	// Mark as stopped
	fw.mu.Lock()
	fw.started = false
	fw.mu.Unlock()

	slog.Info("FileWatcher stopped")
	return nil
}
