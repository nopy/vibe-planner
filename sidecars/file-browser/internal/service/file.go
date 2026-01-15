package service

import (
	"os"
	"path/filepath"
	"strings"
)

type FileService struct {
	WorkspaceDir string
}

func NewFileService(workspaceDir string) *FileService {
	return &FileService{
		WorkspaceDir: workspaceDir,
	}
}

func (s *FileService) validatePath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	fullPath := filepath.Join(s.WorkspaceDir, cleanPath)

	if !strings.HasPrefix(fullPath, s.WorkspaceDir) {
		return "", filepath.ErrBadPattern
	}

	return fullPath, nil
}

func (s *FileService) GetTree(path string, maxDepth int) (interface{}, error) {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	return s.buildTree(fullPath, 0, maxDepth)
}

func (s *FileService) buildTree(path string, depth int, maxDepth int) (map[string]interface{}, error) {
	if depth >= maxDepth {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := map[string]interface{}{
		"name":  filepath.Base(path),
		"path":  strings.TrimPrefix(path, s.WorkspaceDir),
		"isDir": info.IsDir(),
		"size":  info.Size(),
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil
		}

		children := make([]interface{}, 0)
		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			if child, err := s.buildTree(childPath, depth+1, maxDepth); err == nil && child != nil {
				children = append(children, child)
			}
		}
		node["children"] = children
	}

	return node, nil
}

func (s *FileService) ReadFile(path string) (string, error) {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (s *FileService) WriteFile(path string, content string) error {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

func (s *FileService) DeleteFile(path string) error {
	fullPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	return os.RemoveAll(fullPath)
}
