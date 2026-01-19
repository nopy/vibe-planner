package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OpenCodeConfig represents an OpenCode agent configuration with versioning
type OpenCodeConfig struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID uuid.UUID `gorm:"type:uuid;column:project_id;not null;index" json:"project_id"`
	Version   int       `gorm:"column:version;not null;default:1" json:"version"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true" json:"is_active"`

	// Model configuration
	ModelProvider string  `gorm:"column:model_provider;size:50;not null" json:"model_provider"`
	ModelName     string  `gorm:"column:model_name;size:100;not null" json:"model_name"`
	ModelVersion  *string `gorm:"column:model_version;size:50" json:"model_version,omitempty"`

	// Provider configuration
	APIEndpoint     *string `gorm:"column:api_endpoint;type:text" json:"api_endpoint,omitempty"`
	APIKeyEncrypted []byte  `gorm:"column:api_key_encrypted;type:bytea" json:"-"` // Never expose in JSON
	Temperature     float64 `gorm:"column:temperature;type:decimal(3,2);default:0.7" json:"temperature"`
	MaxTokens       int     `gorm:"column:max_tokens;default:4096" json:"max_tokens"`

	// Tools configuration
	EnabledTools ToolsList `gorm:"column:enabled_tools;type:jsonb;not null;default:'[\"file_ops\",\"web_search\",\"code_exec\"]'" json:"enabled_tools"`
	ToolsConfig  JSONB     `gorm:"column:tools_config;type:jsonb" json:"tools_config,omitempty"`

	// System configuration
	SystemPrompt   *string `gorm:"column:system_prompt;type:text" json:"system_prompt,omitempty"`
	MaxIterations  int     `gorm:"column:max_iterations;default:10" json:"max_iterations"`
	TimeoutSeconds int     `gorm:"column:timeout_seconds;default:300" json:"timeout_seconds"`

	// Metadata
	CreatedBy uuid.UUID `gorm:"type:uuid;column:created_by;not null" json:"created_by"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relations
	Project *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Creator *User    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName specifies the table name for GORM
func (OpenCodeConfig) TableName() string {
	return "opencode_configs"
}

// ToolsList is a custom type for JSONB array storage of tool names
type ToolsList []string

// Value implements the driver.Valuer interface for database writes
func (t ToolsList) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

// Scan implements the sql.Scanner interface for database reads
func (t *ToolsList) Scan(value interface{}) error {
	if value == nil {
		*t = nil
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	return json.Unmarshal(b, t)
}

// JSONB is a custom type for generic JSONB storage
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for database writes
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database reads
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	if len(b) == 0 {
		*j = make(JSONB)
		return nil
	}

	return json.Unmarshal(b, j)
}
