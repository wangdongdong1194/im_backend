package model

import (
	"time"

	"gorm.io/gorm"
)

// FriendRequest stores inbound/outbound friend requests.
type FriendRequest struct {
	ID          uint           `gorm:"primaryKey;comment:好友请求ID" json:"id"`
	FromUserID  uint           `gorm:"not null;index;comment:发送者用户ID" json:"fromUserId"`
	ToUserID    uint           `gorm:"not null;index;comment:接收者用户ID" json:"toUserId"`
	ApplyReason string         `gorm:"size:255;comment:申请理由" json:"applyReason"`
	Status      string         `gorm:"size:16;not null;default:pending;index;comment:请求状态" json:"status"`
	HandledAt   *time.Time     `gorm:"comment:处理时间，null表示未处理" json:"handledAt,omitempty"`
	CreatedAt   time.Time      `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"comment:更新时间" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index;comment:删除时间，null表示未删除" json:"-"`
}
