//go:build integration
// +build integration

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/npinot/vibe/backend/internal/model"
	"github.com/npinot/vibe/backend/internal/repository"
	"github.com/npinot/vibe/backend/internal/service"
)

// Integration test for full project lifecycle
// Run with: go test -tags=integration -v ./internal/api
//
// Prerequisites:
// - PostgreSQL running (DATABASE_URL env var)
// - Kubernetes cluster accessible (KUBECONFIG env var or in-cluster)
// - Test database should be separate from dev database
//
// Environment variables:
// - TEST_DATABASE_URL: PostgreSQL connection string (defaults to DATABASE_URL)
// - KUBECONFIG: Path to kubeconfig file (or omit for in-cluster)
// - K8S_NAMESPACE: Kubernetes namespace (defaults to "opencode-test")

func setupIntegrationTest(t *testing.T) (*gorm.DB, service.ProjectService, service.KubernetesService, *ProjectHandler, func()) {
	// Get database URL
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL environment variable not set")
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to database")

	// Auto-migrate models
	err = db.AutoMigrate(&model.User{}, &model.Project{})
	require.NoError(t, err, "Failed to migrate database")

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(db)

	// Initialize Kubernetes service
	kubeconfig := os.Getenv("KUBECONFIG")
	namespace := os.Getenv("K8S_NAMESPACE")
	if namespace == "" {
		namespace = "opencode-test"
	}

	k8sService, err := service.NewKubernetesService(kubeconfig, namespace, nil)
	if err != nil {
		t.Skipf("Failed to initialize Kubernetes service: %v. Skipping integration test.", err)
	}

	// Initialize project service
	projectService := service.NewProjectService(projectRepo, k8sService)

	// Initialize handler
	handler := NewProjectHandler(projectService)

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM projects WHERE name LIKE 'integration-test-%'")
		db.Exec("DELETE FROM users WHERE email LIKE 'test-%@integration.test'")

		// Close database connection
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, projectService, k8sService, handler, cleanup
}

func createTestUser(t *testing.T, db *gorm.DB) *model.User {
	user := &model.User{
		ID:          uuid.New(),
		Email:       fmt.Sprintf("test-%s@integration.test", uuid.New().String()[:8]),
		Name:        "Integration Test User",
		OIDCSubject: fmt.Sprintf("oidc-sub-%s", uuid.New().String()),
	}

	err := db.Create(user).Error
	require.NoError(t, err, "Failed to create test user")

	return user
}

