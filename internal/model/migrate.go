package model

import "gorm.io/gorm"

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Conversation{},
		&ConversationMember{},
		&Message{},
		&Friendship{},
		&FriendRequest{},
	)
}
