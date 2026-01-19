package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
	"github.com/npinot/vibe/backend/internal/service"
)

type MockFileProjectRepository struct {
	mock.Mock
}

func (m *MockFileProjectRepository) Create(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockFileProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockFileProjectRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Project, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Project), args.Error(1)
}

func (m *MockFileProjectRepository) Update(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockFileProjectRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFileProjectRepository) UpdatePodStatus(ctx context.Context, id uuid.UUID, status string, podError string) error {
	args := m.Called(ctx, id, status, podError)
	return args.Error(0)
}

var _ repository.ProjectRepository = (*MockFileProjectRepository)(nil)

type MockFileK8sService struct {
	mock.Mock
}

func (m *MockFileK8sService) CreateProjectPod(ctx context.Context, project *model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockFileK8sService) DeleteProjectPod(ctx context.Context, podName, namespace string) error {
	args := m.Called(ctx, podName, namespace)
	return args.Error(0)
}

func (m *MockFileK8sService) GetPodStatus(ctx context.Context, podName, namespace string) (string, error) {
	args := m.Called(ctx, podName, namespace)
	return args.String(0), args.Error(1)
}

func (m *MockFileK8sService) WatchPodStatus(ctx context.Context, podName, namespace string) (<-chan string, error) {
	args := m.Called(ctx, podName, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan string), args.Error(1)
}

func (m *MockFileK8sService) GetPodIP(ctx context.Context, podName, namespace string) (string, error) {
	args := m.Called(ctx, podName, namespace)
	return args.String(0), args.Error(1)
}

var _ service.KubernetesService = (*MockFileK8sService)(nil)

func setupFileTestRouter(handler *FileHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	testUser := &model.User{
		ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email: "test@example.com",
	}

	router.Use(func(c *gin.Context) {
		c.Set("currentUser", testUser)
		c.Next()
	})

	projects := router.Group("/api/projects")
	{
		projects.GET("/:id/files/tree", handler.GetTree)
		projects.GET("/:id/files/content", handler.GetContent)
		projects.GET("/:id/files/info", handler.GetFileInfo)
		projects.POST("/:id/files/write", handler.WriteFile)
		projects.DELETE("/:id/files", handler.DeleteFile)
		projects.POST("/:id/files/mkdir", handler.CreateDirectory)
	}

	return router
}

func TestNewFileHandler(t *testing.T) {
	mockRepo := new(MockFileProjectRepository)
	mockK8s := new(MockFileK8sService)

	handler := NewFileHandler(mockRepo, mockK8s)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.projectRepo)
	assert.NotNil(t, handler.k8sService)
	assert.NotNil(t, handler.httpClient)
}

func TestFileHandler_GetTree(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		queryPath      string
		mockSetup      func(*MockFileProjectRepository, *MockFileK8sService, *httptest.Server)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "success",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)

				addr := server.Listener.Addr().(*net.TCPAddr)
				k8s.On("GetPodIP", mock.Anything, "test-pod", "test-ns").Return(addr.IP.String(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid project ID",
			projectID:      "invalid-uuid",
			queryPath:      "/",
			mockSetup:      func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid project ID",
		},
		{
			name:      "project not found",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "unauthorized user",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000099"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "unauthorized",
		},
		{
			name:      "pod IP not found",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)
				k8s.On("GetPodIP", mock.Anything, "test-pod", "test-ns").Return("", errors.New("pod not running"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to get pod IP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockFileProjectRepository)
			mockK8s := new(MockFileK8sService)

			sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gin.H{"files": []string{"file1.txt"}})
			}))
			defer sidecarServer.Close()

			tt.mockSetup(mockRepo, mockK8s, sidecarServer)

			handler := NewFileHandler(mockRepo, mockK8s)
			addr := sidecarServer.Listener.Addr().(*net.TCPAddr)
			handler.sidecarPort = addr.Port
			router := setupFileTestRouter(handler)

			req := httptest.NewRequest("GET", "/api/projects/"+tt.projectID+"/files/tree?path="+tt.queryPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockRepo.AssertExpectations(t)
			mockK8s.AssertExpectations(t)
		})
	}
}

