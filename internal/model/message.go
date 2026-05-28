package model

import (
	"time"

	"gorm.io/gorm"
)

// Message stores chat content for a conversation.
type Message struct {
	ID             uint           `gorm:"primaryKey;comment:消息ID" json:"id"`
	ConversationID uint           `gorm:"not null;index;comment:会话ID" json:"conversationId"`
	SenderID       uint           `gorm:"not null;index;comment:发送者ID" json:"senderId"`
	ClientMsgID    string         `gorm:"size:64;index;comment:客户端消息ID" json:"clientMsgId"`
	Type           string         `gorm:"size:32;not null;index;comment:消息类型" json:"type"`
	Content        string         `gorm:"type:text;not null;comment:消息内容" json:"content"`
	Metadata       string         `gorm:"type:text;comment:消息元数据" json:"metadata"`
	Status         string         `gorm:"size:16;not null;default:sent;index;comment:消息状态" json:"status"`
	EditedAt       *time.Time     `gorm:"comment:编辑时间，null表示未编辑" json:"editedAt,omitempty"`
	RecalledAt     *time.Time     `gorm:"comment:撤回时间，null表示未撤回" json:"recalledAt,omitempty"`
	CreatedAt      time.Time      `gorm:"index;comment:创建时间" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"comment:更新时间" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index;comment:删除时间，null表示未删除" json:"-"`
}
