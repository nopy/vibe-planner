package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

func setupTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Create users table (required for foreign key)
	createUsersTableSQL := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			oidc_subject TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			name TEXT,
			picture_url TEXT,
			last_login_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`
	err = db.Exec(createUsersTableSQL).Error
	require.NoError(t, err)

	// Create projects table (required for foreign key)
	createProjectsTableSQL := `
		CREATE TABLE projects (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			description TEXT,
			repo_url TEXT,
			pod_name TEXT,
			pod_namespace TEXT,
			pod_status TEXT,
			workspace_pvc_name TEXT,
			pod_created_at DATETIME,
			pod_error TEXT,
			status TEXT NOT NULL DEFAULT 'initializing',
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`
	err = db.Exec(createProjectsTableSQL).Error
	require.NoError(t, err)

	// Create tasks table
	createTasksTableSQL := `
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'todo',
			position INTEGER NOT NULL DEFAULT 0,
			priority TEXT DEFAULT 'medium',
			assigned_to TEXT,
			current_session_id TEXT,
			opencode_output TEXT,
			execution_duration_ms INTEGER,
			file_references TEXT,
			created_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY (created_by) REFERENCES users(id),
			FOREIGN KEY (assigned_to) REFERENCES users(id)
		)
	`
	err = db.Exec(createTasksTableSQL).Error
	require.NoError(t, err)

	// Create indexes
	err = db.Exec("CREATE INDEX idx_tasks_project_id ON tasks(project_id)").Error
	require.NoError(t, err)
	err = db.Exec("CREATE INDEX idx_tasks_project_position ON tasks(project_id, position)").Error
	require.NoError(t, err)
	err = db.Exec("CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at)").Error
	require.NoError(t, err)

	return db
}

func createTestUserForTask(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	err := db.Exec("INSERT INTO users (id, oidc_subject, email, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		userID.String(),
		"test-subject-"+userID.String(),
		"test@example.com",
		"Test User",
		time.Now(),
		time.Now(),
	).Error
	require.NoError(t, err)

	return userID
}

func createTestProject(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()

	projectID := uuid.New()
	err := db.Exec("INSERT INTO projects (id, user_id, name, slug, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		projectID.String(),
		userID.String(),
		"Test Project",
		"test-project",
		"ready",
		time.Now(),
		time.Now(),
	).Error
	require.NoError(t, err)

	return projectID
}

func TestNewTaskRepository(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	assert.NotNil(t, repo)
}

func TestTaskRepository_Create(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	tests := []struct {
		name    string
		task    *model.Task
		wantErr bool
	}{
		{
			name: "valid task with all fields",
			task: &model.Task{
				ProjectID:   projectID,
				Title:       "Test Task",
				Description: "A test task description",
				Status:      model.TaskStatusTodo,
				Position:    0,
				Priority:    model.TaskPriorityMedium,
				CreatedBy:   userID,
			},
			wantErr: false,
		},
		{
			name: "minimal task",
			task: &model.Task{
				ProjectID: projectID,
				Title:     "Minimal Task",
				Status:    model.TaskStatusTodo,
				Position:  1,
				CreatedBy: userID,
			},
			wantErr: false,
		},
		{
			name: "high priority task",
			task: &model.Task{
				ProjectID: projectID,
				Title:     "High Priority Task",
				Status:    model.TaskStatusTodo,
				Position:  2,
				Priority:  model.TaskPriorityHigh,
				CreatedBy: userID,
			},
			wantErr: false,
		},
		{
			name: "in progress task",
			task: &model.Task{
				ProjectID: projectID,
				Title:     "In Progress Task",
				Status:    model.TaskStatusInProgress,
				Position:  0,
				Priority:  model.TaskPriorityMedium,
				CreatedBy: userID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.task)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.task.ID)
				assert.False(t, tt.task.CreatedAt.IsZero())
			}
		})
	}
}

func TestTaskRepository_FindByID(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	// Create test task
	testTask := &model.Task{
		ProjectID:   projectID,
		Title:       "Test Task",
		Description: "Test description",
		Status:      model.TaskStatusTodo,
		Position:    0,
		Priority:    model.TaskPriorityMedium,
		CreatedBy:   userID,
	}
	err := repo.Create(ctx, testTask)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing task",
			id:      testTask.ID,
			wantErr: false,
		},
		{
			name:    "non-existent task",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := repo.FindByID(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
				assert.Equal(t, testTask.Title, task.Title)
				assert.Equal(t, testTask.Description, task.Description)
				assert.Equal(t, testTask.Status, task.Status)
				assert.Equal(t, testTask.Position, task.Position)
				assert.Equal(t, testTask.Priority, task.Priority)
			}
		})
	}
}

func TestTaskRepository_FindByProjectID(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	project1ID := createTestProject(t, db, userID)
	project2ID := createTestProject(t, db, userID)

	// Create tasks for project1 with specific positions
	task1 := &model.Task{
		ProjectID: project1ID,
		Title:     "Task 1",
		Status:    model.TaskStatusTodo,
		Position:  2,
		CreatedBy: userID,
	}
	err := repo.Create(ctx, task1)
	require.NoError(t, err)

	task2 := &model.Task{
		ProjectID: project1ID,
		Title:     "Task 2",
		Status:    model.TaskStatusInProgress,
		Position:  0,
		CreatedBy: userID,
	}
	err = repo.Create(ctx, task2)
	require.NoError(t, err)

	task3 := &model.Task{
		ProjectID: project1ID,
		Title:     "Task 3",
		Status:    model.TaskStatusDone,
		Position:  1,
		CreatedBy: userID,
	}
	err = repo.Create(ctx, task3)
	require.NoError(t, err)

	// Create task for project2
	task4 := &model.Task{
		ProjectID: project2ID,
		Title:     "Task 4",
		Status:    model.TaskStatusTodo,
		Position:  0,
		CreatedBy: userID,
	}
	err = repo.Create(ctx, task4)
	require.NoError(t, err)

	tests := []struct {
		name          string
		projectID     uuid.UUID
		expectedCount int
		expectedOrder []string
	}{
		{
			name:          "project with 3 tasks",
			projectID:     project1ID,
			expectedCount: 3,
			expectedOrder: []string{"Task 2", "Task 3", "Task 1"}, // Ordered by position ASC (0, 1, 2)
		},
		{
			name:          "project with 1 task",
			projectID:     project2ID,
			expectedCount: 1,
			expectedOrder: []string{"Task 4"},
		},
		{
			name:          "project with no tasks",
			projectID:     uuid.New(),
			expectedCount: 0,
			expectedOrder: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := repo.FindByProjectID(ctx, tt.projectID)

			assert.NoError(t, err)
			assert.Len(t, tasks, tt.expectedCount)

			// Verify ordering (ASC by position)
			for i, expectedTitle := range tt.expectedOrder {
				assert.Equal(t, expectedTitle, tasks[i].Title)
			}

			// Verify position ordering
			if len(tasks) > 1 {
				for i := 1; i < len(tasks); i++ {
					assert.True(t, tasks[i-1].Position <= tasks[i].Position)
				}
			}
		})
	}
}

func TestTaskRepository_Update(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	// Create test task
	testTask := &model.Task{
		ProjectID:   projectID,
		Title:       "Original Title",
		Description: "Original description",
		Status:      model.TaskStatusTodo,
		Position:    0,
		Priority:    model.TaskPriorityMedium,
		CreatedBy:   userID,
	}
	err := repo.Create(ctx, testTask)
	require.NoError(t, err)

	// Update task
	testTask.Title = "Updated Title"
	testTask.Description = "Updated description"
	testTask.Status = model.TaskStatusInProgress
	testTask.Position = 5
	testTask.Priority = model.TaskPriorityHigh

	err = repo.Update(ctx, testTask)
	assert.NoError(t, err)

	// Verify update
	updatedTask, err := repo.FindByID(ctx, testTask.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedTask.Title)
	assert.Equal(t, "Updated description", updatedTask.Description)
	assert.Equal(t, model.TaskStatusInProgress, updatedTask.Status)
	assert.Equal(t, 5, updatedTask.Position)
	assert.Equal(t, model.TaskPriorityHigh, updatedTask.Priority)
}

func TestTaskRepository_UpdateStatus(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	// Create test task
	testTask := &model.Task{
		ProjectID: projectID,
		Title:     "Test Task",
		Status:    model.TaskStatusTodo,
		Position:  0,
		CreatedBy: userID,
	}
	err := repo.Create(ctx, testTask)
	require.NoError(t, err)

	tests := []struct {
		name      string
		newStatus model.TaskStatus
	}{
		{
			name:      "update to in_progress",
			newStatus: model.TaskStatusInProgress,
		},
		{
			name:      "update to ai_review",
			newStatus: model.TaskStatusAIReview,
		},
		{
			name:      "update to human_review",
			newStatus: model.TaskStatusHumanReview,
		},
		{
			name:      "update to done",
			newStatus: model.TaskStatusDone,
		},
		{
			name:      "update back to todo",
			newStatus: model.TaskStatusTodo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateStatus(ctx, testTask.ID, tt.newStatus)
			assert.NoError(t, err)

			// Verify update
			task, err := repo.FindByID(ctx, testTask.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.newStatus, task.Status)
		})
	}

	// Test updating non-existent task
	t.Run("update non-existent task", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, uuid.New(), model.TaskStatusDone)
		// GORM's Updates on non-existent record doesn't return error, just affects 0 rows
		assert.NoError(t, err)
	})
}

func TestTaskRepository_UpdatePosition(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	// Create test task
	testTask := &model.Task{
		ProjectID: projectID,
		Title:     "Test Task",
		Status:    model.TaskStatusTodo,
		Position:  0,
		CreatedBy: userID,
	}
	err := repo.Create(ctx, testTask)
	require.NoError(t, err)

	tests := []struct {
		name        string
		newPosition int
	}{
		{
			name:        "update to position 1",
			newPosition: 1,
		},
		{
			name:        "update to position 5",
			newPosition: 5,
		},
		{
			name:        "update back to position 0",
			newPosition: 0,
		},
		{
			name:        "update to position 100",
			newPosition: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdatePosition(ctx, testTask.ID, tt.newPosition)
			assert.NoError(t, err)

			// Verify update
			task, err := repo.FindByID(ctx, testTask.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.newPosition, task.Position)
		})
	}

	// Test updating non-existent task
	t.Run("update non-existent task", func(t *testing.T) {
		err := repo.UpdatePosition(ctx, uuid.New(), 10)
		// GORM's Updates on non-existent record doesn't return error, just affects 0 rows
		assert.NoError(t, err)
	})
}

func TestTaskRepository_SoftDelete(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	// Create test task
	testTask := &model.Task{
		ProjectID: projectID,
		Title:     "Test Task",
		Status:    model.TaskStatusTodo,
		Position:  0,
		CreatedBy: userID,
	}
	err := repo.Create(ctx, testTask)
	require.NoError(t, err)

	// Soft delete
	err = repo.SoftDelete(ctx, testTask.ID)
	assert.NoError(t, err)

	// Verify task is soft deleted (FindByID should not find it)
	task, err := repo.FindByID(ctx, testTask.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.Nil(t, task)

	// Verify deleted_at is set in database
	var deletedAt *time.Time
	err = db.Raw("SELECT deleted_at FROM tasks WHERE id = ?", testTask.ID.String()).Scan(&deletedAt).Error
	require.NoError(t, err)
	assert.NotNil(t, deletedAt)
}

func TestTaskRepository_FindByProjectID_ExcludesSoftDeleted(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	// Create 2 tasks
	task1 := &model.Task{
		ProjectID: projectID,
		Title:     "Active Task",
		Status:    model.TaskStatusTodo,
		Position:  0,
		CreatedBy: userID,
	}
	err := repo.Create(ctx, task1)
	require.NoError(t, err)

	task2 := &model.Task{
		ProjectID: projectID,
		Title:     "Deleted Task",
		Status:    model.TaskStatusTodo,
		Position:  1,
		CreatedBy: userID,
	}
	err = repo.Create(ctx, task2)
	require.NoError(t, err)

	// Soft delete task2
	err = repo.SoftDelete(ctx, task2.ID)
	require.NoError(t, err)

	// Find by project ID should only return active task
	tasks, err := repo.FindByProjectID(ctx, projectID)
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
	assert.Equal(t, "Active Task", tasks[0].Title)
}

func TestTaskRepository_Create_WithAssignedTo(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	userID := createTestUserForTask(t, db)
	assigneeID := createTestUserForTask(t, db)
	projectID := createTestProject(t, db, userID)

	// Create task with assigned_to
	testTask := &model.Task{
		ProjectID:  projectID,
		Title:      "Assigned Task",
		Status:     model.TaskStatusTodo,
		Position:   0,
		CreatedBy:  userID,
		AssignedTo: &assigneeID,
	}
	err := repo.Create(ctx, testTask)
	assert.NoError(t, err)

	// Verify assignment
	task, err := repo.FindByID(ctx, testTask.ID)
	require.NoError(t, err)
	assert.NotNil(t, task.AssignedTo)
	assert.Equal(t, assigneeID, *task.AssignedTo)
}