func TestFileHandler_GetContent(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		queryPath      string
		mockSetup      func(*MockFileProjectRepository, *MockFileK8sService, *httptest.Server)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "success",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/test.txt",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)

				addr := server.Listener.Addr().(*net.TCPAddr)
				k8s.On("GetPodIP", mock.Anything, "test-pod", "test-ns").Return(addr.IP.String(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing path parameter",
			projectID:      "00000000-0000-0000-0000-000000000002",
			queryPath:      "",
			mockSetup:      func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "path parameter is required",
		},
		{
			name:      "unauthorized user",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/test.txt",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000099"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockFileProjectRepository)
			mockK8s := new(MockFileK8sService)

			sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gin.H{"content": "test content"})
			}))
			defer sidecarServer.Close()

			tt.mockSetup(mockRepo, mockK8s, sidecarServer)

			handler := NewFileHandler(mockRepo, mockK8s)
			addr := sidecarServer.Listener.Addr().(*net.TCPAddr)
			handler.sidecarPort = addr.Port
			router := setupFileTestRouter(handler)

			req := httptest.NewRequest("GET", "/api/projects/"+tt.projectID+"/files/content?path="+tt.queryPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockRepo.AssertExpectations(t)
			mockK8s.AssertExpectations(t)
		})
	}
}

func TestFileHandler_GetFileInfo(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		queryPath      string
		mockSetup      func(*MockFileProjectRepository, *MockFileK8sService, *httptest.Server)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "success",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/test.txt",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)

				addr := server.Listener.Addr().(*net.TCPAddr)
				k8s.On("GetPodIP", mock.Anything, "test-pod", "test-ns").Return(addr.IP.String(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing path parameter",
			projectID:      "00000000-0000-0000-0000-000000000002",
			queryPath:      "",
			mockSetup:      func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "path parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockFileProjectRepository)
			mockK8s := new(MockFileK8sService)

			sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gin.H{"name": "test.txt", "size": 123})
			}))
			defer sidecarServer.Close()

			tt.mockSetup(mockRepo, mockK8s, sidecarServer)

			handler := NewFileHandler(mockRepo, mockK8s)
			addr := sidecarServer.Listener.Addr().(*net.TCPAddr)
			handler.sidecarPort = addr.Port
			router := setupFileTestRouter(handler)

			req := httptest.NewRequest("GET", "/api/projects/"+tt.projectID+"/files/info?path="+tt.queryPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockRepo.AssertExpectations(t)
			mockK8s.AssertExpectations(t)
		})
	}
}

func TestFileHandler_WriteFile(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		requestBody    map[string]interface{}
		mockSetup      func(*MockFileProjectRepository, *MockFileK8sService, *httptest.Server)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "success",
			projectID: "00000000-0000-0000-0000-000000000002",
			requestBody: map[string]interface{}{
				"path":    "/test.txt",
				"content": "Hello World",
			},
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)

				addr := server.Listener.Addr().(*net.TCPAddr)
				k8s.On("GetPodIP", mock.Anything, "test-pod", "test-ns").Return(addr.IP.String(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			projectID:      "00000000-0000-0000-0000-000000000002",
			requestBody:    nil,
			mockSetup:      func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "unauthorized user",
			projectID: "00000000-0000-0000-0000-000000000002",
			requestBody: map[string]interface{}{
				"path":    "/test.txt",
				"content": "Hello",
			},
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000099"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockFileProjectRepository)
			mockK8s := new(MockFileK8sService)

			sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gin.H{"message": "file written"})
			}))
			defer sidecarServer.Close()

			tt.mockSetup(mockRepo, mockK8s, sidecarServer)

			handler := NewFileHandler(mockRepo, mockK8s)
			addr := sidecarServer.Listener.Addr().(*net.TCPAddr)
			handler.sidecarPort = addr.Port
			router := setupFileTestRouter(handler)

			var body io.Reader
			if tt.requestBody != nil {
				jsonData, _ := json.Marshal(tt.requestBody)
				body = bytes.NewBuffer(jsonData)
			} else {
				body = bytes.NewBufferString("invalid json")
			}

			req := httptest.NewRequest("POST", "/api/projects/"+tt.projectID+"/files/write", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockRepo.AssertExpectations(t)
			mockK8s.AssertExpectations(t)
		})
	}
}

