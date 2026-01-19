package model

import (
	"time"

	"github.com/google/uuid"
)

type Interaction struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID      uuid.UUID  `gorm:"type:uuid;column:task_id;not null;index" json:"task_id"`
	SessionID   *uuid.UUID `gorm:"type:uuid;column:session_id" json:"session_id,omitempty"`
	UserID      uuid.UUID  `gorm:"type:uuid;column:user_id;not null" json:"user_id"`
	MessageType string     `gorm:"column:message_type;size:50;not null" json:"message_type"`
	Content     string     `gorm:"column:content;type:text;not null" json:"content"`
	Metadata    JSONB      `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	Task    *Task    `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Session *Session `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Interaction) TableName() string {
	return "interactions"
}
