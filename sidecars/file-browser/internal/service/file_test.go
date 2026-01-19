package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFileService(t *testing.T) {
	workspaceDir := "/test/workspace"
	service := NewFileService(workspaceDir)

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.WorkspaceDir != workspaceDir {
		t.Errorf("Expected WorkspaceDir to be %s, got %s", workspaceDir, service.WorkspaceDir)
	}
}

func TestValidatePath(t *testing.T) {
	service := NewFileService("/workspace")

	tests := []struct {
		name        string
		path        string
		expectError error
		description string
	}{
		{
			name:        "valid path",
			path:        "test/file.txt",
			expectError: nil,
			description: "Should accept normal path",
		},
		{
			name:        "empty path",
			path:        "",
			expectError: ErrPathRequired,
			description: "Should reject empty path",
		},
		{
			name:        "parent directory traversal",
			path:        "../etc/passwd",
			expectError: ErrInvalidPath,
			description: "Should reject parent directory traversal",
		},
		{
			name:        "nested parent traversal",
			path:        "test/../../etc/passwd",
			expectError: ErrInvalidPath,
			description: "Should reject nested traversal",
		},
		{
			name:        "absolute path outside workspace",
			path:        "/etc/passwd",
			expectError: nil,
			description: "Should allow absolute paths that resolve inside workspace after join",
		},
		{
			name:        "dot slash path",
			path:        "./test/file.txt",
			expectError: nil,
			description: "Should clean and accept dot slash paths",
		},
		{
			name:        "root path",
			path:        "/",
			expectError: nil,
			description: "Should accept root path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath, err := service.validatePath(tt.path)

			if tt.expectError != nil {
				if !errors.Is(err, tt.expectError) {
					t.Errorf("Expected error %v, got %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if !strings.HasPrefix(fullPath, service.WorkspaceDir) {
					t.Errorf("Expected path to be within workspace %s, got %s", service.WorkspaceDir, fullPath)
				}
			}
		})
	}
}

func TestFileServiceWithTempDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewFileService(tmpDir)

	t.Run("CreateDirectory", func(t *testing.T) {
		err := service.CreateDirectory("test/nested/dir")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		fullPath := filepath.Join(tmpDir, "test/nested/dir")
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Created path is not a directory")
		}
	})

	t.Run("WriteFile", func(t *testing.T) {
		content := "Hello, World!"
		err := service.WriteFile("test/hello.txt", content)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		fullPath := filepath.Join(tmpDir, "test/hello.txt")
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		if string(data) != content {
			t.Errorf("Expected content %q, got %q", content, string(data))
		}
	})

	t.Run("WriteFile creates parent directories", func(t *testing.T) {
		err := service.WriteFile("auto/parent/file.txt", "test")
		if err != nil {
			t.Fatalf("Failed to write file with auto-created parents: %v", err)
		}

		fullPath := filepath.Join(tmpDir, "auto/parent/file.txt")
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("File was not created: %v", err)
		}
	})

	t.Run("ReadFile", func(t *testing.T) {
		testContent := "Test content for reading"
		testPath := "test/read-test.txt"

		err := service.WriteFile(testPath, testContent)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		content, err := service.ReadFile(testPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if content != testContent {
			t.Errorf("Expected content %q, got %q", testContent, content)
		}
	})

	t.Run("ReadFile on non-existent file", func(t *testing.T) {
		_, err := service.ReadFile("nonexistent/file.txt")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ReadFile on directory", func(t *testing.T) {
		err := service.CreateDirectory("dir-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		_, err = service.ReadFile("dir-test")
		if !errors.Is(err, ErrNotDirectory) {
			t.Errorf("Expected ErrNotDirectory, got %v", err)
		}
	})

	t.Run("ReadFile size limit", func(t *testing.T) {
		largeContent := strings.Repeat("a", MaxFileSize+1)
		largePath := "large-file.txt"

		fullPath := filepath.Join(tmpDir, largePath)
		err := os.WriteFile(fullPath, []byte(largeContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		_, err = service.ReadFile(largePath)
		if !errors.Is(err, ErrFileTooLarge) {
			t.Errorf("Expected ErrFileTooLarge, got %v", err)
		}
	})

	t.Run("WriteFile size limit", func(t *testing.T) {
		largeContent := strings.Repeat("b", MaxFileSize+1)
		err := service.WriteFile("large-write.txt", largeContent)
		if !errors.Is(err, ErrFileTooLarge) {
			t.Errorf("Expected ErrFileTooLarge, got %v", err)
		}
	})

	t.Run("GetFileInfo", func(t *testing.T) {
		testPath := "test/info-test.txt"
		err := service.WriteFile(testPath, "file info test")
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		info, err := service.GetFileInfo(testPath)
		if err != nil {
			t.Fatalf("Failed to get file info: %v", err)
		}

		if info.Name != "info-test.txt" {
			t.Errorf("Expected name 'info-test.txt', got %q", info.Name)
		}

		if info.IsDirectory {
			t.Error("Expected IsDirectory to be false")
		}

		if info.Size != 14 {
			t.Errorf("Expected size 14, got %d", info.Size)
		}

		if info.ModifiedAt.IsZero() {
			t.Error("Expected ModifiedAt to be set")
		}
	})

	t.Run("GetFileInfo on directory", func(t *testing.T) {
		dirPath := "test/info-dir"
		err := service.CreateDirectory(dirPath)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		info, err := service.GetFileInfo(dirPath)
		if err != nil {
			t.Fatalf("Failed to get directory info: %v", err)
		}

		if !info.IsDirectory {
			t.Error("Expected IsDirectory to be true")
		}
	})

	t.Run("GetFileInfo on non-existent path", func(t *testing.T) {
		_, err := service.GetFileInfo("nonexistent/path")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		testPath := "test/delete-test.txt"
		err := service.WriteFile(testPath, "delete me")
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		err = service.DeleteFile(testPath)
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		fullPath := filepath.Join(tmpDir, testPath)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Error("File still exists after deletion")
		}
	})

	t.Run("DeleteFile on non-existent file", func(t *testing.T) {
		err := service.DeleteFile("nonexistent-delete.txt")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("DeleteFile directory", func(t *testing.T) {
		dirPath := "test/delete-dir"
		filePath := "test/delete-dir/file.txt"

		err := service.CreateDirectory(dirPath)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = service.WriteFile(filePath, "content")
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		err = service.DeleteFile(dirPath)
		if err != nil {
			t.Fatalf("Failed to delete directory: %v", err)
		}

		fullPath := filepath.Join(tmpDir, dirPath)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Error("Directory still exists after deletion")
		}
	})

	t.Run("GetTree", func(t *testing.T) {
		err := service.CreateDirectory("tree-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = service.WriteFile("tree-test/file1.txt", "content1")
		if err != nil {
			t.Fatalf("Failed to write file1: %v", err)
		}

		err = service.WriteFile("tree-test/file2.txt", "content2")
		if err != nil {
			t.Fatalf("Failed to write file2: %v", err)
		}

		err = service.CreateDirectory("tree-test/subdir")
		if err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		err = service.WriteFile("tree-test/subdir/file3.txt", "content3")
		if err != nil {
			t.Fatalf("Failed to write file3: %v", err)
		}

		tree, err := service.GetTree("tree-test", 3, false)
		if err != nil {
			t.Fatalf("Failed to get tree: %v", err)
		}

		if tree.Name != "tree-test" {
			t.Errorf("Expected root name 'tree-test', got %q", tree.Name)
		}

		if !tree.IsDirectory {
			t.Error("Expected root to be directory")
		}

		if len(tree.Children) < 3 {
			t.Errorf("Expected at least 3 children, got %d", len(tree.Children))
		}
	})

	t.Run("GetTree max depth zero", func(t *testing.T) {
		_, err := service.GetTree("/", 0, false)
		if !errors.Is(err, ErrMaxDepthZero) {
			t.Errorf("Expected ErrMaxDepthZero, got %v", err)
		}
	})

	t.Run("GetTree on non-existent path", func(t *testing.T) {
		_, err := service.GetTree("nonexistent-tree", 5, false)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("GetTree filters hidden files by default", func(t *testing.T) {
		err := service.CreateDirectory("hidden-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = service.WriteFile("hidden-test/visible.txt", "visible")
		if err != nil {
			t.Fatalf("Failed to write visible file: %v", err)
		}

		err = service.WriteFile("hidden-test/.hidden", "hidden")
		if err != nil {
			t.Fatalf("Failed to write hidden file: %v", err)
		}

		err = service.CreateDirectory("hidden-test/.git")
		if err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		tree, err := service.GetTree("hidden-test", 3, false)
		if err != nil {
			t.Fatalf("Failed to get tree: %v", err)
		}

		if len(tree.Children) != 1 {
			t.Errorf("Expected 1 visible child, got %d", len(tree.Children))
		}

		if len(tree.Children) > 0 && tree.Children[0].Name != "visible.txt" {
			t.Errorf("Expected visible.txt, got %s", tree.Children[0].Name)
		}
	})

	t.Run("GetTree shows hidden files with includeHidden=true", func(t *testing.T) {
		err := service.CreateDirectory("hidden-show-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = service.WriteFile("hidden-show-test/visible.txt", "visible")
		if err != nil {
			t.Fatalf("Failed to write visible file: %v", err)
		}

		err = service.WriteFile("hidden-show-test/.hidden", "hidden")
		if err != nil {
			t.Fatalf("Failed to write hidden file: %v", err)
		}

		tree, err := service.GetTree("hidden-show-test", 3, true)
		if err != nil {
			t.Fatalf("Failed to get tree: %v", err)
		}

		if len(tree.Children) != 2 {
			t.Errorf("Expected 2 children (visible + hidden), got %d", len(tree.Children))
		}
	})

	t.Run("GetTree blocks sensitive files even with includeHidden=true", func(t *testing.T) {
		err := service.CreateDirectory("sensitive-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = service.WriteFile("sensitive-test/.env", "SECRET=password")
		if err != nil {
			t.Fatalf("Failed to write .env file: %v", err)
		}

		err = service.WriteFile("sensitive-test/.env.local", "SECRET2=password2")
		if err != nil {
			t.Fatalf("Failed to write .env.local file: %v", err)
		}

		err = service.WriteFile("sensitive-test/credentials.json", `{"key": "value"}`)
		if err != nil {
			t.Fatalf("Failed to write credentials.json: %v", err)
		}

		err = service.WriteFile("sensitive-test/regular.txt", "safe content")
		if err != nil {
			t.Fatalf("Failed to write regular file: %v", err)
		}

		tree, err := service.GetTree("sensitive-test", 3, true)
		if err != nil {
			t.Fatalf("Failed to get tree: %v", err)
		}

		if len(tree.Children) != 1 {
			t.Errorf("Expected only 1 child (regular.txt, all sensitive blocked), got %d", len(tree.Children))
		}

		if len(tree.Children) > 0 && tree.Children[0].Name != "regular.txt" {
			t.Errorf("Expected regular.txt, got %s", tree.Children[0].Name)
		}
	})

	t.Run("GetTree blocks all sensitive file types", func(t *testing.T) {
		err := service.CreateDirectory("all-sensitive-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		sensitiveFiles := []string{
			".env",
			".env.production",
			".env.development",
			"credentials.json",
			"secrets.yaml",
			"secrets.yml",
			"id_rsa",
			"id_rsa.pub",
			".npmrc",
			".pypirc",
			"docker-compose.override.yml",
		}

		for _, filename := range sensitiveFiles {
			err = service.WriteFile("all-sensitive-test/"+filename, "secret")
			if err != nil {
				t.Fatalf("Failed to write %s: %v", filename, err)
			}
		}

		err = service.WriteFile("all-sensitive-test/safe.txt", "safe")
		if err != nil {
			t.Fatalf("Failed to write safe file: %v", err)
		}

		tree, err := service.GetTree("all-sensitive-test", 3, true)
		if err != nil {
			t.Fatalf("Failed to get tree: %v", err)
		}

		if len(tree.Children) != 1 {
			t.Errorf("Expected only safe.txt to be visible, got %d children", len(tree.Children))
		}
	})

	t.Run("GetTree handles dotfiles in subdirectories", func(t *testing.T) {
		err := service.CreateDirectory("nested-hidden-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = service.CreateDirectory("nested-hidden-test/subdir")
		if err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		err = service.WriteFile("nested-hidden-test/subdir/.hidden", "hidden")
		if err != nil {
			t.Fatalf("Failed to write nested hidden file: %v", err)
		}

		err = service.WriteFile("nested-hidden-test/subdir/visible.txt", "visible")
		if err != nil {
			t.Fatalf("Failed to write nested visible file: %v", err)
		}

		tree, err := service.GetTree("nested-hidden-test", 3, false)
		if err != nil {
			t.Fatalf("Failed to get tree: %v", err)
		}

		if len(tree.Children) != 1 || tree.Children[0].Name != "subdir" {
			t.Errorf("Expected subdir child")
		}

		if len(tree.Children[0].Children) != 1 {
			t.Errorf("Expected 1 visible file in subdir, got %d", len(tree.Children[0].Children))
		}
	})

	t.Run("GetTree handles mixed hidden and visible subdirectories", func(t *testing.T) {
		err := service.CreateDirectory("mixed-test")
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = service.CreateDirectory("mixed-test/visible-dir")
		if err != nil {
			t.Fatalf("Failed to create visible-dir: %v", err)
		}

		err = service.CreateDirectory("mixed-test/.hidden-dir")
		if err != nil {
			t.Fatalf("Failed to create .hidden-dir: %v", err)
		}

		tree, err := service.GetTree("mixed-test", 3, false)
		if err != nil {
			t.Fatalf("Failed to get tree: %v", err)
		}

		if len(tree.Children) != 1 {
			t.Errorf("Expected 1 visible directory, got %d", len(tree.Children))
		}

		if tree.Children[0].Name != "visible-dir" {
			t.Errorf("Expected visible-dir, got %s", tree.Children[0].Name)
		}
	})
}