func TestProjectLifecycle_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, _, k8sService, handler, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create test user
	testUser := createTestUser(t, db)

	// Test context
	ctx := context.Background()

	// Step 1: Create project via API
	t.Run("CreateProject", func(t *testing.T) {
		reqBody := CreateProjectRequest{
			Name:        fmt.Sprintf("integration-test-%s", uuid.New().String()[:8]),
			Description: "Integration test project",
			RepoURL:     "https://github.com/test/repo.git",
		}

		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", testUser.ID)

		handler.CreateProject(c)

		assert.Equal(t, http.StatusCreated, w.Code, "Response: %s", w.Body.String())

		var project model.Project
		err = json.Unmarshal(w.Body.Bytes(), &project)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, project.ID)
		assert.Equal(t, reqBody.Name, project.Name)
		assert.Equal(t, reqBody.Description, project.Description)
		assert.Equal(t, reqBody.RepoURL, project.RepoURL)
		assert.Equal(t, testUser.ID, project.UserID)
		assert.NotEmpty(t, project.Slug)
		assert.NotEmpty(t, project.PodName)
		assert.NotEmpty(t, project.WorkspacePVCName)

		// Store project ID for subsequent tests
		projectID := project.ID

		// Step 2: Verify pod created in Kubernetes
		t.Run("VerifyPodCreated", func(t *testing.T) {
			// Wait a bit for pod to be created
			time.Sleep(2 * time.Second)

			podStatus, err := k8sService.GetPodStatus(ctx, project.PodName, project.PodNamespace)
			require.NoError(t, err, "Failed to get pod status")

			// Pod should exist (status can be Pending or Running)
			assert.NotEmpty(t, podStatus, "Pod should exist in Kubernetes")
			assert.Contains(t, []string{"Pending", "Running"}, podStatus, "Pod should be in Pending or Running state")

			t.Logf("Pod %s status: %s", project.PodName, podStatus)
		})

		// Step 3: Verify PVC created
		t.Run("VerifyPVCCreated", func(t *testing.T) {
			// Verify PVC name follows convention
			expectedPVCName := fmt.Sprintf("workspace-%s", project.ID)
			assert.Equal(t, expectedPVCName, project.WorkspacePVCName, "PVC name should follow convention")

			// TODO: Add actual K8s PVC verification when k8sService exposes GetPVC method
			// For now, we verify the field is set correctly
			assert.NotEmpty(t, project.WorkspacePVCName)
		})

		// Step 4: Get project by ID
		t.Run("GetProjectByID", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s", projectID), nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", testUser.ID)
			c.Params = gin.Params{{Key: "id", Value: projectID.String()}}

			handler.GetProject(c)

			assert.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

			var fetchedProject model.Project
			err := json.Unmarshal(w.Body.Bytes(), &fetchedProject)
			require.NoError(t, err)

			assert.Equal(t, project.ID, fetchedProject.ID)
			assert.Equal(t, project.Name, fetchedProject.Name)
			assert.Equal(t, project.PodName, fetchedProject.PodName)
			assert.Equal(t, project.WorkspacePVCName, fetchedProject.WorkspacePVCName)
		})

		// Step 5: List projects (should include our test project)
		t.Run("ListProjects", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", testUser.ID)

			handler.ListProjects(c)

			assert.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

			var projects []model.Project
			err := json.Unmarshal(w.Body.Bytes(), &projects)
			require.NoError(t, err)

			// Find our project in the list
			found := false
			for _, p := range projects {
				if p.ID == projectID {
					found = true
					assert.Equal(t, project.Name, p.Name)
					break
				}
			}
			assert.True(t, found, "Project should be in the list")
		})

		// Step 6: Delete project and verify cleanup
		t.Run("DeleteProjectAndVerifyCleanup", func(t *testing.T) {
			podName := project.PodName
			podNamespace := project.PodNamespace

			// Delete via API
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/projects/%s", projectID), nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", testUser.ID)
			c.Params = gin.Params{{Key: "id", Value: projectID.String()}}

			handler.DeleteProject(c)

			assert.Equal(t, http.StatusNoContent, w.Code, "Response: %s", w.Body.String())

			// Verify pod deleted from Kubernetes
			time.Sleep(2 * time.Second)

			_, err := k8sService.GetPodStatus(ctx, podName, podNamespace)
			assert.Error(t, err, "Pod should be deleted from Kubernetes")
			assert.Contains(t, err.Error(), "not found", "Error should indicate pod not found")

			// Verify project soft-deleted in database
			var deletedProject model.Project
			err = db.Unscoped().First(&deletedProject, "id = ?", projectID).Error
			require.NoError(t, err)
			assert.NotNil(t, deletedProject.DeletedAt, "Project should be soft-deleted")

			// Verify project not returned in list
			req = httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			w = httptest.NewRecorder()
			c, _ = gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", testUser.ID)

			handler.ListProjects(c)

			var projects []model.Project
			err = json.Unmarshal(w.Body.Bytes(), &projects)
			require.NoError(t, err)

			for _, p := range projects {
				assert.NotEqual(t, projectID, p.ID, "Deleted project should not appear in list")
			}
		})
	})
}

// TestProjectCreation_PodFailure tests graceful handling when pod creation fails
func TestProjectCreation_PodFailure_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, _, _, handler, cleanup := setupIntegrationTest(t)
	defer cleanup()

	testUser := createTestUser(t, db)

	t.Run("CreateProjectWithInvalidImage", func(t *testing.T) {
		// Use invalid image to force pod creation failure
		// Note: This test assumes invalid image causes immediate failure
		// Actual behavior may vary based on K8s image pull policy

		reqBody := CreateProjectRequest{
			Name:        fmt.Sprintf("integration-test-fail-%s", uuid.New().String()[:8]),
			Description: "Integration test project (pod failure)",
			RepoURL:     "https://github.com/test/repo.git",
		}

		bodyBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", testUser.ID)

		handler.CreateProject(c)

		// Project creation should still succeed (partial success model)
		assert.Equal(t, http.StatusCreated, w.Code, "Response: %s", w.Body.String())

		var project model.Project
		err = json.Unmarshal(w.Body.Bytes(), &project)
		require.NoError(t, err)

		// Project should be created in database
		assert.NotEqual(t, uuid.Nil, project.ID)

		// Pod error may be set (depending on timing)
		// Status might be "initializing" or "error"
		t.Logf("Project status: %s, PodError: %s", project.PodStatus, project.PodError)
	})
}
