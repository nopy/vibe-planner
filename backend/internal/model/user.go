package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OIDCSubject string     `gorm:"column:oidc_subject;uniqueIndex;not null" json:"oidc_subject"`
	Email       string     `gorm:"column:email;not null" json:"email"`
	Name        string     `gorm:"column:name" json:"name"`
	PictureURL  string     `gorm:"column:picture_url" json:"picture_url"`
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

var TimeNow = time.Now
