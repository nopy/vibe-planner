package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
)

// MockProjectRepository is a mock implementation of ProjectRepository
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Project), args.Error(1)
}

func (m *MockProjectRepository) Update(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProjectRepository) UpdatePodStatus(ctx context.Context, id uuid.UUID, status string, podError string) error {
	args := m.Called(ctx, id, status, podError)
	return args.Error(0)
}

var _ repository.ProjectRepository = (*MockProjectRepository)(nil)

// MockKubernetesService is a mock implementation of KubernetesService
type MockKubernetesService struct {
	mock.Mock
}

func (m *MockKubernetesService) CreateProjectPod(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	// Simulate pod creation by setting pod metadata
	if args.Error(0) == nil {
		project.PodName = "project-12345678"
		project.PodNamespace = "opencode"
		project.WorkspacePVCName = "workspace-12345678"
		project.PodStatus = "Pending"
	}
	return args.Error(0)
}

func (m *MockKubernetesService) DeleteProjectPod(ctx context.Context, podName, namespace string) error {
	args := m.Called(ctx, podName, namespace)
	return args.Error(0)
}

func (m *MockKubernetesService) GetPodStatus(ctx context.Context, podName, namespace string) (string, error) {
	args := m.Called(ctx, podName, namespace)
	return args.String(0), args.Error(1)
}

func (m *MockKubernetesService) WatchPodStatus(ctx context.Context, podName, namespace string) (<-chan string, error) {
	args := m.Called(ctx, podName, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan string), args.Error(1)
}

var _ KubernetesService = (*MockKubernetesService)(nil)

func TestProjectService_CreateProject(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("successful project creation", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Project")).Return(nil)
		mockK8s.On("CreateProjectPod", ctx, mock.AnythingOfType("*model.Project")).Return(nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Project")).Return(nil)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.CreateProject(ctx, userID, "Test Project", "A test project", "https://github.com/test/repo")

		assert.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, "Test Project", project.Name)
		assert.Equal(t, "test-project", project.Slug)
		assert.Equal(t, "A test project", project.Description)
		assert.Equal(t, "https://github.com/test/repo", project.RepoURL)
		assert.Equal(t, userID, project.UserID)
		assert.Equal(t, model.ProjectStatusReady, project.Status)

		mockRepo.AssertExpectations(t)
		mockK8s.AssertExpectations(t)
	})

	t.Run("invalid project name - empty", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.CreateProject(ctx, userID, "", "Description", "")

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrInvalidProjectName)

		mockRepo.AssertNotCalled(t, "Create")
		mockK8s.AssertNotCalled(t, "CreateProjectPod")
	})

	t.Run("invalid project name - too long", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		svc := NewProjectService(mockRepo, mockK8s)

		longName := ""
		for i := 0; i < 101; i++ {
			longName += "a"
		}

		project, err := svc.CreateProject(ctx, userID, longName, "Description", "")

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrInvalidProjectName)

		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("invalid project name - special characters", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.CreateProject(ctx, userID, "Test@Project#123", "Description", "")

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrInvalidProjectName)

		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("invalid repo URL", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.CreateProject(ctx, userID, "Test Project", "Description", "invalid-url")

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrInvalidRepoURL)

		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("database creation failure", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		dbErr := errors.New("database error")
		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Project")).Return(dbErr)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.CreateProject(ctx, userID, "Test Project", "Description", "")

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "failed to create project in database")

		mockRepo.AssertExpectations(t)
		mockK8s.AssertNotCalled(t, "CreateProjectPod")
	})

	t.Run("pod creation failure - error stored in project", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		podErr := errors.New("kubernetes error")
		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Project")).Return(nil)
		mockK8s.On("CreateProjectPod", ctx, mock.AnythingOfType("*model.Project")).Return(podErr)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Project")).Return(nil)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.CreateProject(ctx, userID, "Test Project", "Description", "")

		assert.NoError(t, err) // Project creation succeeds even if pod fails
		assert.NotNil(t, project)
		assert.Equal(t, model.ProjectStatusError, project.Status)
		assert.Contains(t, project.PodError, "Pod creation failed")

		mockRepo.AssertExpectations(t)
		mockK8s.AssertExpectations(t)
	})

	t.Run("pod created but update fails", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		updateErr := errors.New("update error")
		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Project")).Return(nil)
		mockK8s.On("CreateProjectPod", ctx, mock.AnythingOfType("*model.Project")).Return(nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Project")).Return(updateErr)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.CreateProject(ctx, userID, "Test Project", "Description", "")

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "pod created successfully but failed to update project metadata")

		mockRepo.AssertExpectations(t)
		mockK8s.AssertExpectations(t)
	})
}

