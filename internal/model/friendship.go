package model

import (
	"time"

	"gorm.io/gorm"
)

// Friendship stores bilateral relationship state between two users.
type Friendship struct {
	ID        uint           `gorm:"primaryKey;comment:好友关系ID" json:"id"`
	UserID    uint           `gorm:"not null;index:idx_friend_pair,unique;comment:用户ID" json:"userId"`
	FriendID  uint           `gorm:"not null;index:idx_friend_pair,unique;comment:好友ID" json:"friendId"`
	Remark    string         `gorm:"size:64;comment:备注" json:"remark"`
	Status    string         `gorm:"size:16;not null;default:active;index;comment:状态" json:"status"`
	CreatedAt time.Time      `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"comment:更新时间" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间，null表示未删除" json:"-"`
}
