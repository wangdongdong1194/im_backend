package repository

import (
	"context"

	"im_backend/internal/model"

	"gorm.io/gorm"
)

type ListFriendRequestsOption struct {
	ToUserID uint
	Status   string
	Offset   int
	Limit    int
}

type FriendRequestRepository interface {
	Create(ctx context.Context, req *model.FriendRequest) error
	GetByID(ctx context.Context, id uint) (*model.FriendRequest, error)
	ListByToUser(ctx context.Context, opt ListFriendRequestsOption) ([]model.FriendRequest, error)
	UpdateByID(ctx context.Context, id uint, changes map[string]any) error
	DeleteByID(ctx context.Context, id uint) error
}

type GormFriendRequestRepository struct {
	db *gorm.DB
}

func NewGormFriendRequestRepository(db *gorm.DB) *GormFriendRequestRepository {
	return &GormFriendRequestRepository{db: db}
}

func (r *GormFriendRequestRepository) Create(ctx context.Context, req *model.FriendRequest) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if req == nil {
		return gorm.ErrInvalidData
	}

	return r.db.WithContext(ctx).Create(req).Error
}

func (r *GormFriendRequestRepository) GetByID(ctx context.Context, id uint) (*model.FriendRequest, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var req model.FriendRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		return nil, err
	}

	return &req, nil
}

func (r *GormFriendRequestRepository) ListByToUser(ctx context.Context, opt ListFriendRequestsOption) ([]model.FriendRequest, error) {
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

	query := r.db.WithContext(ctx).Model(&model.FriendRequest{}).Where("to_user_id = ?", opt.ToUserID)
	if opt.Status != "" {
		query = query.Where("status = ?", opt.Status)
	}

	items := make([]model.FriendRequest, 0, limit)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *GormFriendRequestRepository) UpdateByID(ctx context.Context, id uint, changes map[string]any) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if len(changes) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&model.FriendRequest{}).Where("id = ?", id).Updates(changes).Error
}

func (r *GormFriendRequestRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	return r.db.WithContext(ctx).Delete(&model.FriendRequest{}, id).Error
}
