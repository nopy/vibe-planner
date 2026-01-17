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

func setupProjectTestDB(t *testing.T) *gorm.DB {
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

	// Create projects table
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

	// Create index on deleted_at
	err = db.Exec("CREATE INDEX idx_projects_deleted_at ON projects(deleted_at)").Error
	require.NoError(t, err)

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) uuid.UUID {
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

func TestNewProjectRepository(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)

	assert.NotNil(t, repo)
}

func TestProjectRepository_Create(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)

	tests := []struct {
		name    string
		project *model.Project
		wantErr bool
	}{
		{
			name: "valid project",
			project: &model.Project{
				UserID:      userID,
				Name:        "Test Project",
				Slug:        "test-project",
				Description: "A test project",
				RepoURL:     "https://github.com/test/repo",
				Status:      model.ProjectStatusInitializing,
			},
			wantErr: false,
		},
		{
			name: "minimal project",
			project: &model.Project{
				UserID: userID,
				Name:   "Minimal Project",
				Slug:   "minimal-project",
				Status: model.ProjectStatusInitializing,
			},
			wantErr: false,
		},
		{
			name: "project with pod metadata",
			project: &model.Project{
				UserID:           userID,
				Name:             "Pod Project",
				Slug:             "pod-project",
				PodName:          "project-abc123",
				PodNamespace:     "opencode",
				PodStatus:        "Running",
				WorkspacePVCName: "workspace-abc123",
				Status:           model.ProjectStatusReady,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.project)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.project.ID)
				assert.False(t, tt.project.CreatedAt.IsZero())
			}
		})
	}
}

func TestProjectRepository_FindByID(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)

	// Create test project
	testProject := &model.Project{
		UserID:      userID,
		Name:        "Test Project",
		Slug:        "test-project",
		Description: "Test description",
		Status:      model.ProjectStatusInitializing,
	}
	err := repo.Create(ctx, testProject)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing project",
			id:      testProject.ID,
			wantErr: false,
		},
		{
			name:    "non-existent project",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := repo.FindByID(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
				assert.Nil(t, project)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, project)
				assert.Equal(t, testProject.Name, project.Name)
				assert.Equal(t, testProject.Slug, project.Slug)
				assert.Equal(t, testProject.Description, project.Description)
				assert.Equal(t, testProject.UserID, project.UserID)
			}
		})
	}
}

func TestProjectRepository_FindByUserID(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	user1ID := createTestUser(t, db)
	user2ID := createTestUser(t, db)

	// Create projects for user1
	project1 := &model.Project{
		UserID: user1ID,
		Name:   "Project 1",
		Slug:   "project-1",
		Status: model.ProjectStatusInitializing,
	}
	err := repo.Create(ctx, project1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	project2 := &model.Project{
		UserID: user1ID,
		Name:   "Project 2",
		Slug:   "project-2",
		Status: model.ProjectStatusReady,
	}
	err = repo.Create(ctx, project2)
	require.NoError(t, err)

	// Create project for user2
	project3 := &model.Project{
		UserID: user2ID,
		Name:   "Project 3",
		Slug:   "project-3",
		Status: model.ProjectStatusInitializing,
	}
	err = repo.Create(ctx, project3)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        uuid.UUID
		expectedCount int
	}{
		{
			name:          "user with 2 projects",
			userID:        user1ID,
			expectedCount: 2,
		},
		{
			name:          "user with 1 project",
			userID:        user2ID,
			expectedCount: 1,
		},
		{
			name:          "user with no projects",
			userID:        uuid.New(),
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := repo.FindByUserID(ctx, tt.userID)

			assert.NoError(t, err)
			assert.Len(t, projects, tt.expectedCount)

			// Verify ordering (DESC by created_at)
			if len(projects) > 1 {
				assert.True(t, projects[0].CreatedAt.After(projects[1].CreatedAt) ||
					projects[0].CreatedAt.Equal(projects[1].CreatedAt))
			}
		})
	}
}

