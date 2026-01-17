package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/service"
)

type MockProjectService struct {
	mock.Mock
}

func (m *MockProjectService) CreateProject(ctx context.Context, userID uuid.UUID, name, description, repoURL string) (*model.Project, error) {
	args := m.Called(ctx, userID, name, description, repoURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) GetProject(ctx context.Context, id, userID uuid.UUID) (*model.Project, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) ListProjects(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Project), args.Error(1)
}

func (m *MockProjectService) UpdateProject(ctx context.Context, id, userID uuid.UUID, updates map[string]interface{}) (*model.Project, error) {
	args := m.Called(ctx, id, userID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) DeleteProject(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func setupProjectTestRouter(handler *ProjectHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		testUser := &model.User{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Email:       "test@example.com",
			Name:        "Test User",
			OIDCSubject: "test-subject",
		}
		c.Set("currentUser", testUser)
		c.Next()
	})

	return router
}

func TestProjectHandler_ListProjects(t *testing.T) {
	mockService := new(MockProjectService)
	handler := NewProjectHandler(mockService)
	router := setupProjectTestRouter(handler)

	router.GET("/projects", handler.ListProjects)

	t.Run("successful list retrieval", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projects := []model.Project{
			{
				ID:     uuid.New(),
				UserID: userID,
				Name:   "Project 1",
				Slug:   "project-1",
				Status: model.ProjectStatusReady,
			},
			{
				ID:     uuid.New(),
				UserID: userID,
				Name:   "Project 2",
				Slug:   "project-2",
				Status: model.ProjectStatusInitializing,
			},
		}

		mockService.On("ListProjects", mock.Anything, userID).Return(projects, nil).Once()

		req, _ := http.NewRequest("GET", "/projects", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []model.Project
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "Project 1", response[0].Name)
		assert.Equal(t, "Project 2", response[1].Name)

		mockService.AssertExpectations(t)
	})

	t.Run("empty project list", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		mockService.On("ListProjects", mock.Anything, userID).Return([]model.Project{}, nil).Once()

		req, _ := http.NewRequest("GET", "/projects", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []model.Project
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 0)

		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		mockService.On("ListProjects", mock.Anything, userID).Return(nil, errors.New("database error")).Once()

		req, _ := http.NewRequest("GET", "/projects", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Failed to fetch projects")

		mockService.AssertExpectations(t)
	})
}

func TestProjectHandler_CreateProject(t *testing.T) {
	mockService := new(MockProjectService)
	handler := NewProjectHandler(mockService)
	router := setupProjectTestRouter(handler)

	router.POST("/projects", handler.CreateProject)

	t.Run("successful project creation", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		reqBody := CreateProjectRequest{
			Name:        "Test Project",
			Description: "Test Description",
			RepoURL:     "https://github.com/test/repo",
		}

		expectedProject := &model.Project{
			ID:          projectID,
			UserID:      userID,
			Name:        "Test Project",
			Slug:        "test-project",
			Description: "Test Description",
			RepoURL:     "https://github.com/test/repo",
			Status:      model.ProjectStatusReady,
		}

		mockService.On("CreateProject", mock.Anything, userID, "Test Project", "Test Description", "https://github.com/test/repo").
			Return(expectedProject, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.Project
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Test Project", response.Name)
		assert.Equal(t, "test-project", response.Slug)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required field", func(t *testing.T) {
		reqBody := map[string]string{
			"description": "Missing name field",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid project name", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		reqBody := CreateProjectRequest{
			Name: "Invalid@Name#",
		}

		mockService.On("CreateProject", mock.Anything, userID, "Invalid@Name#", "", "").
			Return(nil, service.ErrInvalidProjectName).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid repo URL", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

		reqBody := CreateProjectRequest{
			Name:    "Valid Name",
			RepoURL: "invalid-url",
		}

		mockService.On("CreateProject", mock.Anything, userID, "Valid Name", "", "invalid-url").
			Return(nil, service.ErrInvalidRepoURL).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/projects", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestProjectHandler_GetProject(t *testing.T) {
	mockService := new(MockProjectService)
	handler := NewProjectHandler(mockService)
	router := setupProjectTestRouter(handler)

	router.GET("/projects/:id", handler.GetProject)

	t.Run("successful project retrieval", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		expectedProject := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Test Project",
			Slug:   "test-project",
			Status: model.ProjectStatusReady,
		}

		mockService.On("GetProject", mock.Anything, projectID, userID).Return(expectedProject, nil).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.Project
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Test Project", response.Name)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/projects/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("project not found", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		mockService.On("GetProject", mock.Anything, projectID, userID).Return(nil, service.ErrProjectNotFound).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		mockService.On("GetProject", mock.Anything, projectID, userID).Return(nil, service.ErrUnauthorized).Once()

		req, _ := http.NewRequest("GET", "/projects/"+projectID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestProjectHandler_UpdateProject(t *testing.T) {
	mockService := new(MockProjectService)
	handler := NewProjectHandler(mockService)
	router := setupProjectTestRouter(handler)

	router.PATCH("/projects/:id", handler.UpdateProject)

	t.Run("successful update", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		newName := "Updated Name"
		reqBody := UpdateProjectRequest{
			Name: &newName,
		}

		expectedProject := &model.Project{
			ID:     projectID,
			UserID: userID,
			Name:   "Updated Name",
			Slug:   "updated-name",
			Status: model.ProjectStatusReady,
		}

		mockService.On("UpdateProject", mock.Anything, projectID, userID, map[string]interface{}{"name": "Updated Name"}).
			Return(expectedProject, nil).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.Project
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", response.Name)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		newName := "Updated Name"
		reqBody := UpdateProjectRequest{
			Name: &newName,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/invalid-uuid", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no fields to update", func(t *testing.T) {
		projectID := uuid.New()
		reqBody := UpdateProjectRequest{}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("project not found", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		newName := "Updated Name"
		reqBody := UpdateProjectRequest{
			Name: &newName,
		}

		mockService.On("UpdateProject", mock.Anything, projectID, userID, map[string]interface{}{"name": "Updated Name"}).
			Return(nil, service.ErrProjectNotFound).Once()

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", "/projects/"+projectID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockService.AssertExpectations(t)
	})
}

func TestProjectHandler_DeleteProject(t *testing.T) {
	mockService := new(MockProjectService)
	handler := NewProjectHandler(mockService)
	router := setupProjectTestRouter(handler)

	router.DELETE("/projects/:id", handler.DeleteProject)

	t.Run("successful deletion", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		mockService.On("DeleteProject", mock.Anything, projectID, userID).Return(nil).Once()

		req, _ := http.NewRequest("DELETE", "/projects/"+projectID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid project ID", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/projects/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("project not found", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		mockService.On("DeleteProject", mock.Anything, projectID, userID).Return(service.ErrProjectNotFound).Once()

		req, _ := http.NewRequest("DELETE", "/projects/"+projectID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized access", func(t *testing.T) {
		userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		projectID := uuid.New()

		mockService.On("DeleteProject", mock.Anything, projectID, userID).Return(service.ErrUnauthorized).Once()

		req, _ := http.NewRequest("DELETE", "/projects/"+projectID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		mockService.AssertExpectations(t)
	})
}
