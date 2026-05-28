package repository

import (
	"context"

	"im_backend/internal/model"

	"gorm.io/gorm"
)

type ListFriendshipsOption struct {
	UserID uint
	Offset int
	Limit  int
}

type FriendshipRepository interface {
	Create(ctx context.Context, friendship *model.Friendship) error
	GetByID(ctx context.Context, id uint) (*model.Friendship, error)
	GetByPair(ctx context.Context, userID uint, friendID uint) (*model.Friendship, error)
	ListByUser(ctx context.Context, opt ListFriendshipsOption) ([]model.Friendship, error)
	DeleteByID(ctx context.Context, id uint) error
}

type GormFriendshipRepository struct {
	db *gorm.DB
}

func NewGormFriendshipRepository(db *gorm.DB) *GormFriendshipRepository {
	return &GormFriendshipRepository{db: db}
}

func (r *GormFriendshipRepository) Create(ctx context.Context, friendship *model.Friendship) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if friendship == nil {
		return gorm.ErrInvalidData
	}

	return r.db.WithContext(ctx).Create(friendship).Error
}

func (r *GormFriendshipRepository) GetByID(ctx context.Context, id uint) (*model.Friendship, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var friendship model.Friendship
	if err := r.db.WithContext(ctx).First(&friendship, id).Error; err != nil {
		return nil, err
	}

	return &friendship, nil
}

func (r *GormFriendshipRepository) GetByPair(ctx context.Context, userID uint, friendID uint) (*model.Friendship, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var friendship model.Friendship
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		First(&friendship).Error; err != nil {
		return nil, err
	}

	return &friendship, nil
}

func (r *GormFriendshipRepository) ListByUser(ctx context.Context, opt ListFriendshipsOption) ([]model.Friendship, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	limit := opt.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := opt.Offset
	if offset < 0 {
		offset = 0
	}

	items := make([]model.Friendship, 0, limit)
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", opt.UserID).
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *GormFriendshipRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	return r.db.WithContext(ctx).Delete(&model.Friendship{}, id).Error
}
