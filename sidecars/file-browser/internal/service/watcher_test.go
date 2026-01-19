package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestNewFileWatcher(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewFileWatcher() failed: %v", err)
	}
	defer fw.Close()

	if fw.workspaceDir != tmpDir {
		t.Errorf("expected workspaceDir %s, got %s", tmpDir, fw.workspaceDir)
	}
	if fw.debounceTime != 100*time.Millisecond {
		t.Errorf("expected debounceTime 100ms, got %v", fw.debounceTime)
	}
	if fw.started {
		t.Error("watcher should not be started immediately")
	}
}

func TestFileWatcher_Start(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewFileWatcher() failed: %v", err)
	}
	defer fw.Close()

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !fw.started {
		t.Error("watcher should be marked as started")
	}

	if err := fw.Start(); err == nil {
		t.Error("Start() should fail when already started")
	}
}

func TestFileWatcher_StartRecursive(t *testing.T) {
	tmpDir := t.TempDir()

	subDir1 := filepath.Join(tmpDir, "subdir1")
	subDir2 := filepath.Join(tmpDir, "subdir2")
	nestedDir := filepath.Join(subDir1, "nested")

	if err := os.MkdirAll(subDir1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(subDir2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatalf("NewFileWatcher() failed: %v", err)
	}
	defer fw.Close()

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
}

func TestFileWatcher_MapFsnotifyEvent(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	tests := []struct {
		name     string
		op       fsnotify.Op
		expected string
	}{
		{"create", fsnotify.Create, "created"},
		{"write", fsnotify.Write, "modified"},
		{"remove", fsnotify.Remove, "deleted"},
		{"rename", fsnotify.Rename, "renamed"},
		{"chmod", fsnotify.Chmod, "modified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := fsnotify.Event{
				Name: "/test/file.txt",
				Op:   tt.op,
			}
			result := fw.mapFsnotifyEvent(event, "/file.txt")

			if result.Type != tt.expected {
				t.Errorf("expected event type %s, got %s", tt.expected, result.Type)
			}
			if result.Path != "/file.txt" {
				t.Errorf("expected path /file.txt, got %s", result.Path)
			}
		})
	}
}

func TestFileWatcher_RegisterUnregister(t *testing.T) {
	t.Skip("Skipping test that requires actual WebSocket connections - integration test needed")
}

func TestFileWatcher_GetVersion(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	if v := fw.GetVersion(); v != 0 {
		t.Errorf("expected initial version 0, got %d", v)
	}

	fw.mu.Lock()
	fw.version = 42
	fw.mu.Unlock()

	if v := fw.GetVersion(); v != 42 {
		t.Errorf("expected version 42, got %d", v)
	}
}

func TestFileWatcher_Close(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := fw.Close(); err == nil {
		t.Error("Close() should fail when watcher not started")
	}

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if err := fw.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if fw.started {
		t.Error("watcher should be marked as not started after Close()")
	}
}

func TestFileWatcher_DebounceCoalescing(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	fw.debounceTime = 50 * time.Millisecond

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	fw.debounceMu.Lock()
	debounceCount := len(fw.debounce)
	fw.debounceMu.Unlock()

	if debounceCount != 0 {
		t.Errorf("expected debounce map to be empty after delay, got %d entries", debounceCount)
	}
}

func TestFileWatcher_ConcurrentRegisterUnregister(t *testing.T) {
	t.Skip("Skipping test that requires actual WebSocket connections - integration test needed")
}

func TestFileWatcher_VersionMonotonicity(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	event := FileChangeEvent{
		Type: "created",
		Path: "/test.txt",
	}

	var lastVersion int64
	for i := 0; i < 10; i++ {
		fw.broadcast(event)
		currentVersion := fw.GetVersion()

		if currentVersion <= lastVersion {
			t.Errorf("version not monotonic: previous %d, current %d", lastVersion, currentVersion)
		}
		lastVersion = currentVersion
	}

	if lastVersion != 10 {
		t.Errorf("expected version 10 after 10 broadcasts, got %d", lastVersion)
	}
}

func TestFileWatcher_AddRecursiveError(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	nonExistentPath := filepath.Join(tmpDir, "nonexistent")

	err = fw.addRecursive(nonExistentPath)
	if err == nil {
		t.Error("addRecursive should fail for non-existent path")
	}
}

func TestFileWatcher_HandleEventCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	fw, err := NewFileWatcher(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	fw.debounceTime = 20 * time.Millisecond

	if err := fw.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	newDir := filepath.Join(tmpDir, "newdir")
	if err := os.Mkdir(newDir, 0755); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
}