func TestProjectService_GetProject(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successful project retrieval", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		expectedProject := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(expectedProject, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.GetProject(ctx, projectID, userID)

		assert.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, projectID, project.ID)
		assert.Equal(t, userID, project.UserID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("project not found", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		mockRepo.On("FindByID", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.GetProject(ctx, projectID, userID)

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrProjectNotFound)

		mockRepo.AssertExpectations(t)
	})

	t.Run("unauthorized access - different user", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		otherUserID := uuid.New()
		expectedProject := &model.Project{
			ID:     projectID,
			UserID: otherUserID,
			Name:   "Test Project",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(expectedProject, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.GetProject(ctx, projectID, userID)

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrUnauthorized)

		mockRepo.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		dbErr := errors.New("database error")
		mockRepo.On("FindByID", ctx, projectID).Return(nil, dbErr)

		svc := NewProjectService(mockRepo, mockK8s)

		project, err := svc.GetProject(ctx, projectID, userID)

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "failed to retrieve project")

		mockRepo.AssertExpectations(t)
	})
}

func TestProjectService_ListProjects(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("successful list retrieval", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		expectedProjects := []model.Project{
			{ID: uuid.New(), UserID: userID, Name: "Project 1"},
			{ID: uuid.New(), UserID: userID, Name: "Project 2"},
			{ID: uuid.New(), UserID: userID, Name: "Project 3"},
		}

		mockRepo.On("FindByUserID", ctx, userID).Return(expectedProjects, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		projects, err := svc.ListProjects(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, projects)
		assert.Len(t, projects, 3)

		mockRepo.AssertExpectations(t)
	})

	t.Run("empty project list", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		emptyProjects := []model.Project{}
		mockRepo.On("FindByUserID", ctx, userID).Return(emptyProjects, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		projects, err := svc.ListProjects(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, projects)
		assert.Len(t, projects, 0)

		mockRepo.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		dbErr := errors.New("database error")
		mockRepo.On("FindByUserID", ctx, userID).Return(nil, dbErr)

		svc := NewProjectService(mockRepo, mockK8s)

		projects, err := svc.ListProjects(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, projects)
		assert.Contains(t, err.Error(), "failed to list projects")

		mockRepo.AssertExpectations(t)
	})
}

func TestProjectService_UpdateProject(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successful update - name", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Old Name",
			Slug:   "old-name",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Project")).Return(nil)

		svc := NewProjectService(mockRepo, mockK8s)

		updates := map[string]interface{}{
			"name": "New Name",
		}

		project, err := svc.UpdateProject(ctx, projectID, userID, updates)

		assert.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, "New Name", project.Name)
		assert.Equal(t, "new-name", project.Slug)

		mockRepo.AssertExpectations(t)
	})

	t.Run("successful update - description", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:          projectID,
			UserID:      userID,
			Name:        "Test Project",
			Description: "Old description",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Project")).Return(nil)

		svc := NewProjectService(mockRepo, mockK8s)

		updates := map[string]interface{}{
			"description": "New description",
		}

		project, err := svc.UpdateProject(ctx, projectID, userID, updates)

		assert.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, "New description", project.Description)

		mockRepo.AssertExpectations(t)
	})

	t.Run("successful update - repo URL", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:      projectID,
			UserID:  userID,
			Name:    "Test Project",
			RepoURL: "https://github.com/old/repo",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.Project")).Return(nil)

		svc := NewProjectService(mockRepo, mockK8s)

		updates := map[string]interface{}{
			"repo_url": "https://github.com/new/repo",
		}

		project, err := svc.UpdateProject(ctx, projectID, userID, updates)

		assert.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, "https://github.com/new/repo", project.RepoURL)

		mockRepo.AssertExpectations(t)
	})

	t.Run("project not found", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		mockRepo.On("FindByID", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

		svc := NewProjectService(mockRepo, mockK8s)

		updates := map[string]interface{}{"name": "New Name"}
		project, err := svc.UpdateProject(ctx, projectID, userID, updates)

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrProjectNotFound)

		mockRepo.AssertExpectations(t)
	})

	t.Run("unauthorized update", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		otherUserID := uuid.New()
		existingProject := &model.Project{
			ID:     projectID,
			UserID: otherUserID,
			Name:   "Test Project",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		updates := map[string]interface{}{"name": "New Name"}
		project, err := svc.UpdateProject(ctx, projectID, userID, updates)

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrUnauthorized)

		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Update")
	})

	t.Run("invalid name in update", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		updates := map[string]interface{}{"name": ""}
		project, err := svc.UpdateProject(ctx, projectID, userID, updates)

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrInvalidProjectName)

		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Update")
	})

	t.Run("invalid repo URL in update", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		updates := map[string]interface{}{"repo_url": "invalid-url"}
		project, err := svc.UpdateProject(ctx, projectID, userID, updates)

		assert.Error(t, err)
		assert.Nil(t, project)
		assert.ErrorIs(t, err, ErrInvalidRepoURL)

		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Update")
	})
}

