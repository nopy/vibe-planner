package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusTodo        TaskStatus = "todo"
	TaskStatusInProgress  TaskStatus = "in_progress"
	TaskStatusAIReview    TaskStatus = "ai_review"
	TaskStatusHumanReview TaskStatus = "human_review"
	TaskStatusDone        TaskStatus = "done"
)

type Task struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProjectID           uuid.UUID  `gorm:"type:uuid;not null;index" json:"project_id"`
	Title               string     `gorm:"not null" json:"title"`
	Description         string     `json:"description"`
	Status              TaskStatus `gorm:"type:varchar(20);default:'todo'" json:"status"`
	CurrentSessionID    *uuid.UUID `gorm:"type:uuid" json:"current_session_id,omitempty"`
	OpenCodeOutput      string     `gorm:"type:text" json:"opencode_output,omitempty"`
	ExecutionDurationMs int64      `json:"execution_duration_ms"`
	FileReferences      string     `gorm:"type:jsonb" json:"file_references,omitempty"`
	CreatedBy           uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Creator *User    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (Task) TableName() string {
	return "tasks"
}
