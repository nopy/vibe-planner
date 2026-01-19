package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestHandler(t *testing.T) (*FileHandler, string, func()) {
	tmpDir, err := os.MkdirTemp("", "handler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	handler := NewFileHandler(tmpDir)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return handler, tmpDir, cleanup
}

func TestNewFileHandler(t *testing.T) {
	handler := NewFileHandler("/test/workspace")
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	if handler.fileService == nil {
		t.Fatal("Expected fileService to be initialized, got nil")
	}
}

func TestGetTree(t *testing.T) {
	handler, tmpDir, cleanup := setupTestHandler(t)
	defer cleanup()

	os.MkdirAll(filepath.Join(tmpDir, "test", "nested"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "test", "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test", "nested", "file2.txt"), []byte("content2"), 0644)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/files/tree", handler.GetTree)

	t.Run("success with default path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/tree", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["name"] == nil {
			t.Error("Expected response to have 'name' field")
		}
	})

	t.Run("success with specific path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/tree?path=test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/tree?path=nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/tree?path=../etc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("filters hidden files by default", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "hidden-filter-test")
		os.MkdirAll(testDir, 0755)
		os.WriteFile(filepath.Join(testDir, "visible.txt"), []byte("visible"), 0644)
		os.WriteFile(filepath.Join(testDir, ".hidden"), []byte("hidden"), 0644)
		os.MkdirAll(filepath.Join(testDir, ".git"), 0755)

		req := httptest.NewRequest("GET", "/files/tree?path=hidden-filter-test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		children := response["children"].([]interface{})
		if len(children) != 1 {
			t.Errorf("Expected 1 visible child, got %d", len(children))
		}
	})

	t.Run("shows hidden files with include_hidden=true", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "hidden-show-test")
		os.MkdirAll(testDir, 0755)
		os.WriteFile(filepath.Join(testDir, "visible.txt"), []byte("visible"), 0644)
		os.WriteFile(filepath.Join(testDir, ".hidden"), []byte("hidden"), 0644)

		req := httptest.NewRequest("GET", "/files/tree?path=hidden-show-test&include_hidden=true", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		children := response["children"].([]interface{})
		if len(children) != 2 {
			t.Errorf("Expected 2 children (visible + hidden), got %d", len(children))
		}
	})

	t.Run("blocks sensitive files even with include_hidden=true", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "sensitive-block-test")
		os.MkdirAll(testDir, 0755)
		os.WriteFile(filepath.Join(testDir, ".env"), []byte("SECRET=value"), 0644)
		os.WriteFile(filepath.Join(testDir, "credentials.json"), []byte("{}"), 0644)
		os.WriteFile(filepath.Join(testDir, "safe.txt"), []byte("safe"), 0644)

		req := httptest.NewRequest("GET", "/files/tree?path=sensitive-block-test&include_hidden=true", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		children := response["children"].([]interface{})
		if len(children) != 1 {
			t.Errorf("Expected only 1 child (safe.txt), got %d", len(children))
		}

		if len(children) > 0 {
			child := children[0].(map[string]interface{})
			if child["name"] != "safe.txt" {
				t.Errorf("Expected safe.txt, got %s", child["name"])
			}
		}
	})

	t.Run("include_hidden=false equivalent to default", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "explicit-false-test")
		os.MkdirAll(testDir, 0755)
		os.WriteFile(filepath.Join(testDir, "visible.txt"), []byte("visible"), 0644)
		os.WriteFile(filepath.Join(testDir, ".hidden"), []byte("hidden"), 0644)

		req := httptest.NewRequest("GET", "/files/tree?path=explicit-false-test&include_hidden=false", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		children := response["children"].([]interface{})
		if len(children) != 1 {
			t.Errorf("Expected 1 child (hidden filtered), got %d", len(children))
		}
	})
}

