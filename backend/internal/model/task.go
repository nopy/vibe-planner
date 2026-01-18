package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskStatusTodo        TaskStatus = "todo"
	TaskStatusInProgress  TaskStatus = "in_progress"
	TaskStatusAIReview    TaskStatus = "ai_review"
	TaskStatusHumanReview TaskStatus = "human_review"
	TaskStatusDone        TaskStatus = "done"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID           uuid.UUID      `gorm:"type:uuid;column:project_id;not null;index" json:"project_id"`
	Title               string         `gorm:"column:title;not null" json:"title"`
	Description         string         `gorm:"column:description;type:text" json:"description"`
	Status              TaskStatus     `gorm:"column:status;type:varchar(20);default:'todo'" json:"status"`
	Position            int            `gorm:"column:position;not null;default:0" json:"position"`
	Priority            TaskPriority   `gorm:"column:priority;type:varchar(20);default:'medium'" json:"priority"`
	AssignedTo          *uuid.UUID     `gorm:"type:uuid;column:assigned_to" json:"assigned_to,omitempty"`
	CurrentSessionID    *uuid.UUID     `gorm:"type:uuid;column:current_session_id" json:"current_session_id,omitempty"`
	OpenCodeOutput      string         `gorm:"column:opencode_output;type:text" json:"opencode_output,omitempty"`
	ExecutionDurationMs int64          `gorm:"column:execution_duration_ms" json:"execution_duration_ms"`
	FileReferences      string         `gorm:"column:file_references;type:jsonb" json:"file_references,omitempty"`
	CreatedBy           uuid.UUID      `gorm:"type:uuid;column:created_by;not null" json:"created_by"`
	CreatedAt           time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	Project  *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Creator  *User    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Assignee *User    `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
}

func (Task) TableName() string {
	return "tasks"
}
