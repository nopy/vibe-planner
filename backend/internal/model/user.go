package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OIDCSubject string     `gorm:"uniqueIndex;not null" json:"oidc_subject"`
	Email       string     `gorm:"not null" json:"email"`
	Name        string     `json:"name"`
	PictureURL  string     `json:"picture_url"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