func TestProjectService_DeleteProject(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("successful deletion with pod", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:           projectID,
			UserID:       userID,
			Name:         "Test Project",
			PodName:      "project-12345678",
			PodNamespace: "opencode",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
		mockK8s.On("DeleteProjectPod", ctx, "project-12345678", "opencode").Return(nil)
		mockRepo.On("SoftDelete", ctx, projectID).Return(nil)

		svc := NewProjectService(mockRepo, mockK8s)

		err := svc.DeleteProject(ctx, projectID, userID)

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockK8s.AssertExpectations(t)
	})

	t.Run("successful deletion without pod", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
			// No pod name/namespace
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
		mockRepo.On("SoftDelete", ctx, projectID).Return(nil)

		svc := NewProjectService(mockRepo, mockK8s)

		err := svc.DeleteProject(ctx, projectID, userID)

		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockK8s.AssertNotCalled(t, "DeleteProjectPod")
	})

	t.Run("project not found", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		mockRepo.On("FindByID", ctx, projectID).Return(nil, gorm.ErrRecordNotFound)

		svc := NewProjectService(mockRepo, mockK8s)

		err := svc.DeleteProject(ctx, projectID, userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrProjectNotFound)

		mockRepo.AssertExpectations(t)
		mockK8s.AssertNotCalled(t, "DeleteProjectPod")
		mockRepo.AssertNotCalled(t, "SoftDelete")
	})

	t.Run("unauthorized deletion", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		otherUserID := uuid.New()
		existingProject := &model.Project{
			ID:     projectID,
			UserID: otherUserID,
			Name:   "Test Project",
		}

		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)

		svc := NewProjectService(mockRepo, mockK8s)

		err := svc.DeleteProject(ctx, projectID, userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUnauthorized)

		mockRepo.AssertExpectations(t)
		mockK8s.AssertNotCalled(t, "DeleteProjectPod")
		mockRepo.AssertNotCalled(t, "SoftDelete")
	})

	t.Run("pod deletion failure", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:           projectID,
			UserID:       userID,
			Name:         "Test Project",
			PodName:      "project-12345678",
			PodNamespace: "opencode",
		}

		podErr := errors.New("kubernetes error")
		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
		mockK8s.On("DeleteProjectPod", ctx, "project-12345678", "opencode").Return(podErr)

		svc := NewProjectService(mockRepo, mockK8s)

		err := svc.DeleteProject(ctx, projectID, userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPodDeletionFailed)

		mockRepo.AssertExpectations(t)
		mockK8s.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "SoftDelete")
	})

	t.Run("database deletion failure", func(t *testing.T) {
		mockRepo := new(MockProjectRepository)
		mockK8s := new(MockKubernetesService)

		existingProject := &model.Project{
			ID:           projectID,
			UserID:       userID,
			Name:         "Test Project",
			PodName:      "project-12345678",
			PodNamespace: "opencode",
		}

		dbErr := errors.New("database error")
		mockRepo.On("FindByID", ctx, projectID).Return(existingProject, nil)
		mockK8s.On("DeleteProjectPod", ctx, "project-12345678", "opencode").Return(nil)
		mockRepo.On("SoftDelete", ctx, projectID).Return(dbErr)

		svc := NewProjectService(mockRepo, mockK8s)

		err := svc.DeleteProject(ctx, projectID, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete project from database")

		mockRepo.AssertExpectations(t)
		mockK8s.AssertExpectations(t)
	})
}

func TestValidateProjectName(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid name", "My Project", false},
		{"valid with hyphen", "my-project", false},
		{"valid with underscore", "my_project", false},
		{"valid with number", "project123", false},
		{"empty string", "", true},
		{"too long", string(make([]byte, 101)), true},
		{"special chars", "test@project", true},
		{"slash", "test/project", true},
		{"dot", "test.project", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProjectName(tc.input)
			if tc.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRepoURL(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid https", "https://github.com/user/repo", false},
		{"valid http", "http://github.com/user/repo", false},
		{"valid git", "git@github.com:user/repo.git", false},
		{"empty string", "", false}, // Empty is valid (optional)
		{"invalid prefix", "ftp://example.com", true},
		{"no protocol", "github.com/user/repo", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRepoURL(tc.input)
			if tc.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "My Project", "my-project"},
		{"with hyphen", "my-project", "my-project"},
		{"with underscore", "my_project", "myproject"},
		{"multiple spaces", "my   project", "my-project"},
		{"special chars", "My@Project#123", "myproject123"},
		{"consecutive hyphens", "my---project", "my-project"},
		{"leading hyphen", "-myproject", "myproject"},
		{"trailing hyphen", "myproject-", "myproject"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generateSlug(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
