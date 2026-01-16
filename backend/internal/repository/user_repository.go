package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/npinot/vibe/backend/internal/model"
)

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByOIDCSubject(ctx context.Context, subject string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	CreateOrUpdateFromOIDC(ctx context.Context, subject, email, name, picture string) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) FindByOIDCSubject(ctx context.Context, subject string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("oidc_subject = ?", subject).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find user by OIDC subject: %w", err)
	}

	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *userRepository) CreateOrUpdateFromOIDC(ctx context.Context, subject, email, name, picture string) (*model.User, error) {
	user, err := r.FindByOIDCSubject(ctx, subject)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := model.TimeNow()
	if user == nil {
		user = &model.User{
			OIDCSubject: subject,
			Email:       email,
			Name:        name,
			PictureURL:  picture,
			LastLoginAt: &now,
		}
		if err := r.Create(ctx, user); err != nil {
			return nil, err
		}
	} else {
		user.Email = email
		user.Name = name
		user.PictureURL = picture
		user.LastLoginAt = &now
		if err := r.Update(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}
