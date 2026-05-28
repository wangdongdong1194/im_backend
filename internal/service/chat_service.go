package service

import (
	"context"
	"errors"
	"time"

	"im_backend/internal/model"
	"im_backend/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrChatServiceNotReady     = errors.New("chat service not ready")
	ErrInvalidConversationID   = errors.New("invalid conversation id")
	ErrInvalidSenderID         = errors.New("invalid sender id")
	ErrConversationNotFound    = errors.New("conversation not found")
	ErrConversationTypeInvalid = errors.New("conversation type invalid")
	ErrSenderNotInConversation = errors.New("sender not in conversation")
)

type chatConversationRepository interface {
	GetByID(ctx context.Context, id uint) (*model.Conversation, error)
	UpdateByID(ctx context.Context, id uint, changes map[string]any) error
}

type chatConversationMemberRepository interface {
	GetByConversationAndUser(ctx context.Context, conversationID uint, userID uint) (*model.ConversationMember, error)
	ListByConversation(ctx context.Context, opt repository.ListConversationMembersOption) ([]model.ConversationMember, error)
}

type chatMessageRepository interface {
	Create(ctx context.Context, message *model.Message) error
	GetByConversationAndClientMsgID(ctx context.Context, conversationID uint, clientMsgID string) (*model.Message, error)
	ListByConversation(ctx context.Context, opt repository.ListMessagesOption) ([]model.Message, error)
}

type ChatService struct {
	conversationRepo       chatConversationRepository
	conversationMemberRepo chatConversationMemberRepository
	messageRepo            chatMessageRepository
}

const groupMemberPageSize = 500

func NewChatService(
	conversationRepo chatConversationRepository,
	conversationMemberRepo chatConversationMemberRepository,
	messageRepo chatMessageRepository,
) *ChatService {
	return &ChatService{
		conversationRepo:       conversationRepo,
		conversationMemberRepo: conversationMemberRepo,
		messageRepo:            messageRepo,
	}
}

func (s *ChatService) SendGroupMessage(
	ctx context.Context,
	conversationID uint,
	senderUserID uint,
	content string,
	clientMsgID string,
) (*model.Message, []uint, bool, error) {
	if s == nil || s.conversationRepo == nil || s.conversationMemberRepo == nil || s.messageRepo == nil {
		return nil, nil, false, ErrChatServiceNotReady
	}
	if conversationID == 0 {
		return nil, nil, false, ErrInvalidConversationID
	}
	if senderUserID == 0 {
		return nil, nil, false, ErrInvalidSenderID
	}
	if content == "" {
		return nil, nil, false, ErrInvalidMessage
	}

	conversation, err := s.conversationRepo.GetByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotReady) {
			return nil, nil, false, ErrChatServiceNotReady
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, false, ErrConversationNotFound
		}
		return nil, nil, false, err
	}

	if conversation.Type != "group" {
		return nil, nil, false, ErrConversationTypeInvalid
	}

	if _, err := s.conversationMemberRepo.GetByConversationAndUser(ctx, conversationID, senderUserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, false, ErrSenderNotInConversation
		}
		return nil, nil, false, ErrSenderNotInConversation
	}

	if clientMsgID != "" {
		existing, err := s.messageRepo.GetByConversationAndClientMsgID(ctx, conversationID, clientMsgID)
		switch {
		case err == nil:
			return existing, nil, true, nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			// continue to insert when no existing message
		case err != nil:
			return nil, nil, false, err
		}
	}

	message := &model.Message{
		ConversationID: conversationID,
		SenderID:       senderUserID,
		ClientMsgID:    clientMsgID,
		Type:           "text",
		Content:        content,
		Status:         "sent",
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, nil, false, err
	}

	now := time.Now()
	if err := s.conversationRepo.UpdateByID(ctx, conversationID, map[string]any{
		"last_message_id": message.ID,
		"last_message_at": now,
	}); err != nil {
		return nil, nil, false, err
	}

	recipients, err := s.listRecipientUserIDs(ctx, conversationID, senderUserID)
	if err != nil {
		return nil, nil, false, err
	}

	return message, recipients, false, nil
}

func (s *ChatService) listRecipientUserIDs(ctx context.Context, conversationID uint, senderUserID uint) ([]uint, error) {
	offset := 0
	recipients := make([]uint, 0)

	for {
		members, err := s.conversationMemberRepo.ListByConversation(ctx, repository.ListConversationMembersOption{
			ConversationID: conversationID,
			Offset:         offset,
			Limit:          groupMemberPageSize,
		})
		if err != nil {
			return nil, err
		}

		if len(members) == 0 {
			break
		}

		for _, member := range members {
			if member.UserID == senderUserID {
				continue
			}
			recipients = append(recipients, member.UserID)
		}

		if len(members) < groupMemberPageSize {
			break
		}

		offset += len(members)
	}

	return recipients, nil
}

func (s *ChatService) ListConversationMessages(
	ctx context.Context,
	conversationID uint,
	beforeID uint,
	offset int,
	limit int,
) ([]model.Message, error) {
	if s == nil || s.messageRepo == nil {
		return nil, ErrChatServiceNotReady
	}
	if conversationID == 0 {
		return nil, ErrInvalidConversationID
	}

	return s.messageRepo.ListByConversation(ctx, repository.ListMessagesOption{
		ConversationID: conversationID,
		BeforeID:       beforeID,
		Offset:         offset,
		Limit:          limit,
	})
}
