//go:build integration
// +build integration

package api

import (
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

// Integration tests for task execution workflow (Phase 5.7)
// Run with: go test -tags=integration -v ./internal/api
//
// Prerequisites:
// - PostgreSQL running (TEST_DATABASE_URL env var)
// - Kubernetes cluster accessible (KUBECONFIG env var or in-cluster)
// - OpenCode server sidecar image available in cluster
// - Test database should be separate from dev database
//
// Environment variables:
// - TEST_DATABASE_URL: PostgreSQL connection string (defaults to DATABASE_URL)
// - KUBECONFIG: Path to kubeconfig file (or omit for in-cluster)
// - K8S_NAMESPACE: Kubernetes namespace (defaults to "opencode-test")

func setupTaskExecutionIntegrationTest(t *testing.T) (*gorm.DB, *TaskHandler, service.TaskService, service.ProjectService, func()) {
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

	// Auto-migrate all models
	err = db.AutoMigrate(&model.User{}, &model.Project{}, &model.Task{}, &model.Session{})
	require.NoError(t, err, "Failed to migrate database")

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	configRepo := repository.NewConfigRepository(db)

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

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		// Use a test key (base64-encoded 32 bytes) - safe for testing only
		encryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	}
	configService, err := service.NewConfigService(configRepo, encryptionKey)
	require.NoError(t, err, "Failed to initialize config service")

	// Initialize services
	projectService := service.NewProjectService(projectRepo, k8sService)
	sessionService := service.NewSessionService(sessionRepo, taskRepo, projectRepo, k8sService, configService)
	taskService := service.NewTaskService(taskRepo, projectRepo, sessionService)

	// Initialize handlers
	taskHandler := NewTaskHandler(taskService, projectRepo, k8sService)

	// Cleanup function
	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM sessions WHERE task_id IN (SELECT id FROM tasks WHERE title LIKE 'integration-test-%')")
		db.Exec("DELETE FROM tasks WHERE title LIKE 'integration-test-%'")
		db.Exec("DELETE FROM projects WHERE name LIKE 'integration-test-%'")
		db.Exec("DELETE FROM users WHERE email LIKE 'test-%@integration.test'")

		// Close database connection
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, taskHandler, taskService, projectService, cleanup
}

func createTestUserForExecution(t *testing.T, db *gorm.DB) *model.User {
	user := &model.User{
		ID:          uuid.New(),
		Email:       fmt.Sprintf("test-%s@integration.test", uuid.New().String()[:8]),
		Name:        "Integration Test User (Execution)",
		OIDCSubject: fmt.Sprintf("oidc-sub-%s", uuid.New().String()),
	}

	err := db.Create(user).Error
	require.NoError(t, err, "Failed to create test user")

	return user
}

func createTestProject(t *testing.T, db *gorm.DB, projectService service.ProjectService, userID uuid.UUID) *model.Project {
	ctx := context.Background()

	project, err := projectService.CreateProject(ctx, userID,
		fmt.Sprintf("integration-test-%s", uuid.New().String()[:8]),
		"Integration test project for task execution",
		"https://github.com/test/repo.git",
	)
	require.NoError(t, err, "Failed to create test project")

	// Wait for pod to be created
	time.Sleep(2 * time.Second)

	return project
}

func createTestTask(t *testing.T, taskService service.TaskService, projectID, userID uuid.UUID) *model.Task {
	ctx := context.Background()

	task, err := taskService.CreateTask(ctx, projectID, userID,
		fmt.Sprintf("integration-test-%s", uuid.New().String()[:8]),
		"Test task for execution",
		model.TaskPriorityMedium,
	)
	require.NoError(t, err, "Failed to create test task")

	return task
}

// TestTaskExecution_FullLifecycle tests the complete task execution workflow
func TestTaskExecution_FullLifecycle_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	// Create test user
	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	// Create test project (with pod)
	project := createTestProject(t, db, projectService, testUser.ID)
	t.Logf("Created project %s with pod %s", project.ID, project.PodName)

	// Create test task
	task := createTestTask(t, taskService, project.ID, testUser.ID)
	t.Logf("Created task %s", task.ID)

	// Step 1: Execute task
	t.Run("ExecuteTask", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.ExecuteTask(c)

		assert.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response contains session_id and status
		assert.NotEmpty(t, response["session_id"])
		assert.Equal(t, string(model.SessionStatusPending), response["status"])

		sessionID := response["session_id"].(string)
		t.Logf("Started execution with session ID: %s", sessionID)

		// Step 2: Verify session created in database
		t.Run("VerifySessionCreated", func(t *testing.T) {
			var session model.Session
			err := db.First(&session, "id = ?", sessionID).Error
			require.NoError(t, err, "Session should be created in database")

			assert.Equal(t, task.ID, session.TaskID)
			assert.Equal(t, project.ID, session.ProjectID)
			assert.NotEmpty(t, session.Prompt)
			assert.Contains(t, []model.SessionStatus{model.SessionStatusPending, model.SessionStatusRunning}, session.Status)

			t.Logf("Session status: %s", session.Status)
		})

		// Step 3: Verify task status changed to IN_PROGRESS
		t.Run("VerifyTaskStatusChanged", func(t *testing.T) {
			var updatedTask model.Task
			err := db.First(&updatedTask, "id = ?", task.ID).Error
			require.NoError(t, err)

			assert.Equal(t, model.TaskStatusInProgress, updatedTask.Status, "Task should be IN_PROGRESS after execution")

			t.Logf("Task status: %s", updatedTask.Status)
		})
	})

	// Step 4: Cleanup - delete project
	t.Run("CleanupProject", func(t *testing.T) {
		err := projectService.DeleteProject(ctx, project.ID, testUser.ID)
		require.NoError(t, err)

		t.Logf("Cleaned up project %s", project.ID)
	})
}

