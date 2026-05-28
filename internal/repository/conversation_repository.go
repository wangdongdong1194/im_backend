package repository

import (
	"context"

	"im_backend/internal/model"

	"gorm.io/gorm"
)

type ListConversationsOption struct {
	Offset int
	Limit  int
	Type   string
}

type ConversationRepository interface {
	Create(ctx context.Context, conversation *model.Conversation) error
	GetByID(ctx context.Context, id uint) (*model.Conversation, error)
	List(ctx context.Context, opt ListConversationsOption) ([]model.Conversation, error)
	Count(ctx context.Context, conversationType string) (int64, error)
	UpdateByID(ctx context.Context, id uint, changes map[string]any) error
	DeleteByID(ctx context.Context, id uint) error
}

type GormConversationRepository struct {
	db *gorm.DB
}

func NewGormConversationRepository(db *gorm.DB) *GormConversationRepository {
	return &GormConversationRepository{db: db}
}

func (r *GormConversationRepository) Create(ctx context.Context, conversation *model.Conversation) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if conversation == nil {
		return gorm.ErrInvalidData
	}

	return r.db.WithContext(ctx).Create(conversation).Error
}

func (r *GormConversationRepository) GetByID(ctx context.Context, id uint) (*model.Conversation, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var conversation model.Conversation
	if err := r.db.WithContext(ctx).First(&conversation, id).Error; err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (r *GormConversationRepository) List(ctx context.Context, opt ListConversationsOption) ([]model.Conversation, error) {
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

	query := r.db.WithContext(ctx).Model(&model.Conversation{})
	if opt.Type != "" {
		query = query.Where("type = ?", opt.Type)
	}

	items := make([]model.Conversation, 0, limit)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *GormConversationRepository) Count(ctx context.Context, conversationType string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, ErrRepositoryNotReady
	}

	query := r.db.WithContext(ctx).Model(&model.Conversation{})
	if conversationType != "" {
		query = query.Where("type = ?", conversationType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *GormConversationRepository) UpdateByID(ctx context.Context, id uint, changes map[string]any) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if len(changes) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&model.Conversation{}).Where("id = ?", id).Updates(changes).Error
}

func (r *GormConversationRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	return r.db.WithContext(ctx).Delete(&model.Conversation{}, id).Error
}
