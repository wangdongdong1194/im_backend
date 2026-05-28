package repository

import (
	"context"

	"im_backend/internal/model"

	"gorm.io/gorm"
)

type ListConversationMembersOption struct {
	ConversationID uint
	Offset         int
	Limit          int
}

type ConversationMemberRepository interface {
	Create(ctx context.Context, member *model.ConversationMember) error
	GetByID(ctx context.Context, id uint) (*model.ConversationMember, error)
	GetByConversationAndUser(ctx context.Context, conversationID uint, userID uint) (*model.ConversationMember, error)
	ListByConversation(ctx context.Context, opt ListConversationMembersOption) ([]model.ConversationMember, error)
	CountByConversation(ctx context.Context, conversationID uint) (int64, error)
	UpdateByID(ctx context.Context, id uint, changes map[string]any) error
	DeleteByID(ctx context.Context, id uint) error
}

type GormConversationMemberRepository struct {
	db *gorm.DB
}

func NewGormConversationMemberRepository(db *gorm.DB) *GormConversationMemberRepository {
	return &GormConversationMemberRepository{db: db}
}

func (r *GormConversationMemberRepository) Create(ctx context.Context, member *model.ConversationMember) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if member == nil {
		return gorm.ErrInvalidData
	}

	return r.db.WithContext(ctx).Create(member).Error
}

func (r *GormConversationMemberRepository) GetByID(ctx context.Context, id uint) (*model.ConversationMember, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var member model.ConversationMember
	if err := r.db.WithContext(ctx).First(&member, id).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

func (r *GormConversationMemberRepository) GetByConversationAndUser(ctx context.Context, conversationID uint, userID uint) (*model.ConversationMember, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var member model.ConversationMember
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

func (r *GormConversationMemberRepository) ListByConversation(ctx context.Context, opt ListConversationMembersOption) ([]model.ConversationMember, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	limit := opt.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := opt.Offset
	if offset < 0 {
		offset = 0
	}

	items := make([]model.ConversationMember, 0, limit)
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", opt.ConversationID).
		Order("id ASC").
		Offset(offset).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *GormConversationMemberRepository) CountByConversation(ctx context.Context, conversationID uint) (int64, error) {
	if r == nil || r.db == nil {
		return 0, ErrRepositoryNotReady
	}

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ?", conversationID).
		Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *GormConversationMemberRepository) UpdateByID(ctx context.Context, id uint, changes map[string]any) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if len(changes) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&model.ConversationMember{}).Where("id = ?", id).Updates(changes).Error
}

func (r *GormConversationMemberRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	return r.db.WithContext(ctx).Delete(&model.ConversationMember{}, id).Error
}
