package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrUnauthorized       = errors.New("unauthorized to access this project")
	ErrInvalidProjectName = errors.New("invalid project name")
	ErrInvalidRepoURL     = errors.New("invalid repository URL")
	ErrPodCreationFailed  = errors.New("failed to create Kubernetes pod")
	ErrPodDeletionFailed  = errors.New("failed to delete Kubernetes pod")
)

// ProjectService defines business logic operations for project management
type ProjectService interface {
	// CreateProject creates a new project with a Kubernetes pod
	CreateProject(ctx context.Context, userID uuid.UUID, name, description, repoURL string) (*model.Project, error)

	// GetProject retrieves a project by ID with authorization check
	GetProject(ctx context.Context, id, userID uuid.UUID) (*model.Project, error)

	// ListProjects retrieves all projects for a user
	ListProjects(ctx context.Context, userID uuid.UUID) ([]model.Project, error)

	// UpdateProject updates project fields with authorization check
	UpdateProject(ctx context.Context, id, userID uuid.UUID, updates map[string]interface{}) (*model.Project, error)

	// DeleteProject deletes a project and its Kubernetes pod
	DeleteProject(ctx context.Context, id, userID uuid.UUID) error
}

type projectService struct {
	projectRepo repository.ProjectRepository
	k8sService  KubernetesService
}

// NewProjectService creates a new project service
func NewProjectService(projectRepo repository.ProjectRepository, k8sService KubernetesService) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		k8sService:  k8sService,
	}
}

// CreateProject creates a new project with validation and Kubernetes pod spawning
func (s *projectService) CreateProject(ctx context.Context, userID uuid.UUID, name, description, repoURL string) (*model.Project, error) {
	// Validate input
	if err := validateProjectName(name); err != nil {
		return nil, err
	}

	if repoURL != "" {
		if err := validateRepoURL(repoURL); err != nil {
			return nil, err
		}
	}

	// Create project entity
	project := &model.Project{
		UserID:      userID,
		Name:        name,
		Slug:        generateSlug(name),
		Description: description,
		RepoURL:     repoURL,
		Status:      model.ProjectStatusInitializing,
	}

	// Save to database first
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project in database: %w", err)
	}

	// Spawn Kubernetes pod
	if err := s.k8sService.CreateProjectPod(ctx, project); err != nil {
		// Store error in project metadata but don't fail the creation
		project.Status = model.ProjectStatusError
		project.PodError = fmt.Sprintf("Pod creation failed: %s", err.Error())

		// Update project with error status
		if updateErr := s.projectRepo.Update(ctx, project); updateErr != nil {
			return nil, fmt.Errorf("pod creation failed and failed to update project status: %w (original error: %v)", updateErr, err)
		}

		return project, nil
	}

	// Update project with pod metadata
	project.Status = model.ProjectStatusReady
	if err := s.projectRepo.Update(ctx, project); err != nil {
		// Pod was created but we failed to update DB - this is a partial failure
		// The project exists in DB but status is still "initializing"
		return nil, fmt.Errorf("pod created successfully but failed to update project metadata: %w", err)
	}

	return project, nil
}

// GetProject retrieves a project with authorization check
func (s *projectService) GetProject(ctx context.Context, id, userID uuid.UUID) (*model.Project, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to retrieve project: %w", err)
	}

	// Authorization check
	if project.UserID != userID {
		return nil, ErrUnauthorized
	}

	return project, nil
}

// ListProjects retrieves all projects for a user
func (s *projectService) ListProjects(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	projects, err := s.projectRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

// UpdateProject updates project fields with authorization check
func (s *projectService) UpdateProject(ctx context.Context, id, userID uuid.UUID, updates map[string]interface{}) (*model.Project, error) {
	// Retrieve and authorize
	project, err := s.GetProject(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Validate updates
	if name, ok := updates["name"].(string); ok {
		if err := validateProjectName(name); err != nil {
			return nil, err
		}
		project.Name = name
		project.Slug = generateSlug(name)
	}

	if description, ok := updates["description"].(string); ok {
		project.Description = description
	}

	if repoURL, ok := updates["repo_url"].(string); ok {
		if repoURL != "" {
			if err := validateRepoURL(repoURL); err != nil {
				return nil, err
			}
		}
		project.RepoURL = repoURL
	}

	// Update in database
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return project, nil
}

// DeleteProject deletes a project and its Kubernetes resources
func (s *projectService) DeleteProject(ctx context.Context, id, userID uuid.UUID) error {
	// Retrieve and authorize
	project, err := s.GetProject(ctx, id, userID)
	if err != nil {
		return err
	}

	// Delete Kubernetes pod if it exists
	if project.PodName != "" && project.PodNamespace != "" {
		if err := s.k8sService.DeleteProjectPod(ctx, project.PodName, project.PodNamespace); err != nil {
			// Log the error but continue with soft delete
			// In production, you might want to queue this for retry
			return fmt.Errorf("%w: %v", ErrPodDeletionFailed, err)
		}
	}

	// Soft delete in database
	if err := s.projectRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete project from database: %w", err)
	}

	return nil
}

// validateProjectName validates project name constraints
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidProjectName)
	}

	if len(name) > 100 {
		return fmt.Errorf("%w: name cannot exceed 100 characters", ErrInvalidProjectName)
	}

	// Check for valid characters (alphanumeric, spaces, hyphens, underscores)
	validName := regexp.MustCompile(`^[a-zA-Z0-9\s\-_]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("%w: name can only contain alphanumeric characters, spaces, hyphens, and underscores", ErrInvalidProjectName)
	}

	return nil
}

// validateRepoURL validates repository URL format
func validateRepoURL(url string) error {
	if url == "" {
		return nil // Empty URL is valid (optional field)
	}

	// Basic URL validation - must start with http://, https://, or git@
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "http://") &&
		!strings.HasPrefix(url, "https://") &&
		!strings.HasPrefix(url, "git@") {
		return fmt.Errorf("%w: must start with http://, https://, or git@", ErrInvalidRepoURL)
	}

	return nil
}

// generateSlug generates a URL-friendly slug from project name
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove invalid characters (keep only alphanumeric and hyphens)
	validSlug := regexp.MustCompile(`[^a-z0-9\-]`)
	slug = validSlug.ReplaceAllString(slug, "")

	// Remove consecutive hyphens
	consecutiveHyphens := regexp.MustCompile(`-+`)
	slug = consecutiveHyphens.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}
