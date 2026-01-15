package model

import (
	"time"

	"github.com/google/uuid"
)

type ProjectStatus string

const (
	ProjectStatusInitializing ProjectStatus = "initializing"
	ProjectStatusReady        ProjectStatus = "ready"
	ProjectStatusError        ProjectStatus = "error"
	ProjectStatusArchived     ProjectStatus = "archived"
)

type Project struct {
	ID               uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID     `gorm:"type:uuid;not null;index" json:"user_id"`
	Name             string        `gorm:"not null" json:"name"`
	Slug             string        `gorm:"not null;index" json:"slug"`
	Description      string        `json:"description"`
	PodName          string        `json:"pod_name"`
	PodNamespace     string        `json:"pod_namespace"`
	PodStatus        string        `json:"pod_status"`
	WorkspacePVCName string        `json:"workspace_pvc_name"`
	Status           ProjectStatus `gorm:"type:varchar(20);default:'initializing'" json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Project) TableName() string {
	return "projects"
}