func TestProjectRepository_Update(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)

	// Create test project
	testProject := &model.Project{
		UserID:      userID,
		Name:        "Original Name",
		Slug:        "original-slug",
		Description: "Original description",
		Status:      model.ProjectStatusInitializing,
	}
	err := repo.Create(ctx, testProject)
	require.NoError(t, err)

	// Update project
	testProject.Name = "Updated Name"
	testProject.Description = "Updated description"
	testProject.Status = model.ProjectStatusReady
	testProject.PodName = "project-abc123"
	testProject.PodNamespace = "opencode"
	testProject.PodStatus = "Running"

	err = repo.Update(ctx, testProject)
	assert.NoError(t, err)

	// Verify update
	updatedProject, err := repo.FindByID(ctx, testProject.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedProject.Name)
	assert.Equal(t, "Updated description", updatedProject.Description)
	assert.Equal(t, model.ProjectStatusReady, updatedProject.Status)
	assert.Equal(t, "project-abc123", updatedProject.PodName)
	assert.Equal(t, "opencode", updatedProject.PodNamespace)
	assert.Equal(t, "Running", updatedProject.PodStatus)
}

func TestProjectRepository_SoftDelete(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)

	// Create test project
	testProject := &model.Project{
		UserID: userID,
		Name:   "Test Project",
		Slug:   "test-project",
		Status: model.ProjectStatusInitializing,
	}
	err := repo.Create(ctx, testProject)
	require.NoError(t, err)

	// Soft delete
	err = repo.SoftDelete(ctx, testProject.ID)
	assert.NoError(t, err)

	// Verify project is soft deleted (FindByID should not find it)
	project, err := repo.FindByID(ctx, testProject.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.Nil(t, project)

	// Verify deleted_at is set in database
	var deletedAt *time.Time
	err = db.Raw("SELECT deleted_at FROM projects WHERE id = ?", testProject.ID.String()).Scan(&deletedAt).Error
	require.NoError(t, err)
	assert.NotNil(t, deletedAt)
}

func TestProjectRepository_UpdatePodStatus(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)

	// Create test project
	testProject := &model.Project{
		UserID: userID,
		Name:   "Test Project",
		Slug:   "test-project",
		Status: model.ProjectStatusInitializing,
	}
	err := repo.Create(ctx, testProject)
	require.NoError(t, err)

	tests := []struct {
		name      string
		status    string
		podError  string
		wantError bool
	}{
		{
			name:      "update to Running",
			status:    "Running",
			podError:  "",
			wantError: false,
		},
		{
			name:      "update to Failed with error",
			status:    "Failed",
			podError:  "ImagePullBackOff: failed to pull image",
			wantError: false,
		},
		{
			name:      "update to Pending without error",
			status:    "Pending",
			podError:  "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdatePodStatus(ctx, testProject.ID, tt.status, tt.podError)
			assert.NoError(t, err)

			// Verify update
			project, err := repo.FindByID(ctx, testProject.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.status, project.PodStatus)
			if tt.podError != "" {
				assert.Equal(t, tt.podError, project.PodError)
			}
		})
	}

	// Test updating non-existent project
	t.Run("update non-existent project", func(t *testing.T) {
		err := repo.UpdatePodStatus(ctx, uuid.New(), "Running", "")
		// GORM's Updates on non-existent record doesn't return error, just affects 0 rows
		// So this test verifies no panic or unexpected error
		assert.NoError(t, err)
	})
}

func TestProjectRepository_FindByUserID_ExcludesSoftDeleted(t *testing.T) {
	db := setupProjectTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	userID := createTestUser(t, db)

	// Create 2 projects
	project1 := &model.Project{
		UserID: userID,
		Name:   "Active Project",
		Slug:   "active-project",
		Status: model.ProjectStatusReady,
	}
	err := repo.Create(ctx, project1)
	require.NoError(t, err)

	project2 := &model.Project{
		UserID: userID,
		Name:   "Deleted Project",
		Slug:   "deleted-project",
		Status: model.ProjectStatusReady,
	}
	err = repo.Create(ctx, project2)
	require.NoError(t, err)

	// Soft delete project2
	err = repo.SoftDelete(ctx, project2.ID)
	require.NoError(t, err)

	// Find by user ID should only return active project
	projects, err := repo.FindByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Equal(t, project1.ID, projects[0].ID)
	assert.Equal(t, "Active Project", projects[0].Name)
}
