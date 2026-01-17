package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectStatus string

const (
	ProjectStatusInitializing ProjectStatus = "initializing"
	ProjectStatusReady        ProjectStatus = "ready"
	ProjectStatusError        ProjectStatus = "error"
	ProjectStatusArchived     ProjectStatus = "archived"
)

type Project struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;column:user_id;not null;index" json:"user_id"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	Slug        string    `gorm:"column:slug;not null;index" json:"slug"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	RepoURL     string    `gorm:"column:repo_url;type:text" json:"repo_url"`

	// Kubernetes metadata
	PodName          string     `gorm:"column:pod_name" json:"pod_name"`
	PodNamespace     string     `gorm:"column:pod_namespace" json:"pod_namespace"`
	PodStatus        string     `gorm:"column:pod_status" json:"pod_status"`
	WorkspacePVCName string     `gorm:"column:workspace_pvc_name" json:"workspace_pvc_name"`
	PodCreatedAt     *time.Time `gorm:"column:pod_created_at" json:"pod_created_at"`
	PodError         string     `gorm:"column:pod_error;type:text" json:"pod_error"`

	Status    ProjectStatus  `gorm:"column:status;type:varchar(20);default:'initializing';index" json:"status"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Project) TableName() string {
	return "projects"
}
