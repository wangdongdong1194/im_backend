package repository

import (
	"context"

	"im_backend/internal/model"

	"gorm.io/gorm"
)

type ListMessagesOption struct {
	ConversationID uint
	BeforeID       uint
	Offset         int
	Limit          int
}

type MessageRepository interface {
	Create(ctx context.Context, message *model.Message) error
	GetByID(ctx context.Context, id uint) (*model.Message, error)
	GetByConversationAndClientMsgID(ctx context.Context, conversationID uint, clientMsgID string) (*model.Message, error)
	ListByConversation(ctx context.Context, opt ListMessagesOption) ([]model.Message, error)
	CountByConversation(ctx context.Context, conversationID uint) (int64, error)
	UpdateByID(ctx context.Context, id uint, changes map[string]any) error
	DeleteByID(ctx context.Context, id uint) error
}

type GormMessageRepository struct {
	db *gorm.DB
}

func NewGormMessageRepository(db *gorm.DB) *GormMessageRepository {
	return &GormMessageRepository{db: db}
}

func (r *GormMessageRepository) Create(ctx context.Context, message *model.Message) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if message == nil {
		return gorm.ErrInvalidData
	}

	return r.db.WithContext(ctx).Create(message).Error
}

func (r *GormMessageRepository) GetByID(ctx context.Context, id uint) (*model.Message, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var message model.Message
	if err := r.db.WithContext(ctx).First(&message, id).Error; err != nil {
		return nil, err
	}

	return &message, nil
}

func (r *GormMessageRepository) GetByConversationAndClientMsgID(ctx context.Context, conversationID uint, clientMsgID string) (*model.Message, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryNotReady
	}

	var message model.Message
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND client_msg_id = ?", conversationID, clientMsgID).
		First(&message).Error; err != nil {
		return nil, err
	}

	return &message, nil
}

func (r *GormMessageRepository) ListByConversation(ctx context.Context, opt ListMessagesOption) ([]model.Message, error) {
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

	query := r.db.WithContext(ctx).Model(&model.Message{}).Where("conversation_id = ?", opt.ConversationID)
	if opt.BeforeID > 0 {
		query = query.Where("id < ?", opt.BeforeID)
	}

	items := make([]model.Message, 0, limit)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *GormMessageRepository) CountByConversation(ctx context.Context, conversationID uint) (int64, error) {
	if r == nil || r.db == nil {
		return 0, ErrRepositoryNotReady
	}

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("conversation_id = ?", conversationID).
		Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *GormMessageRepository) UpdateByID(ctx context.Context, id uint, changes map[string]any) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}
	if len(changes) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&model.Message{}).Where("id = ?", id).Updates(changes).Error
}

func (r *GormMessageRepository) DeleteByID(ctx context.Context, id uint) error {
	if r == nil || r.db == nil {
		return ErrRepositoryNotReady
	}

	return r.db.WithContext(ctx).Delete(&model.Message{}, id).Error
}
