package model

import (
	"time"

	"gorm.io/gorm"
)

// ConversationMember stores user membership and read state in a conversation.
type ConversationMember struct {
	ID                uint           `gorm:"primaryKey;comment:成员ID" json:"id"`
	ConversationID    uint           `gorm:"not null;index:idx_conv_member,unique;comment:会话ID" json:"conversationId"`
	UserID            uint           `gorm:"not null;index:idx_conv_member,unique;index;comment:用户ID" json:"userId"`
	Role              string         `gorm:"size:16;not null;default:member;comment:成员角色" json:"role"`
	NicknameInGroup   string         `gorm:"size:64;comment:群昵称" json:"nicknameInGroup"`
	MuteUntil         *time.Time     `gorm:"comment:禁言截止时间，null表示不禁言" json:"muteUntil,omitempty"`
	LastReadMessageID *uint          `gorm:"index;comment:最后已读消息ID" json:"lastReadMessageId,omitempty"`
	LastReadAt        *time.Time     `gorm:"index;comment:最后已读时间" json:"lastReadAt,omitempty"`
	JoinedAt          time.Time      `gorm:"not null;comment:加入时间" json:"joinedAt"`
	CreatedAt         time.Time      `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt         time.Time      `gorm:"comment:更新时间" json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index;comment:删除时间" json:"-"`
}