// TestTaskExecution_StopSession tests stopping a running session
func TestTaskExecution_StopSession_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	// Execute task first
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("currentUser", testUser)
	c.Params = gin.Params{
		{Key: "id", Value: project.ID.String()},
		{Key: "taskId", Value: task.ID.String()},
	}
	taskHandler.ExecuteTask(c)
	require.Equal(t, http.StatusOK, w.Code)

	var execResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &execResponse)
	sessionID := execResponse["session_id"].(string)

	t.Logf("Started session %s", sessionID)

	// Now stop the session
	t.Run("StopSession", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/stop", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.StopTask(c)

		assert.Equal(t, http.StatusNoContent, w.Code, "Response: %s", w.Body.String())

		t.Logf("Stopped session successfully")
	})

	// Verify session status changed to CANCELLED
	t.Run("VerifySessionCancelled", func(t *testing.T) {
		var session model.Session
		err := db.First(&session, "id = ?", sessionID).Error
		require.NoError(t, err)

		assert.Equal(t, model.SessionStatusCancelled, session.Status, "Session should be CANCELLED after stop")

		t.Logf("Session final status: %s", session.Status)
	})

	// Verify task status reset to TODO
	t.Run("VerifyTaskResetToTodo", func(t *testing.T) {
		var updatedTask model.Task
		err := db.First(&updatedTask, "id = ?", task.ID).Error
		require.NoError(t, err)

		assert.Equal(t, model.TaskStatusTodo, updatedTask.Status, "Task should be TODO after stop")

		t.Logf("Task final status: %s", updatedTask.Status)
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_ConcurrentExecutionPrevented tests that concurrent executions are rejected
func TestTaskExecution_ConcurrentExecutionPrevented_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	// Execute task first time
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = req1
	c1.Set("currentUser", testUser)
	c1.Params = gin.Params{
		{Key: "id", Value: project.ID.String()},
		{Key: "taskId", Value: task.ID.String()},
	}
	taskHandler.ExecuteTask(c1)
	require.Equal(t, http.StatusOK, w1.Code)

	t.Logf("First execution succeeded")

	// Try to execute again (should fail with 409 Conflict)
	t.Run("SecondExecutionRejected", func(t *testing.T) {
		req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = req2
		c2.Set("currentUser", testUser)
		c2.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.ExecuteTask(c2)

		assert.Equal(t, http.StatusConflict, w2.Code, "Second execution should be rejected with 409 Conflict")

		t.Logf("Second execution correctly rejected: %s", w2.Body.String())
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_OpenCodeSidecarUnavailable tests graceful error handling when sidecar is unavailable
func TestTaskExecution_OpenCodeSidecarUnavailable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	// Create project (pod might not have OpenCode sidecar running yet)
	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	t.Run("ExecuteWithSidecarUnavailable", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.ExecuteTask(c)

		// Execution might succeed (session created) but OpenCode API call could fail
		// This is expected behavior - partial success model
		if w.Code == http.StatusOK {
			t.Logf("Execution succeeded despite potential sidecar unavailability")
		} else {
			t.Logf("Execution failed (expected if sidecar not ready): %s", w.Body.String())
		}

		// Either 200 (partial success) or 500 (OpenCode API call failed) is acceptable
		assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_GetTaskSessions tests retrieving execution history
func TestTaskExecution_GetTaskSessions_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	// Execute task to create a session
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("currentUser", testUser)
	c.Params = gin.Params{
		{Key: "id", Value: project.ID.String()},
		{Key: "taskId", Value: task.ID.String()},
	}
	taskHandler.ExecuteTask(c)
	require.Equal(t, http.StatusOK, w.Code)

	t.Logf("Created execution session")

	// Get task sessions
	t.Run("GetTaskSessions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s/tasks/%s/sessions", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.GetTaskSessions(c)

		assert.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

		var sessions []model.Session
		err := json.Unmarshal(w.Body.Bytes(), &sessions)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(sessions), 1, "Should have at least 1 session")

		// Verify session data
		session := sessions[0]
		assert.Equal(t, task.ID, session.TaskID)
		assert.Equal(t, project.ID, session.ProjectID)
		assert.NotEmpty(t, session.Prompt)

		t.Logf("Retrieved %d sessions for task", len(sessions))
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_InvalidTaskState tests execution rejection for invalid task states
func TestTaskExecution_InvalidTaskState_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	// Manually update task to DONE state
	db.Model(&task).Update("status", model.TaskStatusDone)

	t.Run("ExecuteCompletedTask", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.ExecuteTask(c)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Cannot execute task in DONE state")

		t.Logf("Execution correctly rejected for DONE task: %s", w.Body.String())
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_UnauthorizedAccess tests authorization for task execution
func TestTaskExecution_UnauthorizedAccess_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	otherUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	t.Run("ExecuteTaskAsOtherUser", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", otherUser) // Different user
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.ExecuteTask(c)

		assert.Equal(t, http.StatusForbidden, w.Code, "Other user should not be able to execute task")

		t.Logf("Execution correctly rejected for unauthorized user: %s", w.Body.String())
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_SessionListForProject tests listing all active sessions for a project
func TestTaskExecution_SessionListForProject_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)

	// Create and execute multiple tasks
	task1 := createTestTask(t, taskService, project.ID, testUser.ID)
	task2 := createTestTask(t, taskService, project.ID, testUser.ID)

	// Execute both tasks
	for _, task := range []*model.Task{task1, task2} {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}
		taskHandler.ExecuteTask(c)
		require.Equal(t, http.StatusOK, w.Code)
	}

	t.Logf("Created 2 execution sessions")

	// Verify we can query sessions across tasks
	t.Run("VerifyMultipleSessions", func(t *testing.T) {
		var sessions []model.Session
		err := db.Where("project_id = ?", project.ID).Find(&sessions).Error
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(sessions), 2, "Should have at least 2 sessions for project")

		t.Logf("Found %d sessions for project", len(sessions))
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_StopNonRunningTask tests stopping a task that's not running
func TestTaskExecution_StopNonRunningTask_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	// Task is in TODO state, try to stop it
	t.Run("StopTaskNotRunning", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/stop", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.StopTask(c)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Cannot stop task that's not running")

		t.Logf("Stop correctly rejected for TODO task: %s", w.Body.String())
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}

// TestTaskExecution_OutputStreamValidation tests SSE output stream endpoint validation
func TestTaskExecution_OutputStreamValidation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, taskHandler, taskService, projectService, cleanup := setupTaskExecutionIntegrationTest(t)
	defer cleanup()

	testUser := createTestUserForExecution(t, db)
	ctx := context.Background()

	project := createTestProject(t, db, projectService, testUser.ID)
	task := createTestTask(t, taskService, project.ID, testUser.ID)

	// Execute task first
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/execute", project.ID, task.ID), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("currentUser", testUser)
	c.Params = gin.Params{
		{Key: "id", Value: project.ID.String()},
		{Key: "taskId", Value: task.ID.String()},
	}
	taskHandler.ExecuteTask(c)
	require.Equal(t, http.StatusOK, w.Code)

	var execResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &execResponse)
	sessionID := execResponse["session_id"].(string)

	// Test output stream with missing session_id
	t.Run("OutputStreamMissingSessionID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s/tasks/%s/output", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.TaskOutputStream(c)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Missing session_id should return 400")
	})

	// Test output stream with invalid session_id
	t.Run("OutputStreamInvalidSessionID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s/tasks/%s/output?session_id=invalid-uuid", project.ID, task.ID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.TaskOutputStream(c)

		assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid session_id should return 400")
	})

	// Test output stream with valid session_id (should attempt to connect to sidecar)
	t.Run("OutputStreamValidSessionID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s/tasks/%s/output?session_id=%s", project.ID, task.ID, sessionID), nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("currentUser", testUser)
		c.Params = gin.Params{
			{Key: "id", Value: project.ID.String()},
			{Key: "taskId", Value: task.ID.String()},
		}

		taskHandler.TaskOutputStream(c)

		// SSE endpoint might return 502 if sidecar not reachable (expected in test environment)
		// or succeed if sidecar is available
		assert.Contains(t, []int{http.StatusOK, http.StatusBadGateway}, w.Code)

		t.Logf("Output stream response: %d - %s", w.Code, w.Body.String())
	})

	// Cleanup
	projectService.DeleteProject(ctx, project.ID, testUser.ID)
}
