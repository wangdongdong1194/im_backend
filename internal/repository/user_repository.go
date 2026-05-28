package repository

import (
	"context"
	"errors"

	"im_backend/internal/model"

	"gorm.io/gorm"
)

var ErrRepositoryNotReady = errors.New("repository not ready")

type ListUsersOption struct {
	Offset int
	Limit  int
}

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUserID(ctx context.Context, userID string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, opt ListUsersOption) ([]model.User, error)
	Count(ctx context.Context) (int64, error)
	UpdateByID(ctx context.Context, id uint, changes map[string]any) error
	DeleteByID(ctx context.Context, id uint) error
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *model.User) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	if user == nil {
		return errors.New("user is nil")
	}

	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormUserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *GormUserRepository) GetByUserID(ctx context.Context, userID string) (*model.User, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var user model.User
	if err := r.db.WithContext(ctx).Where("erp = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *GormUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *GormUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *GormUserRepository) List(ctx context.Context, opt ListUsersOption) ([]model.User, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}

	if limit > 200 {
		limit = 200
	}

	offset := opt.Offset
	if offset < 0 {
		offset = 0
	}

	users := make([]model.User, 0, limit)
	if err := r.db.WithContext(ctx).
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *GormUserRepository) Count(ctx context.Context) (int64, error) {
	if r == nil || r.db == nil {
		return 0, ErrRepositoryNotReady
	}

	var total int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *GormUserRepository) UpdateByID(ctx context.Context, id uint, changes map[string]any) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	if len(changes) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(changes).Error
}

func (r *GormUserRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}
