package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalidPath  = errors.New("invalid path: directory traversal detected")
	ErrPathRequired = errors.New("path is required")
	ErrNotFound     = errors.New("file or directory not found")
	ErrNotDirectory = errors.New("path is not a directory")
	ErrMaxDepthZero = errors.New("max depth must be greater than zero")
	ErrFileTooLarge = errors.New("file exceeds maximum size limit")
)

const MaxFileSize = 10 * 1024 * 1024

type FileInfo struct {
	Path        string      `json:"path"`
	Name        string      `json:"name"`
	IsDirectory bool        `json:"is_directory"`
	Size        int64       `json:"size"`
	ModifiedAt  time.Time   `json:"modified_at"`
	Children    []*FileInfo `json:"children,omitempty"`
}

type FileService struct {
	WorkspaceDir string
}

func NewFileService(workspaceDir string) *FileService {
	return &FileService{
		WorkspaceDir: workspaceDir,
	}
}

func (s *FileService) validatePath(path string) (string, error) {
	if path == "" {
		return "", ErrPathRequired
	}

	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, "..") {
		return "", ErrInvalidPath
	}

	fullPath := filepath.Join(s.WorkspaceDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.WorkspaceDir) {
		return "", ErrInvalidPath
	}

	return fullPath, nil
}

func (s *FileService) GetTree(path string, maxDepth int) (*FileInfo, error) {
	if maxDepth <= 0 {
		return nil, ErrMaxDepthZero
	}

	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	return s.buildTree(fullPath, 0, maxDepth)
}

func (s *FileService) buildTree(path string, depth int, maxDepth int) (*FileInfo, error) {
	if depth >= maxDepth {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	relativePath := strings.TrimPrefix(path, s.WorkspaceDir)
	if relativePath == "" {
		relativePath = "/"
	}

	node := &FileInfo{
		Name:        filepath.Base(path),
		Path:        relativePath,
		IsDirectory: info.IsDir(),
		Size:        info.Size(),
		ModifiedAt:  info.ModTime(),
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil
		}

		children := make([]*FileInfo, 0, len(entries))
		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			if child, err := s.buildTree(childPath, depth+1, maxDepth); err == nil && child != nil {
				children = append(children, child)
			}
		}
		node.Children = children
	}

	return node, nil
}

func (s *FileService) GetFileInfo(path string) (*FileInfo, error) {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	relativePath := strings.TrimPrefix(fullPath, s.WorkspaceDir)
	if relativePath == "" {
		relativePath = "/"
	}

	return &FileInfo{
		Name:        info.Name(),
		Path:        relativePath,
		IsDirectory: info.IsDir(),
		Size:        info.Size(),
		ModifiedAt:  info.ModTime(),
	}, nil
}

func (s *FileService) ReadFile(path string) (string, error) {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return "", ErrNotDirectory
	}

	if info.Size() > MaxFileSize {
		return "", ErrFileTooLarge
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

func (s *FileService) WriteFile(path string, content string) error {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	if len(content) > MaxFileSize {
		return ErrFileTooLarge
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *FileService) DeleteFile(path string) error {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to delete path: %w", err)
	}

	return nil
}

func (s *FileService) CreateDirectory(path string) error {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}
