package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusCancelled SessionStatus = "cancelled"
)

type Session struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID      uuid.UUID      `gorm:"type:uuid;column:task_id;not null;index" json:"task_id"`
	ProjectID   uuid.UUID      `gorm:"type:uuid;column:project_id;not null;index" json:"project_id"`
	Status      SessionStatus  `gorm:"column:status;type:varchar(20);default:'pending'" json:"status"`
	Prompt      string         `gorm:"column:prompt;type:text" json:"prompt,omitempty"`
	Output      string         `gorm:"column:output;type:text" json:"output,omitempty"`
	Error       string         `gorm:"column:error;type:text" json:"error,omitempty"`
	StartedAt   *time.Time     `gorm:"column:started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time     `gorm:"column:completed_at" json:"completed_at,omitempty"`
	DurationMs  int64          `gorm:"column:duration_ms" json:"duration_ms"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	Task    *Task    `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
}

func (Session) TableName() string {
	return "sessions"
}