func TestFileHandler_DeleteFile(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		queryPath      string
		mockSetup      func(*MockFileProjectRepository, *MockFileK8sService, *httptest.Server)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "success",
			projectID: "00000000-0000-0000-0000-000000000002",
			queryPath: "/test.txt",
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)

				addr := server.Listener.Addr().(*net.TCPAddr)
				k8s.On("GetPodIP", mock.Anything, "test-pod", "test-ns").Return(addr.IP.String(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing path parameter",
			projectID:      "00000000-0000-0000-0000-000000000002",
			queryPath:      "",
			mockSetup:      func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "path parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockFileProjectRepository)
			mockK8s := new(MockFileK8sService)

			sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gin.H{"message": "file deleted"})
			}))
			defer sidecarServer.Close()

			tt.mockSetup(mockRepo, mockK8s, sidecarServer)

			handler := NewFileHandler(mockRepo, mockK8s)
			addr := sidecarServer.Listener.Addr().(*net.TCPAddr)
			handler.sidecarPort = addr.Port
			router := setupFileTestRouter(handler)

			req := httptest.NewRequest("DELETE", "/api/projects/"+tt.projectID+"/files?path="+tt.queryPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockRepo.AssertExpectations(t)
			mockK8s.AssertExpectations(t)
		})
	}
}

func TestFileHandler_CreateDirectory(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		requestBody    map[string]interface{}
		mockSetup      func(*MockFileProjectRepository, *MockFileK8sService, *httptest.Server)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "success",
			projectID: "00000000-0000-0000-0000-000000000002",
			requestBody: map[string]interface{}{
				"path": "/new_dir",
			},
			mockSetup: func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {
				project := &model.Project{
					ID:           uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PodName:      "test-pod",
					PodNamespace: "test-ns",
				}
				repo.On("FindByID", mock.Anything, uuid.MustParse("00000000-0000-0000-0000-000000000002")).Return(project, nil)

				addr := server.Listener.Addr().(*net.TCPAddr)
				k8s.On("GetPodIP", mock.Anything, "test-pod", "test-ns").Return(addr.IP.String(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			projectID:      "00000000-0000-0000-0000-000000000002",
			requestBody:    nil,
			mockSetup:      func(repo *MockFileProjectRepository, k8s *MockFileK8sService, server *httptest.Server) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockFileProjectRepository)
			mockK8s := new(MockFileK8sService)

			sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(gin.H{"message": "directory created"})
			}))
			defer sidecarServer.Close()

			tt.mockSetup(mockRepo, mockK8s, sidecarServer)

			handler := NewFileHandler(mockRepo, mockK8s)
			addr := sidecarServer.Listener.Addr().(*net.TCPAddr)
			handler.sidecarPort = addr.Port
			router := setupFileTestRouter(handler)

			var body io.Reader
			if tt.requestBody != nil {
				jsonData, _ := json.Marshal(tt.requestBody)
				body = bytes.NewBuffer(jsonData)
			} else {
				body = bytes.NewBufferString("invalid json")
			}

			req := httptest.NewRequest("POST", "/api/projects/"+tt.projectID+"/files/mkdir", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockRepo.AssertExpectations(t)
			mockK8s.AssertExpectations(t)
		})
	}
}
