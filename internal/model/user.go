package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey;comment:用户ID" json:"id"`
	Erp          string         `gorm:"size:64;not null;uniqueIndex;comment:用户erp，登录使用" json:"erp"`
	Username     string         `gorm:"size:64;not null;uniqueIndex;comment:用户名" json:"username"`
	Nickname     string         `gorm:"size:64;not null;comment:昵称" json:"nickname"`
	AvatarURL    string         `gorm:"size:255;comment:头像URL" json:"avatarUrl"`
	Bio          string         `gorm:"size:255;comment:个人简介" json:"bio"`
	Email        *string        `gorm:"size:128;uniqueIndex;comment:邮箱地址" json:"email,omitempty"`
	Phone        *string        `gorm:"size:32;uniqueIndex;comment:手机号" json:"phone,omitempty"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Status       string         `gorm:"size:32;not null;default:active;index;comment:状态" json:"status"`
	LastSeenAt   *time.Time     `gorm:"comment:最后在线时间" json:"lastSeenAt,omitempty"`
	CreatedAt    time.Time      `gorm:"comment:创建时间" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"comment:更新时间" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:删除时间，null表示未删除" json:"-"`
}