func TestGetContent(t *testing.T) {
	handler, tmpDir, cleanup := setupTestHandler(t)
	defer cleanup()

	testContent := "test file content"
	testPath := "test/file.txt"
	fullPath := filepath.Join(tmpDir, testPath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, []byte(testContent), 0644)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/files/content", handler.GetContent)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/content?path="+testPath, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["content"] != testContent {
			t.Errorf("Expected content %q, got %q", testContent, response["content"])
		}
	})

	t.Run("missing path parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/content", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/content?path=nonexistent.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("path is directory", func(t *testing.T) {
		os.MkdirAll(filepath.Join(tmpDir, "dirtest"), 0755)
		req := httptest.NewRequest("GET", "/files/content?path=dirtest", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("file too large", func(t *testing.T) {
		largePath := "large.txt"
		largeContent := strings.Repeat("a", 10*1024*1024+1)
		os.WriteFile(filepath.Join(tmpDir, largePath), []byte(largeContent), 0644)

		req := httptest.NewRequest("GET", "/files/content?path="+largePath, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", w.Code)
		}
	})
}

func TestGetFileInfo(t *testing.T) {
	handler, tmpDir, cleanup := setupTestHandler(t)
	defer cleanup()

	testPath := "info-test.txt"
	os.WriteFile(filepath.Join(tmpDir, testPath), []byte("info content"), 0644)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/files/info", handler.GetFileInfo)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/info?path="+testPath, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["name"] != testPath {
			t.Errorf("Expected name %q, got %q", testPath, response["name"])
		}
	})

	t.Run("missing path parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/info", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/info?path=nonexistent.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestWriteFile(t *testing.T) {
	handler, tmpDir, cleanup := setupTestHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/files/write", handler.WriteFile)

	t.Run("success", func(t *testing.T) {
		payload := map[string]string{
			"path":    "write-test.txt",
			"content": "written content",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/write", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		content, err := os.ReadFile(filepath.Join(tmpDir, "write-test.txt"))
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		if string(content) != "written content" {
			t.Errorf("Expected content %q, got %q", "written content", string(content))
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		payload := map[string]string{
			"path":    "nested/path/file.txt",
			"content": "nested content",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/write", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if _, err := os.Stat(filepath.Join(tmpDir, "nested/path/file.txt")); err != nil {
			t.Errorf("File was not created: %v", err)
		}
	})

	t.Run("missing path", func(t *testing.T) {
		payload := map[string]string{
			"content": "content without path",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/write", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/files/write", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("content too large", func(t *testing.T) {
		payload := map[string]string{
			"path":    "large.txt",
			"content": strings.Repeat("x", 10*1024*1024+1),
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/write", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", w.Code)
		}
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		payload := map[string]string{
			"path":    "../etc/passwd",
			"content": "malicious",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/write", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})
}

func TestDeleteFile(t *testing.T) {
	handler, tmpDir, cleanup := setupTestHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/files", handler.DeleteFile)

	t.Run("success delete file", func(t *testing.T) {
		testPath := "delete-test.txt"
		os.WriteFile(filepath.Join(tmpDir, testPath), []byte("delete me"), 0644)

		req := httptest.NewRequest("DELETE", "/files?path="+testPath, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if _, err := os.Stat(filepath.Join(tmpDir, testPath)); !os.IsNotExist(err) {
			t.Error("File still exists after deletion")
		}
	})

	t.Run("success delete directory", func(t *testing.T) {
		dirPath := "delete-dir"
		os.MkdirAll(filepath.Join(tmpDir, dirPath, "subdir"), 0755)
		os.WriteFile(filepath.Join(tmpDir, dirPath, "file.txt"), []byte("content"), 0644)

		req := httptest.NewRequest("DELETE", "/files?path="+dirPath, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if _, err := os.Stat(filepath.Join(tmpDir, dirPath)); !os.IsNotExist(err) {
			t.Error("Directory still exists after deletion")
		}
	})

	t.Run("missing path parameter", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/files", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/files?path=nonexistent.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/files?path=../etc/passwd", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})
}

func TestCreateDirectory(t *testing.T) {
	handler, tmpDir, cleanup := setupTestHandler(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/files/mkdir", handler.CreateDirectory)

	t.Run("success", func(t *testing.T) {
		payload := map[string]string{
			"path": "new/nested/directory",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/mkdir", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		info, err := os.Stat(filepath.Join(tmpDir, "new/nested/directory"))
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Created path is not a directory")
		}
	})

	t.Run("missing path", func(t *testing.T) {
		payload := map[string]string{}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/mkdir", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/files/mkdir", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid path traversal", func(t *testing.T) {
		payload := map[string]string{
			"path": "../etc/malicious",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/files/mkdir", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})
}
