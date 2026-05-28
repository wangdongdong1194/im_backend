package model

import (
	"time"

	"gorm.io/gorm"
)

// Conversation represents both single chat and group chat sessions.
type Conversation struct {
	ID            uint           `gorm:"primaryKey;comment:会话ID" json:"id"`
	Type          string         `gorm:"size:16;not null;index;comment:会话类型" json:"type"`
	Name          string         `gorm:"size:128;comment:会话名称" json:"name"`
	AvatarURL     string         `gorm:"size:255;comment:会话头像URL" json:"avatarUrl"`
	OwnerID       uint           `gorm:"index;comment:会话拥有者ID" json:"ownerId"`
	LastMessageID *uint          `gorm:"index;comment:最后一条消息ID" json:"lastMessageId,omitempty"`
	LastMessageAt *time.Time     `gorm:"index;comment:最后一条消息时间" json:"lastMessageAt,omitempty"`
	CreatedAt     time.Time      `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"comment:更新时间" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index;comment:删除时间" json:"-"`
}
